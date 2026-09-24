# React frontend: add a CRUD module

Stack: React 19 + Ant Design v6 + ProComponents 2 + TanStack React Query 5 + Zustand + i18next + React Router v6. Path root: `frontend/admin/react/src/`.

**Prerequisite:** the backend proto + codegen has run — `cd backend && make api` (Go) then `cd backend && make ts` (this end's TypeScript; there is **no** `generate:api` script in this frontend) — so `apiClient.<entity>Service` exists in `src/api/generated/admin/service/v1/index.ts`. If it doesn't, stop — do the backend first.

**Mirror these real samples — read them before writing:**
- Hooks layer: `src/api/hooks/role.ts` (canonical 6-hook set)
- List page: `src/pages/app/permission/role/index.tsx` (ProTable + delete + drawer)
- Drawer form: `src/pages/app/permission/role/components/RoleDrawer.tsx` (DrawerForm + formRef)
- Constants: `src/pages/app/permission/role/constants.ts` (enum maps)
- Route module: `src/router/modules/permissions.tsx`
- Authoritative doc: `src/api/README.md`

## Files you will create or edit

**Create (5):**
1. `src/locales/zh-CN/_modules/<entity>.json` — page copy
2. `src/locales/en-US/_modules/<entity>.json` — English mirror
3. `src/api/hooks/<entity>.ts` — 6 hooks
4. `src/pages/app/<group>/<entity>/index.tsx` — list page
5. `src/pages/app/<group>/<entity>/components/<Entity>Drawer.tsx` — form drawer

Plus optional `constants.ts` next to `index.tsx` for enum maps (recommended when there are enums).

**Edit (2):**
6. `src/api/hooks/index.ts` — add `export * from './<entity>';` (the **only** manual registration point)
7. `src/locales/zh-CN/_core/routes.json` + `en-US` — add the menu title key referenced by `meta.title`

`src/api/client.ts` is **never edited** — `apiClient.<entity>Service` is auto-generated.

i18n and routes are auto-discovered via `import.meta.glob`:
- locales: `src/locales/<lang>/_modules/*.json` → namespace = filename. No registration.
- routes: `src/router/modules/*.tsx` → default OR named export. No registration.
- pages: scanned for the dynamic-routing backend mode (pageMap). No registration.

## Step 1 — i18n JSON

`src/locales/zh-CN/_modules/<entity>.json` (en-US same shape, translated):
```json
{
  "pageTitle": "<中文>管理",
  "moduleName": "<中文>",
  "name": "名称",
  "namePlaceholder": "请输入名称",
  "requiredName": "请输入名称",
  "status": "状态",
  "action": "操作",
  "create": "新建<中文>",
  "edit": "编辑<中文>",
  "deleteConfirmTitle": "确认删除",
  "deleteConfirmDesc": "确定要删除该{{moduleName}}吗？",
  "createSuccess": "创建成功",
  "updateSuccess": "更新成功",
  "deleteSuccess": "删除成功",
  "fetchFailed": "获取数据失败",
  "statusMap": { "ON": "启用", "OFF": "禁用" }
}
```
Namespace = filename, so `useTranslation('<entity>')` reads this. Interpolation is `{{var}}`, never `#{var}` or `${var}`.

Add the **menu title** to `src/locales/zh-CN/_core/routes.json` (and en-US mirror): `"products": "商品管理"`. The route `meta.title` points here with the `'routes:'` prefix (see Step 5).

`common` namespace already has `common:button.submit/cancel`, `common:button.*` — reuse for buttons.

## Step 2 — API hooks

Path: `src/api/hooks/<entity>.ts`. Mirror `role.ts`. Six exports (the README documents these; reality is that `useListXxx` is rarely used because ProTable's `request` callback needs a promise, so **list pages default to `fetchListXxx`** — but write both).

```ts
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';
import { makeUpdateMask } from '@/core/transport/rest/utils';
import type {
  <entityservicev1>_List<Entity>Response,
  <entityservicev1>_<Entity>,
  <entityservicev1>_Create<Entity>Request,
  <entityservicev1>_Delete<Entity>Request,
} from '@/api/generated/admin/service/v1';

// 1. useListXxx — for components needing reactive loading/data
export function useList<Entity>s(
  query: PaginationQuery,
  options?: UseQueryOptions<<entityservicev1>_List<Entity>Response, Error>,
) {
  return useQuery({
    queryKey: ['list<Entity>s', query],
    queryFn: () => apiClient.<entity>Service.List(query.toRawParams()),
    ...options,
  });
}

// 2. fetchListXxx — for ProTable request callback, stores, guards (NON-component)
export async function fetchList<Entity>s(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['list<Entity>s', params],
    queryFn: () => apiClient.<entity>Service.List(params.toRawParams()),
    retry: 0,
  });
}

// 3. useGet<Entity>
export function useGet<Entity>(req, options?) { /* useQuery, queryKey: ['get<Entity>', req] */ }

// 4. useCreate<Entity>
export function useCreate<Entity>(options?: UseMutationOptions<{}, Error, <entityservicev1>_Create<Entity>Request>) {
  return useMutation({ mutationFn: (data) => apiClient.<entity>Service.Create(data), ...options });
}

// 5. useUpdate<Entity> — note the FIXED signature { id, values }
export function useUpdate<Entity>(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.<entity>Service.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),  // auto-builds mask incl. 'id'
      }),
    ...options,
  });
}

// 6. useDelete<Entity>
export function useDelete<Entity>(options?: UseMutationOptions<{}, Error, <entityservicev1>_Delete<Entity>Request>) {
  return useMutation({ mutationFn: (req) => apiClient.<entity>Service.Delete(req), ...options });
}
```

**`queryKey` consistency is critical:** `useList<Entity>s` and `fetchList<Entity>s` MUST use the same key prefix `['list<Entity>s', ...]`. After mutations the component invalidates by prefix `['list<Entity>s']` — write the keys wrong and the list won't refresh.

`onSuccess` invalidation is **not** written inside the hook — it's written in the calling component (Step 4), so the component controls what to refresh.

## Step 3 — Register hooks export

Edit `src/api/hooks/index.ts`, add under the appropriate grouping comment:
```ts
export * from './<entity>';
```
This is the only manual registration step in the entire react flow. Forget it and imports from `@/api/hooks` won't resolve.

## Step 4 — List page

Path: `src/pages/app/<group>/<entity>/index.tsx`. Mirror `role/index.tsx`. Skeleton:

```tsx
import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { <entityservicev1>_<Entity> as <Entity> } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchList<Entity>s, useDelete<Entity> } from '@/api/hooks/<entity>';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TABLE } from '@/config/constants';
import <Entity>Drawer from './components/<Entity>Drawer';
// import { getStatusOptions } from './constants';  // if you have enums

export default function <Entity>Page() {
  const { t } = useTranslation('<entity>');           // namespace = filename
  const { message } = App.useApp();                   // NOT bare `message` from antd
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>();
  const tableScrollY = useProTableScrollY();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<<Entity>>();

  const deleteMutation = useDelete<Entity>({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['list<Entity>s'] });  // prefix match
    },
    onError: (error: Error) => message.error(error.message || t('fetchFailed')),
  });

  const columns: ProColumns[] = [ /* your columns; action column: valueType:'option', fixed:'right' */ ];

  return (
    <ContentContainer heightMode="fixed">
      <ProTable
        actionRef={actionRef}
        rowKey="id"
        search={{ labelWidth: 'auto', defaultCollapsed: false }}
        scroll={{ y: tableScrollY, x: 1000 }}            // y MUST be a pixel number, not '%'
        request={async (params) => {
          try {
            const { current, pageSize, ...rest } = params;
            const query = new PaginationQuery({           // MUST use new
              paging: { page: current || 1, pageSize: pageSize || TABLE.DEFAULT_PAGE_SIZE },
              formValues: rest,
            });
            const res = await fetchList<Entity>s(query);
            return { data: res.items || [], total: res.total || 0, success: true };
          } catch (error) {
            return { data: [], total: 0, success: false };
          }
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />}
            onClick={() => { setSelected(undefined); setDrawerMode('create'); setDrawerOpen(true); }}>
            {t('create')}
          </Button>,
        ]}
        columns={columns}
      />
      <<Entity>Drawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected}
        onClose={() => setDrawerOpen(false)}
        onSuccess={() => { setDrawerOpen(false); actionRef.current?.reload(); }}
      />
    </ContentContainer>
  );
}
```

`PaginationQuery` strips empty values from `formValues` automatically, so search inputs that the user left blank won't be sent as filters.

## Step 5 — Drawer form

Path: `src/pages/app/<group>/<entity>/components/<Entity>Drawer.tsx`. Mirror `RoleDrawer.tsx`. Props are fixed:

```tsx
interface <Entity>DrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: <Entity>;
  onClose: () => void;
  onSuccess: () => void;
}
```

Key conventions:
- Use `const formRef = useRef<ProFormInstance>(null);`. **DrawerForm has no `useForm`** — this is the project's hard rule.
- **Edit refill:** in a `useEffect` keyed on `open`/`data`, wrap `formRef.current?.setFieldsValue({...})` in `setTimeout(() => {...}, 0)` so the drawer has rendered. Do NOT pass the record via `initialValues` (it won't update on edit).
- **Close:** `formRef.current?.resetFields()` in `onOpenChange` when `visible === false`.
- Create and update are separate mutations, each with `onSuccess` doing the four-step: `message.success` → `onSuccess()` → `onClose()` → `queryClient.invalidateQueries({ queryKey: ['list<Entity>s'] })`.
- Submit: `mutateAsync` + a `confirmLoading` state wired to `submitButtonProps.loading`.
- Use `App.useApp()` for `message`, never bare `import { message } from 'antd'` (loses theme context in v6).
- Form fields are ProComponents' `ProFormText / ProFormDigit / ProFormTextArea / ProFormRadio.Group` etc.
- For update, call `updateMutation.mutateAsync({ id: data.id, values: formValues })` — the hook builds the updateMask; don't construct it yourself.

## Step 6 — Route

Either edit an existing `src/router/modules/<group>.tsx` (add a child) or create `src/router/modules/<group>.tsx`. Mirror `permissions.tsx`. The leaf entry:

```tsx
import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

export const <group>Routes: AppRouteObject[] = [
  {
    name: '<group>',
    path: '<group>',
    meta: {
      title: 'routes:<group>',
      icon: 'lucide:shield-check',
      order: 2002,
      keepAlive: true,
    },
    children: [
      {
        name: '<entity>s',
        path: '<entity>s',
        element: createLazyRoute(() => import('@/pages/app/<group>/<entity>')),  // points at the DIRECTORY
        meta: {
          title: 'routes:<entity>s',
          icon: 'lucide:shield-user',
          order: 4,
          // authority: ['sys:<group>:view'],   // when ready to enforce, use ARRAY
        },
      },
    ],
  },
];
export default <group>Routes;
```

Rules:
- `element` uses `createLazyRoute(() => import('@/pages/app/.../<entity>'))` — point at the **directory**, not `index.tsx`. Do not use bare `lazy()` or `import()`.
- `meta.title` MUST have the `'routes:'` prefix and the key MUST exist in `routes.json` (Step 1) — otherwise the menu title is blank.
- `icon` is an Iconify string (`'lucide:xxx'`), not an antd Icon component.
- `path` is relative (no leading `/`), auto-joined to the parent.
- For access control use `authority: ['sys:<group>:view']` (an **array**). The codebase currently has `permission: '...'` (string, commented out) in places — that field is NOT in the `RouteMeta` type and won't enforce. Use `authority`.

## React-specific pitfalls

1. **`PaginationQuery` must be `new`-ed.** `new PaginationQuery({ paging, formValues })`. A plain object literal breaks `.toRawParams()`.

2. **Non-component contexts (stores, route guards, `useEffect` async bodies) MUST use `fetchListXxx` / `fetchXxx`**, never `useListXxx`. React Hooks can't run outside a component render.

3. **DrawerForm has no `useForm`.** Use `formRef` + `setFieldsValue` / `resetFields`. Edit refill needs `setTimeout(0)`.

4. **`invalidateQueries` key must prefix-match the list queryKey.** `['list<Entity>s']` invalidates `['list<Entity>s', query]`. Mismatched casing/entity pluralization is the usual silent-refresh bug.

5. **`useUpdate<Entity>` signature is fixed** `{ id, values: Record<string, any> }`. It builds `updateMask` via `makeUpdateMask(Object.keys(values))` (which also appends `'id'`). Don't call `Update` directly.

6. **`meta.title` needs the `'routes:'` prefix and a matching `routes.json` key.** The route's own page copy lives in `_modules/<entity>.json` (no prefix). Two different files, two different responsibilities.

7. **`message` comes from `App.useApp()`.** Bare `message.success(...)` from antd top-level import works visually but loses theme/context.

8. **ProTable `scroll.y` must be a pixel number**, not a percentage string — initial render computes heights.

9. **antd v6 changes:** `items` API replaces `TabPane`; `Alert` uses `title` not `message`. Match existing samples.

10. **`permission:` vs `authority:`** — the type field is `authority: string[]`. Existing `permission: '...'` strings are non-typed and inert. Use `authority`.
