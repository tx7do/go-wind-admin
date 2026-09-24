# 如何搭建后端开发环境

## 安装开发工具

需要安装的软件有：

- [Git](https://git-scm.com/)
- [Visual Studio Code](https://code.visualstudio.com/)
- [Goland](https://www.jetbrains.com/go/)
- [Docker](https://www.docker.com/)
- [Go](https://go.dev/)
- [protobuf-compiler](https://grpc.io/docs/protoc-installation/)
- [Make](https://www.gnu.org/software/make/)
- [Buf](https://buf.build/)
- [gawk](https://www.gnu.org/software/gawk/)
- [grep](https://www.gnu.org/software/grep/)
- [sed](https://www.gnu.org/software/sed/)

### Windows

Windows下安装软件的方法有很多种，这里推荐使用软件包管理工具：[scoop](https://scoop.sh/)。

一键安装所有的开发软件：

```shell
scoop bucket add extras
scoop install git vscode goland docker go protobuf make buf gawk grep sed
```

### MacOS

MacOS下安装软件的方法有很多种，这里推荐使用软件包管理工具：[Homebrew](https://brew.sh/)。

```shell
brew install git docker go protobuf make buf gawk grep gnu-sed
brew install --cask visual-studio-code goland
```

## 安装插件

后端需要的插件主要是 Protobuf 的插件。**唯一权威清单是 `backend/Makefile` 的 `plugin` 目标**，
本文档只是索引——新增/下线插件改那个目标即可，不要在这里加第 N 条 `go install`：

- [protoc-gen-go](https://google.golang.org/protobuf/cmd/protoc-gen-go)
- [protoc-gen-go-grpc](https://google.golang.org/grpc/cmd/protoc-gen-go-grpc)
- [protoc-gen-go-http](https://github.com/go-kratos/kratos/cmd/protoc-gen-go-http)
- [protoc-gen-go-errors](https://github.com/go-kratos/kratos/cmd/protoc-gen-go-errors)
- [protoc-gen-openapi](https://github.com/google/gnostic/cmd/protoc-gen-openapi)
- [protoc-gen-validate](https://github.com/envoyproxy/protoc-gen-validate)
- [protoc-gen-go-redact](https://github.com/tx7do/go-wind-toolkit)（生成脱敏字段）
- [protoc-gen-typescript-http](https://github.com/tx7do/go-wind-toolkit)（**三端 TS 客户端就靠它**，`make ts` 用）

一条命令全装（**注意目录：仓库根目录没有 Makefile，须在 `backend/` 下执行**）：

```shell
cd backend
make plugin
```

等价的手动安装（与 `plugin` 目标内容一致）：

```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
go install github.com/envoyproxy/protoc-gen-validate@latest
go install github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact@latest
go install github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http@latest
```

CLI 工具（buf、ent、gow、golangci-lint 等）在另一个目标里，`make init` = `plugin` + `cli`，
首次搭建直接跑 `make init` 即可（同样在 `backend/` 下）。

## 装完之后：跑起来

代理与插件都配好后，最短路径两条命令（完整版见
[教程 02 · 从零跑起来](./tutorial/02-get-it-running.md)、
[Windows 本地开发启动指南](./windows-startup-guide.md)与
[后端项目部署](./backend_deploy.md)）：

```shell
cd backend
docker compose -f docker-compose.libs.yaml up -d   # 起 PostgreSQL / Redis / MinIO
gow run admin                                      # 起后端，HTTP :7788
```

前端侧的三端命令见 [如何搭建前端开发环境](./frontend_development_environment_preparation.md)。

## Golang设置网络代理

### 打开模块支持

```shell
go env -w GO111MODULE=on
```

### 取消代理

```shell
go env -w GOPROXY=direct
```

### 取消校验

```shell
go env -w GOSUMDB=off
```

### 设置不走 proxy 的私有仓库或组，多个用逗号相隔（可选）

```shell
go env -w GOPRIVATE=git.mycompany.com,github.com/my/private
```

### 设置代理

#### 国内常用代理列表

> 可用性会变，**以下单实测为准**（2026-09-25 各拉一次 `.../gin-gonic/gin/@v/list`：
> 官方全球代理、七牛云、阿里云、goproxy.io、百度均 200；GoCenter 域名已无法访问，故从下表删除）。

| 提供者      | 地址                                  |
|----------|-------------------------------------|
| 官方全球代理   | https://proxy.golang.com.cn         |
| 七牛云      | https://goproxy.cn                  |
| 阿里云      | https://mirrors.aliyun.com/goproxy/ |
| 百度       | https://goproxy.bj.bcebos.com/      |

**“direct”** 为特殊指示符，用于指示 Go 回源到模块版本的源地址去抓取(比如 GitHub 等)，当值列表中上一个 Go module proxy 返回
404 或 410 错误时，Go 自动尝试列表中的下一个，遇见 **“direct”** 时回源，遇见 EOF 时终止并抛出类似 “invalid version: unknown
revision...” 的错误。

#### 官方全球代理

```shell
go env -w GOPROXY=https://proxy.golang.com.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
```

或者

```shell
go env -w GOPROXY=https://goproxy.io,direct
go env -w GOSUMDB=gosum.io+ce6e7565+AY5qEHUk/qmHc5btzW45JVoENfazw8LielDsaI+lEbq6
```

#### 七牛云

```shell
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.org
```

> `GOSUMDB` 的合法取值只有三种形态：`off`、已知校验库名（如 `sum.golang.org`、`sum.golang.google.cn`）、
> 或 `库名+Base64公钥`（可再跟一个 URL 字段）。旧文档里常见的
> `GOSUMDB=goproxy.cn/sumdb/sum.golang.org` **不是合法取值**——Go 会把第一个字段当公钥名解析并直接报
> `invalid GOSUMDB`（见 `$GOROOT/src/cmd/go/internal/modfetch/sumdb.go` 的 `dbDial`）。
> `sum.golang.org` 本就是 Go 的默认值，通常无需改动；离线/内网环境可 `GOSUMDB=off`
> （关闭校验，自担风险）。

#### 阿里云

```shell
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
# GOSUMDB 不支持
```

#### 百度

```shell
go env -w GOPROXY=https://goproxy.bj.bcebos.com/,direct
# 不支持 GOSUMDB
```

### warning: go env -w GOPROXY=... does not override conflicting OS environment variable

**原因：**

之前安装go的时候，用环境变量的方式设置过代理地址，go13提供了-w参数来设置GOPROXY变量，但无法覆盖OS级别的环境变量

**解决方法：**

```bash
unset GOPROXY

# or 

Clear-Variable GOPROXY
```

## IDE 插件安装

后端开发需要安装Buf的插件，否则，通过Buf引用的第三方proto，IDE会无法解析。

- [VSC](https://marketplace.visualstudio.com/items?itemName=bufbuild.vscode-buf)
- [Goland](https://plugins.jetbrains.com/plugin/19147-buf-for-protocol-buffers)
