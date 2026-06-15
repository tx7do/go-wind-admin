---
name: scaffold-dev-guide
description: Vue3 中后台脚手架项目二次开发指南。指导开发者按照项目架构规范创建新业务模块，包括 API 两层架构（generated/client + composables）、路由配置、VxeTable + VbenForm 页面开发、i18n 国际化、Pinia Store 等完整工作流。当用户要求新建模块、新增页面、添加 API、创建 CRUD 功能、或询问项目开发规范时使用此 Skill。
---

# 脚手架项目二次开发指南

## 项目概览

基于 Vue 3 + Vite + TypeScript 的中后台管理系统脚手架，面向二开场景。

**核心技术栈**: Vue 3.5, Ant Design Vue 4.2, Tailwind CSS, Shadcn-ui, Pinia, Vue Router, Vue Query (TanStack Query), VxeTable, i18n, Axios

**应用入口**: `apps/admin/src/`

## Vben 框架核心机制

本项目基于 Vben Admin 配置型框架，**通过配置而非编码来完成大部分开发工作**。理解以下机制至关重要：

### 组件注册机制（不要绕过）

框架有 **两套组件注册体系**，分别用于不同场景：

**1. VbenForm 组件（schema 中使用）**

定义在 `adapter/component/index.ts`，通过 `globalShareState.setComponents()` 注册。在 `schema` 的 `component` 字段中 **只能使用这些注册过的名称**：

```
Input, InputNumber, InputPassword, Select, ApiSelect, TreeSelect,
ApiTreeSelect, RadioGroup, Checkbox, CheckboxGroup, Switch, DatePicker,
RangePicker, TimePicker, Textarea, Upload, Editor, IconPicker, AutoComplete,
Mentions, Rate, Divider, Space, DefaultButton, PrimaryButton, ApiTree
```

**2. Template 全局组件（template 中使用）**

定义在 `registerGlobComp.ts`，通过 `app.use()` 全局注册。在 template 中 **直接用 `a-*` 前缀**：

```
<a-button>, <a-tag>, <a-popconfirm>, <a-input>, <a-select>,
<a-tree>, <a-table>, <a-dropdown>, <a-menu>, <a-card>, <a-space>,
<a-switch>, <a-tabs>, <a-divider>, <a-layout>
```

> **禁止**: `import { Tag, Button } from 'ant-design-vue'` 然后在 template 中用 `<Tag>` 或 `<Button>`。

### 配置驱动模式（不要自己写逻辑）

| 场景             | 配置方式                                                 | 不要做                      |
|----------------|------------------------------------------------------|----------------------------|
| 表格列日期格式化    | `formatter: 'formatDateTime'`                        | Slot 中用 dayjs 格式化          |
| 表单必填校验       | `rules: 'required'` / `'selectRequired'`             | 用 Zod 或自定义校验函数             |
| 列表数据加载       | `proxyConfig.ajax.query`                             | 手动 watch + ref + async function |
| 分页             | `pagerConfig: {}`                                    | 手动管理分页状态                  |
| 表格刷新           | `gridApi.reload()`                                   | 手动重新请求数据                  |
| 表单赋值/取值      | `baseFormApi.setValues()` / `baseFormApi.getValues()` | 直接操作 DOM 或 ref              |
| 表单校验          | `baseFormApi.validate()`                             | 手动检查每个字段                  |

## 目录结构速查

```
apps/admin/src/
├── api/                  # API 层（两层架构）
│   ├── generated/        # ← protobuf 自动生成，禁止手动编辑
│   ├── client.ts         # ← ApiClient 单例（ClientTransport 适配器）
│   └── composables/      # ← Vue Query hooks 层：use*/fetch*/枚举工具
├── adapter/              # VbenForm + VxeTable 适配器配置
├── router/routes/modules/# ← 路由模块（按功能拆分）
├── stores/               # Pinia 状态管理
├── views/app/            # 业务页面（按功能模块组织）
├── locales/langs/        # i18n 国际化文件
│   ├── zh-CN/            # enum.json, menu.json, page.json, ui.json
│   └── en-US/
└── transport/rest/       # HTTP 传输层（PaginationQuery, requestApi）
```

## 关键约定（必须遵守）

### 数据层约定

1. **禁止直接引用 `#/api/generated/` 路径** — 业务层通过 `#/api` 统一入口导入
2. **composables 直接使用 `apiClient`** — 导入 `apiClient` from `#/api/client`，调用 `apiClient.xxxService.Method()`
3. **组件内用 `use*` hooks，组件外（Store/路由守卫）用 `fetch*` 函数**
4. **更新操作只传变化字段** — `useUpdate*` 内部自动生成 `updateMask`
5. **Pinia Store 中不可依赖 `useRouter()`** — Store 初始化时路由可能未就绪
6. **所有列表查询统一使用 `PaginationQuery`**

