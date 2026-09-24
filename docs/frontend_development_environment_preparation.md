# 如何搭建前端开发环境

## 安装开发工具

需要安装的软件有：

- [Git](https://git-scm.com/)
- [Visual Studio Code](https://code.visualstudio.com/)
- [WebStorm](https://www.jetbrains.com/webstorm/)
- [Node.js](https://nodejs.org/)
- [pnpm](https://pnpm.io/)（本仓三端唯一使用的包管理器；未随系统装好时用 corepack 启用，见下文）

### Windows

Windows下安装软件的方法有很多种，这里推荐使用软件包管理工具：[scoop](https://scoop.sh/)。

```shell
scoop bucket add extras
scoop install git vscode webstorm nodejs pnpm
```

### MacOS

MacOS下安装软件的方法有很多种，这里推荐使用软件包管理工具：[Homebrew](https://brew.sh/)。

```shell
brew install git node pnpm
brew install --cask visual-studio-code webstorm
```

## 启用 corepack（拿到 pnpm）

三端的 `pnpm-lock.yaml` 表明本仓统一用 pnpm 安装依赖。pnpm 由 Node.js 自带的 corepack 提供，
不需要 `npm install -g pnpm`（那会绕过版本路由，vue-vben 用 `packageManager` 钉死了
`pnpm@11.18.0`）：

```shell
corepack enable
```

## 前端没有要装的代码生成插件

三端的 API 客户端 TypeScript **不是**由 `ts-proto` 之类的 npm 包生成的，因此无需 `npm install -g ts-proto`。
生成链路属于后端 buf 流程：三份 buf 配置把 TS 直接写到各端源码目录——

| 端 | buf 模板（`backend/api/`） | 输出 |
|---|---|---|
| react | `buf.react.admin.typescript.gen.yaml` | `frontend/admin/react/src/api/generated/` |
| vue-element | `buf.vue-element.admin.typescript.gen.yaml` | `frontend/admin/vue-element/src/api/generated/` |
| vue-vben | `buf.vue-vben.admin.typescript.gen.yaml` | `frontend/admin/vue-vben/apps/admin/src/api/generated/` |

三份模板用的都是同一个 buf 本地插件 `protoc-gen-typescript-http`，它由 `make plugin` 安装
（见 [如何搭建后端开发环境](./backend_development_environment_preparation.md)）。生成三端 TS 的命令是
在 `backend/` 目录执行：

```shell
make ts
```

产物是仓库里的普通源文件：改了 `.proto` 后由跑代码生成的一方执行 `make ts` 并提交，其余人拉下来即可用。

## 装完之后：跑起来

前后端都装好后，最短路径三条命令（完整版见
[教程 02 · 从零跑起来](./tutorial/02-get-it-running.md)与
[Windows 本地开发启动指南](./windows-startup-guide.md)）：

```shell
# 1. 起依赖服务（PostgreSQL / Redis / MinIO）
cd backend && docker compose -f docker-compose.libs.yaml up -d

# 2. 起后端（HTTP :7788）
gow run admin

# 3. 起前端——三选一，以 react 端为例
cd ../frontend/admin/react && pnpm install && pnpm dev
```

## npm/pnpm/yarn切换源

> 本仓三端只用 pnpm——下面 **pnpm** 一节是要用的，npm / nrm / yarn / yrm 各节是通用换源备忘，
> 本项目不使用 yarn。镜像可用性随时间变化，**以 `pnpm config get registry` 与实测为准**。

* 国内镜像

| 提供商  | 搜索地址                   | registry地址                                         |
|------|------------------------|----------------------------------------------------|
| 淘宝   | https://npmmirror.com/ | https://registry.npmmirror.com                     |
| 腾讯云  |                        | http://mirrors.cloud.tencent.com/npm/              |
| 华为云  |                        | https://mirrors.huaweicloud.com/repository/npm     |
| 浙江大学 |                        | http://mirrors.zju.edu.cn/npm/（2026-09 实测已 404，勿用） |
| 南京邮电 |                        | https://mirrors.njupt.edu.cn/nexus/repository/npm/ |

### npm（通用备忘——本仓不用 npm 装依赖）

```shell
# 查看源
npm get registry
npm config get registry

# 临时修改
npm --registry https://registry.npmmirror.com install any-touch

# 永久修改
npm config set registry https://registry.npmmirror.com

# 还原
npm config set registry https://registry.npmjs.org
```

### nrm（通用备忘）

```shell
# 安装 nrm
npm install -g nrm

# 列出当前可用的所有镜像源
nrm ls

# 使用淘宝镜像源
nrm use taobao

# 测试访问速度
nrm test taobao
```

### pnpm（本仓使用）

```shell
# 查看源
pnpm get registry
pnpm config get registry

# 临时修改
pnpm --registry https://registry.npmmirror.com install any-touch

# 永久修改
pnpm config set registry https://registry.npmmirror.com

# 还原
pnpm config set registry https://registry.npmjs.org
```

### yarn（通用备忘，本项目不使用 yarn）

```shell
# 查看源
yarn config get registry

# 临时修改
yarn add any-touch@latest --registry=https://registry.npmjs.org/

# 永久修改
yarn config set registry https://registry.npmmirror.com/

# 还原
yarn config set registry https://registry.yarnpkg.com
```

### yrm（通用备忘）

```shell
# 安装 yrm
npm install -g yrm

# 列出当前可用的所有镜像源
yrm ls

# 使用淘宝镜像源
yrm use taobao

# 测试访问速度
yrm test taobao
```
