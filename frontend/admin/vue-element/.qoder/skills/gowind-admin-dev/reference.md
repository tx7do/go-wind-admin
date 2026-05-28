# GoWind Admin 开发参考手册

## 1. API 三层架构详解

### 层级关系

```
gRPC 生成代码 (generated/)
    ↓ 只导入类型
服务层 (service/) — 封装 gRPC 客户端，处理请求/响应
    ↓ 调用 service 函数
Composable 层 (composables/) — Vue Query hooks，面向组件使用
```

### 1.1 服务层 (`src/api/service/`)

**职责**：创建 gRPC 服务客户端单例，提供纯异步函数。

```typescript
// src/api/service/xxx.ts
import { createXxxServiceClient, type XxxRequest } from "@/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "@/core/transport/rest";

// 单例模式
let _instance: ReturnType<typeof createXxxServiceClient> | null = null;
export function getXxxService() {
  if (!_instance) _instance = createXxxServiceClient(requestApi);
  return _instance;
}

// 列表查询 — PaginationQuery.toRawParams() 转换
export async function listXxx(query: PaginationQuery) {
  const params = query.toRawParams();
  return getXxxService().List({
    ...params,
    sorting: undefined, offset: undefined, limit: undefined,
    token: undefined, filter: undefined, filterExpr: undefined,
  });
}

// CRUD 操作
export async function getXxx(req: GetXxxRequest) { return getXxxService().Get(req); }
export async function createXxx(req: CreateXxxRequest) { return getXxxService().Create(req); }
export async function updateXxx(req: UpdateXxxRequest) { return getXxxService().Update(req); }
export async function deleteXxx(req: DeleteXxxRequest) { return getXxxService().Delete(req); }
```

**注意事项**：
- `toRawParams()` 返回的对象需要手动清除无用字段（`sorting`, `offset`, `limit`, `token`, `filter`, `filterExpr`）
- `requestApi` 是全局请求适配器，从 `@/core/transport/rest` 导入
- 服务文件在 `src/api/service/index.ts` 中添加 `export *`

### 1.2 Composable 层 (`src/api/composables/`)

**职责**：基于 Vue Query 的 hooks，提供响应式数据管理。

#### 标准 Hook 模式

```typescript
// use* 系列 — Vue Composition API Hook（用于组件 setup）
export function useListXxx(query: PaginationQuery, options?: UseQueryOptions) {
  return useQuery({ queryKey: ["listXxx", query], queryFn: () => listXxx(query), ...options });
}

// fetch* 系列 — 非 Hook 的 Promise 函数（用于 Store、事件处理、手动调用）
export async function fetchListXxx(params: PaginationQuery) {
  return queryClient.fetchQuery({ queryKey: ["listXxx", params], queryFn: () => listXxx(params), retry: 0 });
}
```

#### 创建/更新/删除 Mutation

```typescript
// 创建 — 参数用 { data: {...} } 包裹
export function useCreateXxx(options?: UseMutationOptions) {
  return useMutation({
    mutationFn: (values) => createXxx({ data: { ...values } as XxxType }),
    ...options,
  });
}

// 更新 — 必须使用 makeUpdateMask
export function useUpdateXxx(options?: UseMutationOptions) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateXxx({ id, data: { ...values }, updateMask: makeUpdateMask(Object.keys(values ?? {})) }),
    ...options,
  });
}

// 删除
export function useDeleteXxx(options?: UseMutationOptions) {
  return useMutation({ mutationFn: (req) => deleteXxx(req), ...options });
}
```

#### 枚举工具函数

```typescript
// 枚举列表 — computed + i18n
export const xxxStatusList = computed(() => [
  { value: "ACTIVE", label: t("enum.xxx.status.ACTIVE") },
  { value: "INACTIVE", label: t("enum.xxx.status.INACTIVE") },
]);

// 枚举值 → 显示名称
export function xxxStatusToName(status: XxxStatus) {
  return xxxStatusList.value.find((item) => item.value === status)?.label ?? "";
}

// 枚举值 → 颜色
const STATUS_COLOR_MAP: Record<string, string> = {
  ACTIVE: "#52C41A",
  INACTIVE: "#909399",
  DEFAULT: "#C9CDD4",
};
export function xxxStatusToColor(status: XxxStatus) {
  return STATUS_COLOR_MAP[status as string] || STATUS_COLOR_MAP.DEFAULT;
}
```

