# AGENTS.md — 脚手架项目二次开发指南 (Vue3 + Vben Admin)

本文件定义此项目的编码约定和架构规范，供 AI Agent 在生成和修改代码时遵循。

## 项目概览

基于 Vue 3 + Vite + TypeScript 的中后台管理系统脚手架（Vben Admin 配置型框架），面向二开场景。**通过配置而非编码来完成大部分开发工作。**

**核心技术栈**: Vue 3.5, Ant Design Vue 4.2, Tailwind CSS, Shadcn-ui, Pinia, Vue Router, Vue Query (TanStack Query), VxeTable, i18n, Axios

**应用入口**: `apps/admin/src/`

## Vben 框架核心机制

### 组件注册机制（不要绕过）

框架有 **两套组件注册体系**：

**1. VbenForm 组件（schema 中使用）** — 定义在 `adapter/component/index.ts`，通过 `globalShareState.setComponents()` 注册。`schema` 的 `component` 字段 **只能使用这些注册过的名称**：

```
Input, InputNumber, InputPassword, Select, ApiSelect, TreeSelect,
ApiTreeSelect, RadioGroup, Checkbox, CheckboxGroup, Switch, DatePicker,
RangePicker, TimePicker, Textarea, Upload, Editor, IconPicker, AutoComplete,
Mentions, Rate, Divider, Space, DefaultButton, PrimaryButton, ApiTree
```

**2. Template 全局组件（template 中使用）** — 定义在 `registerGlobComp.ts`，通过 `app.use()` 全局注册。在 template 中 **直接用 `a-*` 前缀**：

```
<a-button>, <a-tag>, <a-popconfirm>, <a-input>, <a-select>,
<a-tree>, <a-table>, <a-dropdown>, <a-menu>, <a-card>, <a-space>,
<a-switch>, <a-tabs>, <a-divider>, <a-layout>
```

> **禁止**: `import { Tag, Button } from 'ant-design-vue'` 然后在 template 中用 `<Tag>` 或 `<Button>`。

### 配置驱动模式（不要自己写逻辑）

| 场景 | 配置方式 | 不要做 |
|---|---|---|
| 表格列日期格式化 | `formatter: 'formatDateTime'` | Slot 中用 dayjs 格式化 |
| 表单必填校验 | `rules: 'required'` / `'selectRequired'` | 用 Zod 或自定义校验函数 |
| 列表数据加载 | `proxyConfig.ajax.query` | 手动 watch + ref + async function |
| 分页 | `pagerConfig: {}` | 手动管理分页状态 |
| 表格刷新 | `gridApi.reload()` | 手动重新请求数据 |
| 表单赋值/取值 | `baseFormApi.setValues()` / `baseFormApi.getValues()` | 直接操作 DOM 或 ref |
| 表单校验 | `baseFormApi.validate()` | 手动检查每个字段 |

## 目录结构

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
├── locales/langs/        # i18n 国际化文件（zh-CN/en-US: enum.json, menu.json, page.json, ui.json）
└── transport/rest/       # HTTP 传输层（PaginationQuery, requestApi）
```

## 导入路径约定

```typescript
// API 导入 — 统一通过 #/api 入口
import { useListUsers, PaginationQuery, userStatusToName } from '#/api';
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

// 消息提示
import { notification } from 'ant-design-vue';
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

7. **表单组件必须使用注册名** — `schema` 中 `component` 只能用 `adapter/component/index.ts` 中注册的名称（如 `Input`、`Select`、`ApiSelect`），不要用 `AInput`、`ASelect` 或原生 HTML 标签
8. **Template 中使用 `a-*` 前缀** — Ant Design Vue 组件已全局注册，直接用 `<a-button>`、`<a-tag>`、`<a-popconfirm>` 等
9. **图标在 `:icon` prop 中必须用 `h()` 渲染** — `:icon="h(LucideFilePenLine)"`，图标从 `@vben/icons` 导入
10. **日期列用 `formatter: 'formatDateTime'`** — 不要在 Slot 中用 dayjs 手动格式化
11. **表单校验用内置规则名** — `rules: 'required'` 或 `rules: 'selectRequired'`，不要用 Zod 表达式
12. **删除操作必须二次确认** — 使用 `<a-popconfirm>`，不要用 `window.confirm` 或直接删除
13. **页面必须用 `<Page auto-content-height>` 包裹** — 不要用 `<div>` 替代
14. **消息提示用 `notification`** — 从 `ant-design-vue` 导入，不要用 `alert()` 或 `ElMessage`
15. **国际化文本不硬编码** — 使用 `$t()` 引用 locales 文件中的 key
16. **使用严格相等运算符 `===`** — 禁止使用 `==`
17. **错误处理铁律（不吞错）** — 任何 catch 至少二选一：`console.error/warn` 带出**原始错误对象**，或重新抛出。`notification` 用户提示不等于日志（只有翻译文案，排查靠控制台原始错误）。合法裸 catch 仅限纯本地 best-effort 兜底且注释写明原因。历史教训：静默吞错曾让认证链路 bug 排查数日

