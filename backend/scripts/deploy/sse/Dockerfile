FROM nginx:stable-alpine

# SSE 反向代理网关：无源码构建，仅承载 nginx 配置。
# TLS 终止由外层负载均衡负责，与前端静态 nginx 部署一致。
COPY scripts/deploy/sse/nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080

# start nginx
CMD ["nginx", "-g", "daemon off;"]
