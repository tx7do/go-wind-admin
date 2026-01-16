##################################
# 第一阶段：构建GO可执行文件
##################################

# 使用官方的 Go 基础镜像作为构建环境
FROM golang:1.25.4 AS builder

ARG SERVICE_NAME=app
ARG APP_VERSION=1.0.0

# 设置工作目录
WORKDIR /src

# 复制项目源代码到工作目录
COPY . /src

# 进入到服务目录
RUN cd /src/app/$SERVICE_NAME/service
# 创建二进制文件目录
RUN mkdir -p /src/app/$SERVICE_NAME/service/bin

# 下载依赖，在中国国内，请使用国内代理。有可能会出现下载失败的情况，多试几次即可。
RUN GOPROXY=https://goproxy.cn go mod download
# 编译可执行文件
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -ldflags "-X main.version=$APP_VERSION" -o /src/app/$SERVICE_NAME/service/bin/ ./...

##################################
# 第二阶段：创建最终的运行时镜像
##################################

# 使用 Alpine 作为基础镜像，因为它非常轻量级
FROM alpine:3.18

ARG SERVICE_NAME=app

# 安装必要的证书（如果应用程序需要进行 HTTPS 请求）
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /app

# 从第一阶段的构建结果中复制可执行文件到当前工作目录
COPY --from=builder /src/app/$SERVICE_NAME/service/bin/ /app/bin

# 拷贝配置文件
COPY --from=builder /src/app/$SERVICE_NAME/service/configs/ /app/configs

# 创建一个名为 appuser 的非 root 用户
RUN adduser -D appuser

# 切换到非特权用户
USER appuser:appuser

# 暴露服务端口，根据你的实际服务端口进行修改

# 设置容器启动时执行的命令
CMD ["/app/bin/server", "-c", "/app/configs"]