---

## API 两层架构

```
generated/  +  client.ts  →  composables/  →  views/stores
(自动生成)     (ApiClient 单例)   (Vue Query hooks)
```

**依赖方向**: `views/stores → composables → apiClient.xxxService → transport.unary → requestApi`

### client.ts — ApiClient 单例

生成的 `ApiClient` 通过 `ClientTransport` 接口发送请求，`client.ts` 将 `requestApi` 适配为 `ClientTransport`（保留 token 注入、错误拦截、自动刷新等逻辑）：

```typescript
import { type ClientTransport, createApiClient } from '#/api/generated/admin/service/v1';
import { requestApi } from '#/transport/rest';

const transport: ClientTransport = {
  unary(path, method, body, _meta) { return requestApi({ body, method, path }); },
  serverStream(path, _meta) { throw new Error(`serverStream not supported: ${path}`); },
  duplexStream(path, _meta) { throw new Error(`duplexStream not supported: ${path}`); },
};
export const apiClient = createApiClient(transport);
```

ApiClient 提供的 Service Client：`apiClient.userService`、`apiClient.roleService`、`apiClient.authenticationService`、`apiClient.menuService`、`apiClient.positionService`、`apiClient.orgUnitService` 等（protobuf 重新生成后自动包含新 getter）。

### composables 层模板 (`src/api/composables/xxx.ts`)

```typescript
import type {
  xxxservicev1_GetXxxRequest,
  xxxservicev1_ListXxxResponse,
  xxxservicev1_Xxx,
  xxxservicev1_Xxx_Status as Xxx_Status,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';
import { i18n } from '@vben/locales';
import { useMutation, type UseMutationOptions, useQuery, type UseQueryOptions } from '@tanstack/vue-query';
import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// 列表 — 组件内 hook
export function useListXxxs(query: PaginationQuery, options?: UseQueryOptions<xxxservicev1_ListXxxResponse, Error>) {
  return useQuery({ queryKey: ['listXxxs', query], queryFn: () => apiClient.xxxService.List(query.toRawParams()), ...options });
}

// 列表 — Store / 外部调用
export async function fetchListXxxs(params: PaginationQuery) {
  return queryClient.fetchQuery({ queryKey: ['listXxxs', params], queryFn: () => apiClient.xxxService.List(params.toRawParams()), retry: 0 });
}

// 详情 — 组件内 hook
export function useGetXxx(req: xxxservicev1_GetXxxRequest, options?: UseQueryOptions<xxxservicev1_Xxx, Error>) {
  return useQuery({ queryKey: ['getXxx', req], queryFn: () => apiClient.xxxService.Get(req), ...options });
}

// 创建
export function useCreateXxx(options?: UseMutationOptions<object, Error, { data: xxxservicev1_Xxx }>) {
  return useMutation({ mutationFn: ({ data }) => apiClient.xxxService.Create({ data }), ...options });
}

// 更新（自动生成 updateMask）
export function useUpdateXxx(options?: UseMutationOptions<object, Error, { id: number; values: Record<string, any> }>) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.xxxService.Update({ id, data: { ...values } as any, updateMask: makeUpdateMask(Object.keys(values ?? {})) }),
    ...options,
  });
}

// 删除
export function useDeleteXxx(options?: UseMutationOptions<object, Error, number>) {
  return useMutation({ mutationFn: (id) => apiClient.xxxService.Delete({ id }), ...options });
}

// 枚举与工具函数
export const xxxStatusList = computed(() => [
  { value: 'ON', label: t('enum.xxx.status.ON') },
  { value: 'OFF', label: t('enum.xxx.status.OFF') },
]);
const XXX_STATUS_COLOR_MAP: Record<string, string> = { ON: '#52C41A', OFF: '#909399', DEFAULT: '#86909C' };
export function xxxStatusToColor(status: Xxx_Status) {
  return XXX_STATUS_COLOR_MAP[status as string] ?? XXX_STATUS_COLOR_MAP.DEFAULT ?? '#86909C';
}
export function xxxStatusToName(status?: Xxx_Status) {
  const map: Record<string, string> = { ON: t('enum.xxx.status.ON'), OFF: t('enum.xxx.status.OFF') };
  return map[status as string] ?? '';
}
```