---

## 2. Pro 组件库参考

### 2.1 组件总览

| 组件 | 用途 | 导出 |
|------|------|------|
| `ProPage` | 页面编排（组合所有子组件） | `import { ProPage } from "@/components/Pro"` |
| `ProSearch` | 搜索栏 | 由 ProPage 内部编排 |
| `ProToolbar` | 工具栏 | 由 ProPage 内部编排 |
| `ProTable` | 双引擎表格（vxe/el） | 由 ProPage 内部编排 |
| `ProPagination` | 分页 | 由 ProPage 内部编排 |
| `ProModal` | 弹窗 | 由 ProPage 内部编排 |
| `ProForm` | 动态表单 | 独立使用 |
| `ProFileSelect` | 文件选择器 | 独立使用 |

### 2.2 ProPageConfig 配置

```typescript
interface ProPageConfig {
  engine?: "vxe" | "element";  // 表格引擎，默认 "vxe"
  search?: {
    fields: SearchField[];  // 搜索字段配置
    grid?: boolean;         // 是否使用网格布局
  };
  table: {
    listAction: (params: any) => Promise<{ items: any[]; total: number }>;
    deleteAction?: (ids: string) => Promise<void>;
    toolbar?: string[];           // 自定义工具栏按钮
    toolbarRight?: string[];     // 右侧工具栏按钮（如 ["add"]）
    defaultToolbar?: string[];   // 默认工具栏（如 ["refresh", "filter", "exports"]）
    tableAttrs?: Record<string, any>;  // 表格属性
    columns: ColumnConfig[];     // 列配置
  };
  modal?: {
    component?: "dialog" | "drawer";
    dialog?: { title?: string; width?: string };
    drawer?: { title?: string; size?: string };
    fields: FormField[];  // 弹窗表单字段
  };
}
```

### 2.3 列配置 (ColumnConfig)

```typescript
// 普通列
{ prop: "name", label: "名称", minWidth: 120 }

// 序号列
{ type: "index", label: "序号", width: 60 }

// 自定义插槽列
{ prop: "status", label: "状态", minWidth: 100, slotName: "status" }

// 日期列
{ prop: "createdAt", label: "创建时间", minWidth: 160, cellType: "date", dateFormat: "YYYY-MM-DD HH:mm:ss" }

// 标签列
{ prop: "status", label: "状态", minWidth: 80, cellType: "tag", labelMap: { 1: "启用", 0: "禁用" }, tagType: "success" }

// 操作列
{
  prop: "action", label: "操作", fixed: "right", width: 150, cellType: "tool",
  buttons: [
    { name: "edit", label: "编辑" },
    { name: "delete", label: "删除", attrs: { type: "danger" } },
  ],
}
```

### 2.4 搜索字段配置 (SearchField)

```typescript
{ type: "input", label: "名称", field: "name", attrs: { clearable: true } }
{ type: "select", label: "状态", field: "status", options: [...] }
{ type: "date-picker", label: "日期", field: "date" }
{ type: "input", label: "关键词", field: "keyword", attrs: { placeholder: "请输入", clearable: true } }
```

### 2.5 useProModal — 弹窗连接模式

**列表页（提供者）**：
```typescript
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: XxxDrawer,  // 弹窗组件
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();  // 关闭时刷新列表
  },
});

function handleAdd() { modalApi.open({ create: true }); }
function handleEdit(row: any) { modalApi.open({ create: false, row }); }
```

**弹窗组件（消费者）**：
```typescript
const modalApi = injectProModalApi();
const data = computed(() => modalApi.getData<{ create?: boolean; row?: any }>());
const isCreate = computed(() => !!data.value.create);
const visible = computed({
  get: () => modalApi.store.isOpen,
  set: (v) => { if (!v) modalApi.close(); },
});
```

### 2.6 ProPage 事件

```vue
<ProPage @add="handleAdd" @edit="handleEdit" @delete="handleDelete">
```

