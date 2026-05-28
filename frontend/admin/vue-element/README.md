<div align="center">
  <img alt="go-wind-admin element-admin" width="80" height="80" src="./src/assets/images/logo.png">
  <h1>GoWind Admin - Element Plus</h1>

  <img src="https://img.shields.io/badge/Vue-3.5.30-brightgreen.svg"/>
  <img src="https://img.shields.io/badge/Vite-8.0.0-green.svg"/>
  <img src="https://img.shields.io/badge/Element_Plus-2.13.5-blue.svg"/>
  <img src="https://img.shields.io/badge/TypeScript-5.9.3-blue.svg"/>
  <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
</div>

## 项目简介

[GoWind Admin - Element Plus](https://github.com/tx7do/go-wind-admin/tree/main/frontend/admin/vue-element) 是一个基于 Vue3、Vite 8、TypeScript 和 Element Plus 构建的企业级后台管理系统，适配 Go-Wind 后端 gRPC 服务。

**核心技术栈：**

| 类别 | 技术 | 版本 |
|---|---|---|
| 框架 | Vue 3 | 3.5.30+ |
| 构建 | Vite | 8.0.0+ |
| 类型 | TypeScript | 5.9.3+ |
| UI | Element Plus | 2.13.5+ |
| 表格 | VXE Table | 4.6.25+ |
| 编辑器 | TipTap | 3.20.0+ |
| CSS | UnoCSS + SCSS | — |
| 状态 | Pinia + Vue Query | — |
| 路由 | Vue Router | 5.0+ |
| 国际化 | vue-i18n | 11.3+ |
| 图标 | Iconify (@iconify/vue) | — |

---

## 快速开始

### 环境要求

| 环境 | 版本 |
|---|---|
| Node.js | `^20.19.0` 或 `>=22.12.0` |
| 包管理器 | `pnpm >= 8.0.0` |

### 安装与启动

```bash
npm install pnpm -g                    # 安装 pnpm（如未安装）
pnpm config set registry https://registry.npmmirror.com  # 国内镜像（可选）
pnpm install                           # 安装依赖
pnpm run dev                           # 启动开发服务器 → http://localhost:3000
```

### 构建

```bash
pnpm run build                         # 类型检查 + 生产构建
pnpm run build-only                    # 仅构建（不做类型检查）
pnpm run preview                       # 预览生产构建
```

### 脚本命令

| 命令 | 说明 |
|---|---|
| `pnpm run dev` | 启动开发服务器 |
| `pnpm run build` | 类型检查 + 生产构建 |
| `pnpm run type-check` | TypeScript 类型检查 |
| `pnpm run lint` | ESLint + Prettier + Stylelint 全量检查 |
| `pnpm run lint:eslint` | ESLint 检查 |
| `pnpm run lint:prettier` | Prettier 格式化 |
| `pnpm run lint:stylelint` | Stylelint 样式检查 |
| `pnpm run commit` | 交互式规范提交 |

---

## 项目结构

```
src/
├── api/                    # API 接口层（三层架构）
│   ├── generated/          #   protobuf 自动生成（勿手动修改）
│   ├── service/            #   Service 层（gRPC Client 封装）
│   └── composables/        #   Vue Query hooks（面向组件的最终 API）
│
├── pages/                  # 页面视图
│   ├── app/                #   业务页面
│   │   ├── dashboard/      #     工作台
│   │   ├── opm/            #     组织人员管理（用户/组织/岗位）
│   │   ├── permission/     #     权限管理（权限/权限组/角色/菜单）
│   │   ├── system/         #     系统管理（API/字典/文件/任务/策略/语言）
│   │   ├── log/            #     审计日志（5类日志）
│   │   ├── internal_message/ #   内部消息
│   │   └── tenant/         #     租户管理
│   └── core/               #   框架页面
│       ├── login/          #     登录/注册
│       ├── error/          #     错误页（403/404/500）
│       └── redirect/       #     重定向
│
├── components/             # 公共组件
│   ├── Pro/                #   Pro 高级组件（ProTable/ProForm/ProPage 等）
│   ├── SvgIcon/            #   统一图标组件（Iconify + 本地 SVG）
│   ├── Editor/             #   富文本编辑器（TipTap）
│   ├── CommandPalette/     #   命令面板
│   ├── IconSelect/         #   图标选择器
│   ├── InputTag/           #   标签输入框
│   ├── LangSelect/         #   语言选择器
│   ├── NoticeDropdown/     #   通知下拉
│   ├── ThemeSwitch/        #   主题切换
│   └── AppLink/            #   外链组件
│
├── composables/            # 业务 composables
│   ├── use-auth.ts         #   认证相关
│   ├── use-dict-cache.ts   #   字典缓存
│   ├── use-is-mobile.ts    #   移动端检测
│   ├── use-token-refresh.ts #  Token 刷新
│   ├── use-table-selection.ts # 表格选择
│   ├── use-sortable.ts     #   拖拽排序
│   └── ...                 #   其他
│
├── stores/                 # Pinia 状态管理
│   └── modules/
│       ├── access.store.ts #   权限（accessCodes/accessRoutes/accessMenus）
│       ├── app-user.store.ts # 当前用户信息
│       ├── tabbar.store.ts #   标签页管理
│       ├── tags-view.store.ts # 标签页视图
│       └── lock.store.ts   #   锁屏
│
├── core/                   # 核心模块（可跨项目复用）
│   ├── access/             #   权限控制
│   ├── preferences/        #   偏好设置（主题/布局/侧边栏等）
│   ├── router/             #   路由核心（动态路由/权限过滤/菜单生成）
│   ├── storage/            #   存储管理（TTL/驱逐/多标签页同步）
│   └── transport/          #   传输层
│       ├── rest/           #     HTTP 请求（Axios + 拦截器 + 认证）
│       └── sse/            #     Server-Sent Events
│
├── layouts/                # 布局组件
│   ├── Layout.vue          #   布局入口
│   ├── LeftLayout.vue      #   侧边栏布局（sidebar-nav）
│   ├── TopLayout.vue       #   顶部布局（header-nav）
│   ├── MixLayout.vue       #   混合布局（mixed-nav）
│   └── components/         #   布局子组件（侧边栏/顶栏/标签栏/面包屑等）
│
├── router/                 # 路由业务绑定
│   ├── index.ts            #   路由实例
│   ├── guard.ts            #   路由守卫
│   ├── auth.guard.ts           #   权限路由入口
│   └── routes/modules/app/ #   业务路由模块文件
│
├── styles/                 # 全局样式
├── plugins/                # 插件（ECharts/VXE Table/Vue Query/NProgress）
├── types/                  # TypeScript 类型定义
├── utils/                  # 工具函数
├── constants/              # 常量配置
├── directives/             # 自定义指令
└── locales/                # 翻译资源（zh-CN/en-US）
```

---

## 架构概览

### 分层架构

```
┌─────────────────────────────────────────────────────┐
│  pages/                    页面视图层                │
├─────────────────────────────────────────────────────┤
│  components/  composables/ 组件 & 组合函数层         │
├─────────────────────────────────────────────────────┤
│  api/composables/          API hooks 层（Vue Query） │
│  api/service/              Service 层（gRPC Client） │
│  api/generated/            类型 & Client 工厂（自动）│
├─────────────────────────────────────────────────────┤
│  core/                     核心基础设施层            │
│    transport/              请求传输（HTTP / SSE）    │
│    preferences/            偏好设置                  │
│    router/                 路由核心                  │
│    storage/                存储管理                  │
│    access/                 权限控制                  │
├─────────────────────────────────────────────────────┤
│  stores/                   状态管理（Pinia）         │
├─────────────────────────────────────────────────────┤
│  layouts/                  布局组件                  │
├─────────────────────────────────────────────────────┤
│  locales/                  国际化翻译资源            │
└─────────────────────────────────────────────────────┘
```

### API 三层架构

```
generated/  ←── protobuf 自动生成，勿手动修改
    │
service/    ←── 一个文件 = 一个 gRPC Service 的单例封装
    │
composables/ ←── 面向组件的 Vue Query hooks
    ├── use*()    组件内使用（需要 setup 上下文）
    ├── fetch*()  组件外使用（Store、路由守卫等）
    └── 工具函数   枚举映射（状态/颜色/名称）
```

### 请求流程

```
组件 → useXxx(query)                    Vue Query hook
         ↓
      xxxService.List(params)           service 层
         ↓
      gRPC Client → requestApi()        generated 层
         ↓
      RequestClient (Axios)             transport 层
         ├── 注入 Authorization Token
         ├── 注入 X-Request-ID
         ├── 注入 Accept-Language
         └── 401 → 自动刷新 Token
```

---

## 模块文档

各核心模块有独立的使用文档，面向业务开发者：

| 模块 | 文档 | 说明 |
|---|---|---|
| API 层 | [src/api/README.md](src/api/README.md) | 三层架构、hooks 用法、新增模块步骤 |
| 路由核心 | [src/core/router/README.md](src/core/router/README.md) | 动态路由、权限模式、路由配置方法 |
| 偏好设置 | [src/core/preferences/README.md](src/core/preferences/README.md) | 主题/布局/侧边栏配置、usePreferences() |
| 存储管理 | [src/core/storage/README.md](src/core/storage/README.md) | StorageManager 用法、TTL/驱逐/批量操作 |

---

## 开发指南

### 自动导入

项目配置了 `unplugin-auto-import` 和 `unplugin-vue-components`，以下内容无需手动 import：

- **Vue API**：`ref`、`computed`、`watch`、`onMounted`、`defineComponent`、`useAttrs` 等
- **Vue Router**：`useRoute`、`useRouter` 等
- **Pinia**：`defineStore`、`storeToRefs` 等
- **vue-i18n**：`useI18n` 等
- **@vueuse/core**：`useDebounceFn`、`useClipboard` 等
- **Element Plus**：所有组件和函数（`ElMessage`、`ElMessageBox` 等）

### 新增业务页面

完整流程只需修改 3 类文件：

**1. 路由定义** — `src/router/routes/modules/app/xxx.ts`

```ts
export default {
  path: "/app/xxx",
  name: "Xxx",
  component: () => import("@/pages/app/xxx/index.vue"),
  meta: {
    title: "routes.xxx",           // i18n key
    icon: "lucide:folder",         // Iconify 图标
    authority: ["permission_code"], // 权限码
  },
} satisfies RouteRecordRaw;
```

**2. 翻译文本** — `src/locales/zh-CN/routes.json` 和 `src/locales/en-US/routes.json`

**3. 页面组件** — `src/pages/app/xxx/index.vue`

### 调用 API

```ts
// 组件内 — 使用 Vue Query hooks
import { useListUsers, useCreateUser } from "@/api";
import { PaginationQuery } from "@/core/transport/rest";

const query = new PaginationQuery({ paging: { page: 1, pageSize: 20 } });
const { data, isLoading } = useListUsers(query);

const { mutateAsync } = useCreateUser();
await mutateAsync({ data: formValues });
```

```ts
// 组件外（Store/路由守卫）— 使用 fetch* 方法
import { fetchListUsers } from "@/api";

const users = await fetchListUsers(query);
```

### 图标使用

统一使用 `SvgIcon` 组件，支持所有 Iconify 图标集：

```vue
<SvgIcon icon="lucide:settings" :size="18" />
<SvgIcon icon="ep:user" :size="20" />
<SvgIcon icon="collapse" :size="16" />  <!-- 本地 SVG -->
```

### 权限控制

```vue
<!-- 按钮级权限 -->
<el-button v-access="'user:create'">新增</el-button>
<el-button v-access="'user:delete'">删除</el-button>
```

```ts
// 编程式权限检查
import { useAccess } from "@/core/access";
const { hasAccessByCodes } = useAccess();
if (hasAccessByCodes("user:create")) { /* ... */ }
```

### 偏好设置

```ts
import { updatePreferences, usePreferences } from "@/core/preferences";

// 修改偏好
updatePreferences({ sidebar: { collapsed: true } });

// 读取偏好（组件内）
const { isDark, layout, isMobile } = usePreferences();
```

---

## 功能模块

### 核心功能

- **登录认证**：账号密码登录、验证码、Token 自动刷新、多端认证
- **个人中心**：基本信息、密码修改、消息通知
- **工作台**：数据分析、统计图表

### 组织人员管理 (OPM)

- **用户管理**：用户 CRUD、角色分配、状态控制、密码重置
- **组织架构**：部门树管理、层级维护
- **岗位管理**：岗位配置、岗位分配

### 权限管理

- **权限点管理**：按钮权限、接口权限、权限码
- **权限组管理**：权限分组、批量配置
- **角色管理**：角色 CRUD、权限绑定
- **菜单管理**：菜单配置、路由映射、图标选择

### 系统管理

- **API 管理**：接口定义、权限绑定
- **字典管理**：字典类型 + 字典条目
- **文件管理**：文件上传下载、存储管理
- **任务管理**：异步任务、执行日志
- **登录策略**：安全策略、密码策略
- **语言管理**：多语言配置

### 审计日志

- **登录日志**：登录记录、IP、状态
- **API 日志**：接口调用、响应时间
- **操作日志**：操作记录、变更追踪
- **数据访问日志**：数据查询/修改/导出
- **权限审计日志**：权限变更记录
- **策略评估日志**：策略评估记录

### 其他

- **多租户**：租户管理、资源配额（可配置开关）
- **内部消息**：消息分类、已读标记
- **国际化**：中文/英文，可扩展
- **多布局**：侧边菜单/顶部菜单/混合菜单/全屏内容
- **暗黑模式**：light/dark/auto，17 种内置主题色
- **偏好设置面板**：运行时可配置的主题/布局/快捷键

---

## 配置说明

### 环境变量

**开发环境** (`.env.development`)：

```bash
VITE_APP_PORT=3000                            # 应用端口
VITE_APP_BASE_API=/admin/v1                   # 代理前缀
VITE_APP_API_URL=http://localhost:7788        # 后端 API 地址
VITE_APP_SSE_URL=http://localhost:7789/events # SSE 推送地址
VITE_MOCK_DEV_SERVER=false                    # Mock 开关
VITE_APP_TENANT_ENABLED=false                 # 多租户开关
```

**生产环境** (`.env.production`)：

```bash
VITE_APP_API_URL=https://api.demo.admin.gowind.cloud
VITE_APP_SSE_URL=https://sse.demo.admin.gowind.cloud/events
VITE_APP_TENANT_ENABLED=true
```

### Nginx 配置示例

```nginx
server {
    listen      80;
    server_name localhost;

    location / {
        root   /usr/share/nginx/html;
        index  index.html index.htm;
    }

    location /admin/v1/ {
        proxy_pass http://localhost:7788/;
    }
}
```

---

## 提交规范

采用 [Conventional Commits](https://www.conventionalcommits.org/)，通过 `pnpm run commit` 交互式提交。

格式：`<type>(<scope>): <subject>`

| Type | 说明 |
|---|---|
| `feat` | 新功能 |
| `fix` | 修复 bug |
| `docs` | 文档变更 |
| `style` | 代码格式调整 |
| `refactor` | 重构 |
| `test` | 测试 |
| `chore` | 构建/工具链 |

---

## 常见问题

| 问题 | 解决方案 |
|---|---|
| 启动报错或依赖问题 | `rm -rf node_modules pnpm-lock.yaml && pnpm install` |
| 浏览器访问空白 | 升级浏览器至最新版本 |
| IDE 爆红但能正常运行 | 重启 VSCode，等待 TypeScript 服务初始化 |
| Mock 数据不生效 | 检查 `.env.development` 中 `VITE_MOCK_DEV_SERVER=true` |
| 接口请求失败 | 检查 `VITE_APP_API_URL`，确保后端服务已启动 |
| gRPC 类型报错 | 重新生成 protobuf 代码，确认 `generated/` 目录最新 |

---

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`pnpm run commit`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 特别感谢

- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架
- [Vite](https://vitejs.dev/) - 下一代前端构建工具
- [Element Plus](https://element-plus.org/) - Vue 3 组件库
- [vue3-element-admin](https://gitee.com/panjiachen/vue-element-admin) - 灵感来源

---

**如果这个项目对你有帮助，请给一个 Star 支持！**