注册导出 — 在 `src/api/composables/index.ts` 添加 `export * from './xxx';`

### Query Key 命名约定

| 操作 | Query Key 格式 | 示例 |
|---|---|---|
| 列表 | `["list{Entity}s", query]` | `["listUsers", query]` |
| 详情 | `["get{Entity}", req]` | `["getUser", { id: 1 }]` |

### PaginationQuery 使用模式

```typescript
new PaginationQuery({ paging: { page: 1, pageSize: 20 } });                           // 基础分页
new PaginationQuery({ paging: { page: 1, pageSize: 20 }, formValues: { status: 'ON' } }); // 带搜索
new PaginationQuery({ orderBy: ['-created_at', 'name'] });                              // 带排序
new PaginationQuery({ formValues: { status: 'ON' } });                                  // 不分页
new PaginationQuery({ fieldMask: 'id,name,status' });                                   // 指定字段
```

---

## 视图/页面开发

### 标准模块文件结构

```
views/app/{module}/
├── index.vue              # 列表页（VxeTable + 搜索表单）
└── {entity}-drawer.vue    # 新建/编辑 Drawer 表单
```

### 核心概念：Grid 的两套 Schema

| 配置项 | 作用 | 位置 |
|---|---|---|
| `formOptions.schema` | **搜索表单**（表格上方的筛选区域） | 使用 `#/adapter/form` 的 VbenForm 组件 |
| `gridOptions.columns` | **表格列**（数据展示区域） | 使用 VxeTable 的列配置 |

两者通过 `useVbenVxeGrid({ gridOptions, formOptions })` 合并，搜索表单的值自动传入 `proxyConfig.ajax.query` 的 `formValues`。

### 搜索表单可用组件

| component | 用途 | 关键 props |
|---|---|---|
| `Input` | 文本输入 | `placeholder`, `allowClear` |
| `InputNumber` | 数字输入 | `placeholder`, `allowClear`, `defaultValue` |
| `Select` | 下拉选择 | `options`, `showSearch`, `allowClear`, `filterOption` |
| `ApiSelect` | 远程下拉 | `api`, `afterFetch`, `showSearch`, `allowClear` |
| `ApiTreeSelect` | 远程树选择 | `api`, `treeDefaultExpandAll`, `labelField`, `valueField`, `numberToString` |
| `RadioGroup` | 单选按钮组 | `optionType: 'button'`, `buttonStyle: 'solid'`, `options` |
| `RangePicker` | 日期范围 | `showTime`, `allowClear`, `presets` |
| `DatePicker` | 日期选择 | `placeholder`, `allowClear` |
| `Switch` | 开关 | — |
| `Textarea` | 多行文本 | `placeholder`, `allowClear` |

> `ApiSelect` 的 `afterFetch` 必须将数据转为 `{ label, value }[]` 格式。`Select` 的 `filterOption` 必须提供才能支持输入搜索。

### 表格列类型

| 用法 | 配置方式 |
|---|---|
| 序号列 | `{ type: 'seq' }` |
| 普通文本列 | `{ title, field }` |
| 嵌套对象字段 | `{ title, field: 'meta.title' }`（支持点号路径） |
| 日期列 | `{ title, field, formatter: 'formatDateTime' }` |
| 状态/枚举列 | `{ title, field, slots: { default: 'slotName' } }` |
| 操作列 | `{ title, field: 'action', fixed: 'right', slots: { default: 'action' } }` |
| 树节点列 | `{ title, field, treeNode: true }` |

### Slot 用法

在 columns 中声明 `slots: { default: 'slotName' }`，在 `<template #slotName="{ row }">` 中实现。**必须使用 `a-tag`、`a-button`、`a-popconfirm` 等全局组件，禁止原生 HTML。**