- `add` — 点击新增按钮
- `edit(row)` — 点击编辑按钮，参数为行数据
- `delete(row)` — 点击删除按钮

---

## 3. PaginationQuery 工具类

```typescript
import { PaginationQuery } from "@/core/transport/rest";

// 列表查询
const pq = new PaginationQuery({
  paging: { page: 1, pageSize: 10 },
  formValues: { name: "test", status: "ACTIVE" },
  orderBy: ["-created_at"],  // 负号表示降序
});
pq.toRawParams();  // 转换为后端 API 参数格式
```

`makeUpdateMask` — 生成字段更新掩码：
```typescript
import { makeUpdateMask } from "@/core/transport/rest";
makeUpdateMask(["name", "status"]);  // "name,status"
```

---

## 4. 国际化规范

### 4.1 翻译文件结构

```
locales/zh-CN/
├── common.json       # 通用文本（按钮、表单、表格等）
├── enum.json         # 枚举值翻译
├── routes.json       # 路由标题
├── preferences.json  # 偏好设置
├── core.json         # 核心功能文本
├── validation.json   # 验证消息
└── pages/            # 页面级翻译
    ├── user.json
    ├── tenant.json
    └── xxx.json      # 每个模块一个文件
```

### 4.2 翻译 key 规范

- 页面文本：`pages.<module>.<field>`，如 `pages.tenant.name`
- 枚举翻译：`enum.<module>.<field>.<VALUE>`，如 `enum.tenant.status.ON`
- 路由标题：`routes.<module>.<page>`，如 `routes.tenant.member`
- 通用文本：`common.<category>.<key>`，如 `common.button.edit`
- 表单占位：`common.placeholder.input` / `common.placeholder.select`

### 4.3 在代码中使用

```typescript
// 组件模板中
$t("pages.xxx.name")

// script setup 中
import { $t } from "@/core/i18n";
const label = $t("pages.xxx.name");

// composable 中（非响应式）
import { i18n } from "@/core/i18n";
const t = i18n.global.t;
const label = t("enum.xxx.status.ACTIVE");
```

**注意**：
- `$t()` 用于响应式场景（模板、computed）
- `t()` 用于非响应式场景（composable 顶层）
- 语言包切换后需等待 `nextTick` 再更新 UI

---

## 5. 路由配置

### 5.1 路由分类

| 类型 | 说明 | 位置 |
|------|------|------|
| core | 基础路由（登录等），无需权限 | `routes/core.routes.ts` |
| dynamic | 动态路由，需权限验证 | `routes/modules/app/*.ts` |
| external | 外部路由（无需 Layout） | `routes/external/`（可选） |

### 5.2 动态路由模块

```typescript
// src/router/routes/modules/app/xxx.ts
import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    path: "/xxx",
    name: "XxxManagement",
    component: Layout,
    redirect: "/xxx/list",
    meta: {
      order: 3000,                          // 菜单排序
      icon: "lucide:xxx-icon",             // UnoCSS 图标
      title: "routes.xxx.moduleName",       // i18n key
      authority: ["sys:xxx_admin"],          // 权限标识
    },
    children: [ /* 子路由 */ ],
  },
];

export default routes;
```

**meta 字段说明**：
- `title`: i18n key 或直接文本
- `icon`: `lucide:` 前缀的图标名
- `order`: 数字越小越靠前
- `authority`: 权限标识数组
- `hideInMenu`: 是否隐藏菜单项
- `hideInTab`: 是否隐藏标签页
- `hideInBreadcrumb`: 是否隐藏面包屑
- `ignoreAccess`: 是否忽略权限检查

---

## 6. 构建与命令

```bash
pnpm dev              # 启动开发服务器
pnpm build            # 类型检查 + 生产构建
pnpm build-only       # 仅构建（不检查类型）
pnpm type-check       # TypeScript 类型检查
pnpm lint             # ESLint + Prettier + Stylelint
pnpm commit           # Git 提交（cz-git 交互式）
```

- Node 版本要求：`^20.19.0 || >=22.12.0`
- 包管理器：仅允许 pnpm（preinstall 检查）
- Git 提交信息：遵循 Conventional Commits 规范
