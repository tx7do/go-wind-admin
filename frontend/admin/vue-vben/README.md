# 中后台管理系统前端

## 安装使用

- 启用Corepack

```bash
corepack enable
```

- 安装依赖

```bash
pnpm install
```

- 调试运行

```bash
pnpm dev
```

- 构建打包

```bash
pnpm build
```

- 查看ESlint

```bash
npx eslint --inspect-config
```

## API文档

- 本地文档（内嵌在服务端）： <http://localhost:7788/docs/>
- 远端文档： <https://apifox.com/apidoc/shared-fd4db0fc-f515-4423-85e5-59ad9aaa6a1a>

## Mock

https://apifoxmock.com/m1/5619700-5299226-default/admin/v1

## WebStorm相关

1. `Ctrl+Alt+L`的格式化快捷键需要手动绑定到ESlint fix。

## 环境变量

> 以下仅列出与运行时地址相关的关键项，完整配置见 `apps/admin/.env.development` / `.env.production`。

| 变量 | 说明 | 开发默认值 |
|------|------|-----------|
| `VITE_GLOB_SSE_URL` | SSE 推送地址 | `http://localhost:7789/events` |