```html
<!-- 状态 Tag -->
<template #status="{ row }">
  <a-tag :color="xxxStatusToColor(row.status)">{{ xxxStatusToName(row.status) }}</a-tag>
</template>

<!-- 操作列 -->
<template #action="{ row }">
  <a-button type="link" :icon="h(LucideFilePenLine)" @click.stop="handleEdit(row)" />
  <a-popconfirm
    :cancel-text="$t('ui.button.cancel')"
    :ok-text="$t('ui.button.ok')"
    :title="$t('ui.text.do_you_want_delete', { moduleName: $t('page.xxx.moduleName') })"
    @confirm="handleDelete(row)"
  >
    <a-button danger type="link" :icon="h(LucideTrash2)" />
  </a-popconfirm>
</template>
```

> 颜色和名称映射函数在 `composables` 层定义（如 `xxxStatusToColor`、`xxxStatusToName`），从 `#/api` 导入。**不要**在 Slot 中硬编码颜色值。

### VxeGrid 配置

```typescript
const gridOptions: VxeGridProps<Xxx> = {
  toolbarConfig: { custom: true, export: true, refresh: true, zoom: true },
  exportConfig: {},
  pagerConfig: {},                    // 启用分页（树形表格设 enabled: false）
  rowConfig: { isHover: true },
  height: 'auto',
  stripe: true,
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchListXxxs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },
  columns: [ /* ... */ ],
};
```

**树形表格**：设 `pagerConfig.enabled: false`，并通过 `treeConfig` 配置 `parentField`+`rowField`+`transform: true`（扁平转树）或 `childrenField`（后端已返回树），树节点列标记 `treeNode: true`。

### Drawer 表单

表单校验规则：`rules: 'required'`（输入框）/ `rules: 'selectRequired'`（下拉），不要用 Zod。

```typescript
const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: [ /* ... */ ],
});
```

---

## 新建业务模块 Checklist

```
- [ ] Step 1: 确认 generated 层已有类型（protobuf 已生成）
- [ ] Step 2: 创建 composables 层（src/api/composables/xxx.ts）
- [ ] Step 3: 注册导出（composables/index.ts）
- [ ] Step 4: 添加 i18n 翻译（zh-CN + en-US 的 enum.json, menu.json, page.json）
- [ ] Step 5: 创建路由模块（router/routes/modules/app/xxx.ts）
- [ ] Step 6: 创建视图页面（views/app/xxx/）
```