### Vben 框架强规约

7. **表单组件必须使用注册名** — `schema` 中 `component` 字段只能使用 `adapter/component/index.ts` 中注册的名称（如 `Input`、`Select`、`ApiSelect`），不要使用 Ant Design Vue 原始组件名（如 `AInput`、`ASelect`）或原生 HTML 标签
8. **Template 中使用 `a-*` 前缀** — Ant Design Vue 组件已全局注册，在 template 中直接用 `<a-button>`、`<a-tag>`、`<a-popconfirm>` 等，不需要 `import { Tag } from 'ant-design-vue'`
9. **图标在 `:icon` prop 中必须用 `h()` 渲染** — `:icon="h(LucideFilePenLine)"`，图标组件从 `@vben/icons` 导入
10. **日期列用 `formatter: 'formatDateTime'`** — 不要在 Slot 中用 dayjs 手动格式化
11. **表单校验用内置规则名** — `rules: 'required'` 或 `rules: 'selectRequired'`，不要用 Zod 表达式
12. **删除操作必须二次确认** — 使用 `<a-popconfirm>`，不要用 `window.confirm` 或直接删除
13. **国际化文本不硬编码** — 使用 `$t()` 引用 locales 文件中的 key
14. **使用严格相等运算符 `===`** — 禁止使用 `==`

## 新建业务模块 Checklist

二开时创建新模块的标准流程：

```
- [ ] Step 1: 确认 generated 层已有类型（protobuf 已生成）
- [ ] Step 2: 创建 composables 层（src/api/composables/xxx.ts）
- [ ] Step 3: 注册导出（composables/index.ts）
- [ ] Step 4: 添加 i18n 翻译（zh-CN + en-US）
- [ ] Step 5: 创建路由模块（router/routes/modules/app/xxx.ts）
- [ ] Step 6: 创建视图页面（views/app/xxx/）
```

### Step 1: 确认 generated 类型

确保 `src/api/generated/admin/service/v1` 中包含目标模块的：
- 类型定义: `xxxservicev1_EntityName`
- 请求/响应类型
- ApiClient 中已有 `xxxService` getter

### Step 2-3: API 层开发

详见 [api-patterns.md](api-patterns.md)，包含完整的 composable 模板代码。

### Step 4: i18n 国际化

在四个 JSON 文件中添加对应 key：

| 文件                | 用途       | key 前缀示例                         |
|-------------------|----------|----------------------------------|
| `zh-CN/enum.json` | 枚举翻译     | `"enum.xxx.status.NORMAL": "正常"` |
| `zh-CN/menu.json` | 菜单名称     | `"menu.xxx.moduleName": "模块名"`   |
| `zh-CN/page.json` | 页面文本     | `"page.xxx.name": "名称"`          |
| `zh-CN/ui.json`   | 通用 UI 文本 | 已有通用 key，无需修改                    |

`en-US/` 目录下同步添加英文翻译。

### Step 5: 路由配置

新建 `router/routes/modules/app/xxx.ts`：

```typescript
import type { RouteRecordRaw } from 'vue-router';
import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    path: '/xxx',
    name: 'XxxManagement',
    component: BasicLayout,
    redirect: '/xxx/list',
    meta: {
      order: 3001,           // 菜单排序
      icon: 'lucide:xxx',    // lucide 图标
      title: $t('menu.xxx.moduleName'),
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'list',
        name: 'XxxList',
        meta: {
          title: $t('menu.xxx.list'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/xxx/index.vue'),
      },
    ],
  },
];
export default routes;
```

### Step 6: 视图页面开发

详见 [view-patterns.md](view-patterns.md)，包含列表页、Drawer 表单页的完整模板。

## 导入路径约定

```typescript
// API 导入 — 统一通过 #/api 入口
import { useListUsers, PaginationQuery, userStatusToName } from '#/api';

// 类型导入 — 也通过 #/api（re-export 了 generated 类型）
import { type identityservicev1_User as User } from '#/api';

// 适配器导入
import { useVbenForm } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';

// 布局与通用组件
import { Page, useVbenDrawer } from '@vben/common-ui';

// 国际化
import { $t } from '@vben/locales';

// 图标（lucide）
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';
```

## 参考文件

- API 两层架构详细模板和模式: [api-patterns.md](api-patterns.md)
- 视图/页面/表单/表格详细模板: [view-patterns.md](view-patterns.md)