### 路由配置模板

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
      order: 3001,
      icon: 'lucide:xxx',
      title: $t('menu.xxx.moduleName'),
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'list',
        name: 'XxxList',
        meta: { title: $t('menu.xxx.list'), authority: ['sys:platform_admin'] },
        component: () => import('#/views/app/xxx/index.vue'),
      },
    ],
  },
];
export default routes;
```

---

## Vben 框架 API 速查

| API | 关键方法 |
|---|---|
| **Page** | `<Page auto-content-height>` 必须包裹，禁止 `<div>` |
| **Grid** | `gridApi.reload()`, `gridApi.grid?.setAllTreeExpand(true/false)` |
| **Drawer** | `drawerApi.open/close/setData({create,row})/getData<T>()/setState({loading})` |
| **Form** | `formApi.getValues()/setValues(obj)/validate()/resetForm()` |
| **Notification** | `notification.success/error({ message: $t('ui.notification.xxx') })` |

预定义 i18n key：`ui.notification.create/update/delete_success` 及 `_failed` 版本。

## 工具链与已知坑（改动前必读）

### 版本钉死（不要改回 ^ 范围）

`pnpm-workspace.yaml` catalog 中以下包为**精确版本钉死**，禁止改回 `^` 范围：

- `vue: 3.5.13` — `^` 范围会让 apps 与 packages 解析出两个 vue patch 版本，vue-router 等 peer 包随之产生两份实例，typecheck 出现大批 `Two different types with this name exist` 假错。
- `typescript: 5.6.3` / `vue-tsc: 2.1.10` — 此配对经全量验证；`^` 会因镜像源元数据漂移解析到不可预期的组合（vue-tsc 2.1.x 与 TS 5.7 不兼容，会崩 `Search string not found`）。

**已知例外（别照这条规则去"顺手修正"）**：根 `package.json` 的 devDependencies 里 `"vue": "^3.5.22"` 确实是范围写法，但 `pnpm-workspace.yaml` 的 `overrides: vue: catalog:` 会把全仓所有 vue 请求强制收敛到 catalog 的 `3.5.13`，单实例保证仍然成立——所以既不要动那个 `^`，也不要删 `overrides` 里的这一行。

`packageManager: pnpm@11.18.0` 必须与本机 pnpm 主版本一致：pnpm 的版本自举切换在 npmmirror 下会下载残缺的 `@pnpm/exe`（缺 pnpm.exe），导致 turbo 嵌套的 `pnpm run` 报 `ERR_PNPM_ENGINE_BIN_MISSING`。本仓已在项目级 `.npmrc` 里写 `manage-package-manager-versions=false` 关掉自举（直接用本机 pnpm，版本一致时行为完全相同）——**别删这一行**，删了才会撞上上述错误。

### TS2589（Type instantiation is excessively deep）

vxe-table 的巨型递归类型与 `DeepPartial` 展开做结构比较时极易超深。已做两层防御：

1. `DeepPartial`（`packages/@core/base/typings/src/helper.d.ts`）已改为**深度 6 层封顶**，请勿改回无界递归。
2. `use-vxe-grid.vue` 中向 `setState`/`mergeWithArrayOverride` 传 vxe 大类型处有显式断言截断。

新增页面若再遇 TS2589：优先在调用点按目标类型断言收窄，**不要**升级 TS/vue-tsc 去碰运气。

### 离线图标（lucide）

菜单 `meta.icon` 与图标选择器统一用 lucide。`@iconify/vue` 默认从 api.iconify.design **在线**加载，网络差时图标渲染失败。`bootstrap.ts` 启动时已 `addCollection(lucide)` 离线注册完整集合——不要删除；新增图标只管用 `lucide:xxx`，无需额外注册。

### 构建目标

`build.target` 必须 **es2022+**（`internal/vite-config/src/config/application.ts`）。降到 es2015 会因依赖中 jiti 的顶层 await 直接断构建。

### `$t()` 空键崩溃

`$t(undefined)` 或不存在的 key 会直接抛错崩页面（曾致菜单页白屏崩溃）。用 `$t()` 前确认 key 已在 `locales/langs/`（zh-CN + en-US 同步）登记；key 来自变量时先保证非空。

### 上游维护者的工序：移植优先（不是对采用者的要求）

三端是并列的**三选一**选项（见仓库根 `AGENTS.md` 与 `docs/adopt-one-frontend.md`）：采用者只取一端，不需要也不应该被要求跟另两端。下面这条只约束**本仓上游维护者 / 向本仓贡献代码的人**：

新功能/新模块由上游以 **react 端为行为基准先实现并验证**，再由维护者移植到 vben。vben 框架变体（vben 5.x monorepo）在训练语料中占比低，AI 直接首创容易产出框架级错误（`$t` 误用、忘记离线图标、component 未用注册名等）；移植是有参照的翻译，便宜得多。

---

## 常见错误与纠正


| 错误做法 | 正确做法 |
|---|---|
| 原生 `<span style="color:red">正常</span>` | `<a-tag :color="statusToColor(row.status)">` |
| `window.confirm('确定删除?')` | `<a-popconfirm>` |
| 原生 `<button>` | `<a-button type="primary/link">` |
| Slot 中 `dayjs(row.createdAt).format(...)` | `formatter: 'formatDateTime'` |
| `rules: z.string().min(1, '不能为空')` | `rules: 'required'` 或 `'selectRequired'` |
| Slot 中 `v-if` 硬编码颜色 | composables 层的 `xxxToColor()` |
| `import { Tag } from 'ant-design-vue'` 后用 `<Tag>` | 直接用 `<a-tag>`（全局注册） |
| `<div>` 包裹页面 | `<Page auto-content-height>` |
| `ref(false)` 管理 loading | `drawerApi.setState({ loading })` |
| `alert()` / `ElMessage` 提示 | `notification` from `ant-design-vue'` |
| `component: 'AInput'` | `component: 'Input'`（注册名） |
| 手动 watch 搜索条件重新请求 | 框架自动关联 `formValues` |
| `:icon="LucideTrash2"` 直接传 | `:icon="h(LucideTrash2)"` 用 h() 包裹 |
| `ref` + `watch` 管理列表数据 | `proxyConfig.ajax.query` + `PaginationQuery` |
