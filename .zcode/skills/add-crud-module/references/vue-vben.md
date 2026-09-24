# Vue Vben frontend: add a CRUD module

Stack: Vben Admin monorepo (pnpm + turbo) — Vue 3.5 + Ant Design Vue 4.2 + Tailwind + Shadcn + Pinia + Vue Router + **TanStack Vue Query** + **VxeTable** + i18n + Axios. Path root: `frontend/admin/vue-vben/apps/admin/src/` (the only app — there is no web-antd/web-naive/web-ele multi-UI variant here).

**Prerequisite:** backend proto + regeneration done, so `apiClient.<entity>Service` getter exists in `apps/admin/src/api/generated/admin/service/v1/index.ts`.

**Mirror these real samples — read them before writing:**
- Composables: `apps/admin/src/api/composables/position.ts` (canonical 5-hook + enum utils)
- List page: `apps/admin/src/views/app/opm/position/index.vue` (VxeGrid + proxyConfig)
- Drawer form: `apps/admin/src/views/app/opm/position/position-drawer.vue` (useVbenDrawer + useVbenForm)
- Shared enums: `apps/admin/src/api/composables/shared.ts` (statusList, statusToColor, etc.)
- Route: `apps/admin/src/router/routes/modules/app/opm.ts`
- Coding conventions: `frontend/admin/vue-vben/AGENTS.md`

> **AGENTS.md divergences (verified):** (1) routes are **grouped by functional domain in ONE file** (`opm.ts`, `permission.ts`, …), not one file per module as the doc literally says — add to the existing group file unless it's a brand-new domain. (2) Access control uses **role-based `authority` arrays in route meta**, NOT `v-access`/`useAccess` button-level codes — `views/app/` has zero `v-access` usages.

## Files you will create or edit

**Create (4):**
1. `apps/admin/src/api/composables/<entity>.ts` — hooks + enum utils
2. `apps/admin/src/views/app/<group>/<entity>/index.vue` — list page
3. `apps/admin/src/views/app/<group>/<entity>/<entity>-drawer.vue` — drawer form
4. `apps/admin/src/router/routes/modules/app/<group>.ts` — **only if new domain**; otherwise edit the existing group file

**Edit (5):**
5. `apps/admin/src/api/composables/index.ts` — add `export * from './<entity>';` (only manual registration)
6. `apps/admin/src/locales/langs/zh-CN/menu.json` + `en-US/menu.json` — menu title
7. `apps/admin/src/locales/langs/zh-CN/page.json` + `en-US/page.json` — page copy
8. `apps/admin/src/locales/langs/zh-CN/enum.json` + `en-US/enum.json` — per-module enums (if any)
9. `ui.json` is **usually untouched** — `ui.notification.*_success/failed`, `ui.modal.create/update`, `ui.button.ok/cancel` already exist.

`apps/admin/src/api/client.ts` is **never edited** — `apiClient.<entity>Service` is generated.

i18n and routes auto-discover via `import.meta.glob`:
- locales: `langs/<lang>/*.json` → 4 namespaces (`menu`, `enum`, `page`, `ui`). Path flat, no nesting.
- routes: `router/routes/modules/**/*.ts` → default export of `RouteRecordRaw[]`. No registration.

## Step 1 — Composables (hooks + enum utils)

Path: `apps/admin/src/api/composables/<entity>.ts`. Mirror `position.ts`. Import types directly from `#/api/generated/admin/service/v1` (allowed inside composables even though business code should go via `#/api`). Use `const t = i18n.global.t;` for enum `computed`s (composables have no setup context).

```ts
import { computed } from 'vue';
import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from '@tanstack/vue-query';
import { i18n } from '@vben/locales';
import { apiClient } from '#/api/client';
import { PaginationQuery, makeUpdateMask } from '#/transport/rest';
import { queryClient } from '#/plugins/vue-query';   // verified: apps/admin/src/plugins/vue-query.ts
import type {
  identityservicev1_List<Entity>Response,
  identityservicev1_<Entity>,
  // ...
} from '#/api/generated/admin/service/v1';

const t = i18n.global.t;

// 1. useListXxx — reactive, component-internal
export function useList<Entity>s(query: PaginationQuery, options?: UseQueryOptions<...>) {
  return useQuery({
    queryKey: ['list<Entity>s', query],
    queryFn: () => apiClient.<entity>Service.List(query.toRawParams()),
    ...options,
  });
}

// 2. fetchListXxx — for proxyConfig.ajax.query, dropdowns, stores, guards
export async function fetchList<Entity>s(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['list<Entity>s', params],
    queryFn: () => apiClient.<entity>Service.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

// 3. useGet<Entity>
// 4. useCreate<Entity> — mutationFn: (values) => apiClient.<entity>Service.Create({ data: { ...values } })
// 5. useUpdate<Entity> — mutationFn: ({ id, values }) => apiClient.<entity>Service.Update({
//        id, data: { ...values }, updateMask: makeUpdateMask(Object.keys(values ?? {}))
//    })
// 6. useDelete<Entity>

// Enum utilities (if entity has enums) — three exports each
export const <entity>TypeList = computed(() => [
  { value: 'REGULAR', label: t('enum.<entity>.type.REGULAR') },
]);
export function <entity>TypeToName(v: string) { /* lookup */ }
export function <entity>TypeToColor(v: string, theme?: 'light' | 'dark') { /* return color/type */ }
```

**Status / enable / common enums live in `composables/shared.ts`** (`statusList`, `statusToColor`, `enableList`, …) — reuse, don't redefine.

## Step 2 — Register composables export

Edit `apps/admin/src/api/composables/index.ts`, add (alphabetical within the group):
```ts
export * from './<entity>';
```
Without this, `#/api` won't expose the hooks or enum utils.

## Step 3 — i18n (4 json × 2 languages, mirror key-for-key)

- `langs/zh-CN/menu.json` + `en-US/menu.json` — add the menu title under the domain (or new domain):
  ```json
  "<group>": { "moduleName": "<中文域>", "<entity>": "<中文>管理" }
  ```
- `langs/zh-CN/page.json` + `en-US/page.json` — add page copy:
  ```json
  "<entity>": { "moduleName": "...", "name": "名称", "code": "编码", "button": { "create": "新建" } }
  ```
- `langs/zh-CN/enum.json` + `en-US/enum.json` — add per-module enums (`enum.<entity>.<type>.<VALUE>`); reuse `enum.status.*` if that suffices.
- `ui.json` — only edit if you need a framework-level string that doesn't exist (rare). `ui.notification.create_success`, `ui.modal.create`, `ui.button.ok`, `ui.placeholder.input/select`, `ui.text.do_you_want_delete` (with `{{moduleName}}` interpolation) already exist.

`#/api/index.ts` re-exports `#/transport/rest` (so `PaginationQuery`, `makeUpdateMask` come via `#/api`), `apiClient`, composables, and generated types. Business code imports from `#/api`, not from `#/api/generated/...` directly (composables are the documented exception for type imports).

## Step 4 — List page

Path: `apps/admin/src/views/app/<group>/<entity>/index.vue`. Mirror `position/index.vue`. Three pieces: `formOptions` (search schema), `gridOptions` (VxeTable + proxyConfig), and the assembled `[Grid, gridApi]` + `[Drawer, drawerApi]`.

```vue
<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.<entity>.moduleName')">
      <template #toolbar-tools>
        <a-button type="primary" :icon="h(LucidePlus)" @click="handleCreate">
          {{ $t('page.<entity>.button.create') }}
        </a-button>
      </template>
      <template #status="scope"> <!-- slot name matches column slots.default -->
        <a-tag :color="statusToColor(scope.row.status)">{{ statusToName(scope.row.status) }}</a-tag>
      </template>
      <template #action="{ row }">
        <a-button type="link" :icon="h(LucideFilePenLine)" @click.stop="handleEdit(row)" />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('ui.text.do_you_want_delete', { moduleName: $t('page.<entity>.moduleName') })"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
  </Page>
</template>

<script setup lang="ts">
import { h } from 'vue';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { $t } from '#/locales';
import { LucidePlus, LucideFilePenLine, LucideTrash2 } from '@vben/icons';
import {
  fetchList<Entity>s, PaginationQuery, useDelete<Entity>,
  statusList, statusToColor, statusToName,
} from '#/api';
import { type identityservicev1_<Entity> as <Entity> } from '#/api';
import <Entity>Drawer from './<entity>-drawer.vue';

const { mutateAsync: delete<Entity> } = useDelete<Entity>();

const formOptions: VbenFormProps = {
  schema: [
    { component: 'Input', fieldName: 'name', label: $t('page.<entity>.name'),
      componentProps: { placeholder: $t('ui.placeholder.input') } },
    { component: 'Select', fieldName: 'status', label: $t('ui.table.status'),
      componentProps: { options: statusList, ... } },
  ],
};

const gridOptions = {
  toolbarConfig: { custom: true, export: true, refresh: true, zoom: true },
  pagerConfig: {},
  rowConfig: { isHover: true },
  height: 'auto',
  stripe: true,
  proxyConfig: {                         // ★ data loading — never manage list with ref+watch
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchList<Entity>s(
          new PaginationQuery({           // MUST use new
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },
  columns: [
    { type: 'seq', width: 60 },
    { field: 'name', title: $t('page.<entity>.name'), minWidth: 120 },
    { field: 'status', title: $t('ui.table.status'), slots: { default: 'status' } },
    { field: 'createdAt', title: $t('ui.table.createdAt'), formatter: 'formatDateTime' },
    { field: 'action', title: $t('ui.table.action'), fixed: 'right', width: 150, slots: { default: 'action' } },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });
const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: <Entity>Drawer,
  onOpenChange(isOpen) { if (!isOpen) gridApi.reload(); },   // close ⇒ refresh
});

function openDrawer(create: boolean, row?: <Entity>) { drawerApi.setData({ create, row }); drawerApi.open(); }
function handleCreate() { openDrawer(true); }
function handleEdit(row) { openDrawer(false, row); }
async function handleDelete(row) {
  try {
    await delete<Entity>({ id: row.id });
    // notification.success handled by global interceptor OR explicit:
    // await gridApi.reload();
  } catch { /* error toast handled globally */ }
}
</script>
```

Rules:
- Root node MUST be `<Page auto-content-height>` (from `@vben/common-ui`), not `<div>`.
- **List data goes through `proxyConfig.ajax.query` + `fetchListXxx(new PaginationQuery(...))`.** Never `ref([])` + `watch` + manual request.
- **Refresh via `gridApi.reload()`** (typically in the drawer's `onOpenChange` when it closes). Do NOT call `invalidateQueries`.
- **Icons must be `h()`-wrapped**: `:icon="h(LucideFilePenLine)"`. Icons come from `@vben/icons` (lucide series). Never pass a bare component.
- **Templates use `a-*` global components** (`a-tag`, `a-button`, `a-popconfirm`). Do NOT `import { Tag } from 'ant-design-vue'` and use `<Tag>`.
- **Confirm deletes with `<a-popconfirm>`**, never `window.confirm`.
- **Action-column buttons stay icon-only** (as in `position/index.vue`). `ui.button.edit` / `ui.button.delete` are templates (`编辑{moduleName}` / `删除{moduleName}`) that need a `moduleName` param, and `ui.button.ok` means "确定" — it belongs in the popconfirm's `ok-text`, never as a button label.

## Step 5 — Drawer form

Path: `apps/admin/src/views/app/<group>/<entity>/<entity>-drawer.vue`. Mirror `position-drawer.vue`. Two Vben composables: `useVbenForm` for the form schema, `useVbenDrawer` for the drawer shell.

```vue
<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue';
import { useVbenDrawer } from '@vben/common-ui';
import { useVbenForm } from '#/adapter/form';       // note: the form adapter, not @vben/common-ui
import { $t } from '#/locales';
import { useCreate<Entity>, useUpdate<Entity> } from '#/api';

const data = ref<{ create?: boolean; row?: any }>({});
const { mutateAsync: create<Entity> } = useCreate<Entity>();
const { mutateAsync: update<Entity> } = useUpdate<Entity>();

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: [
    { fieldName: 'name', label: $t('page.<entity>.name'), component: 'Input',
      rules: 'required',                    // 'selectRequired' for dropdowns
      componentProps: { placeholder: $t('ui.placeholder.input') } },
    // more fields
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onConfirm: async () => {
    const { valid } = await baseFormApi.validate();
    if (!valid) return;
    const values = await baseFormApi.getValues();
    try {
      drawerApi.setState({ loading: true });
      if (data.value?.create) await create<Entity>(values);
      else await update<Entity>({ id: data.value?.row?.id, values });
      // success notification (global or explicit)
      drawerApi.close();
    } catch {
      // error notification
    } finally {
      drawerApi.setState({ loading: false });
    }
  },
  onCancel: () => drawerApi.close(),
  onOpenChange(isOpen) {
    if (isOpen) {
      data.value = drawerApi.getData();
      baseFormApi.setValues(data.value?.row ?? {});   // edit refill; empty for create
    }
  },
});

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.<entity>.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.<entity>.moduleName') })
);
</script>
```

Rules:
- `schema[].component` uses the **registered name** (`Input`, `InputNumber`, `Select`, `ApiSelect`, `ApiTreeSelect`, `RadioGroup`, `Switch`, `DatePicker`, …) — see `apps/admin/src/adapter/component/index.ts` `ComponentType` union. Writing `AInput`/`ASelect` breaks.
- `rules: 'required'` for inputs, `'selectRequired'` for dropdowns.
- Loading state via `drawerApi.setState({ loading })` — NOT `ref(false)`.
- `onConfirm` validates → getValues → branch on `data.value.create` → mutate → close. Errors caught here (or globally).

## Step 6 — Route

`apps/admin/src/router/routes/modules/app/<group>.ts`. **Add to the existing group file** unless it's a brand-new domain. Auto-discovered via `import.meta.glob` — no registration.

For a brand-new domain, mirror `opm.ts`:
```ts
import type { RouteRecordRaw } from 'vue-router';
import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const <group>: RouteRecordRaw[] = [{
  path: '/<group>',
  name: '<GroupCamel>Management',
  component: BasicLayout,
  redirect: '/<group>/<entities>',
  meta: {
    order: 2010,                                          // pick an unoccupied menu order
    icon: 'lucide:box',
    title: $t('menu.<group>.moduleName'),
    keepAlive: true,
    authority: ['sys:platform_admin', 'sys:tenant_manager'],   // role codes, ARRAY
  },
  children: [
    {
      path: '<entities>',                                 // no leading slash
      name: '<Entity>Management',                         // globally unique
      meta: {
        order: 1,
        icon: 'lucide:package',
        title: $t('menu.<group>.<entity>'),
        authority: ['sys:platform_admin', 'sys:tenant_manager'],
      },
      component: () => import('#/views/app/<group>/<entity>/index.vue'),
    },
  ],
}];
export default <group>;
```

Detail sub-pages use `meta.hideInMenu: true`. `path` kebab-case, `name` PascalCase, `meta.order` controls menu order, `meta.icon` is `lucide:xxx`. Title uses `$t('menu.<group>.<entity>')` — NOT a `'routes:'` string prefix (that's the react/vue-element convention; vben uses `$t()` directly).

## Step 7 — Access control

Use **`authority: ['role_code', ...]` in route meta** (role-based). The guard calls `useAccess().hasAccess(authority)` (role exact-match ∪ permission-code prefix-match). There is **no button-level `v-access` usage** in `views/app/` currently — if you need it, the directive exists (`v-access="'code'"` / `v-access="['a','b']"`), but the project convention is route-level authority.

## vue-vben-specific pitfalls

1. **Two component naming systems — do not mix.** `schema.component` uses registered names from `adapter/component/index.ts` (`Input`/`Select`/`ApiTreeSelect`/...). Templates use `a-*` global components. Writing `AInput` in a schema or importing `Tag` from antd-design-vue in a template both break.

2. **List data MUST go through `proxyConfig.ajax.query` + `fetchListXxx(new PaginationQuery(...))`.** Never `ref([])` + `watch` + manual fetch. Refresh via `gridApi.reload()`, not `invalidateQueries`.

3. **Update MUST carry `updateMask`.** `useUpdate<Entity>` builds it via `makeUpdateMask(Object.keys(values))`. If you call `apiClient.<entity>Service.Update` directly you must build the mask yourself.

4. **`useList*` vs `fetchList*`:** component-internal reactive needs → `useListXxx`; `proxyConfig.ajax.query` callback, stores, guards, drawer remote dropdowns → `fetchListXxx`.

5. **Icons MUST be `h()`-wrapped** and imported from `@vben/icons`: `:icon="h(LucideFilePenLine)"`. Never pass a bare component reference.

6. **Root node is `<Page auto-content-height>`** from `@vben/common-ui`, never `<div>`.

7. **Loading/visibility state via `drawerApi.setState({ loading })`**, not `ref(false)`.

8. **Query keys are case-sensitive** and must match between `useListXxx`/`fetchListXxx`: `['list<Entity>s', query]`. Pluralization/casing mismatch breaks cache.

9. **Strict `===`, never `==`.** Lint enforces it.

10. **Routes group by functional domain in one file.** Add to `opm.ts`/`permission.ts`/`system.ts`/`tenant.ts`/`log.ts`/`internal_message.ts` as appropriate; only create a new file for a brand-new domain. AGENTS.md says "one file per module" — that is not how the real code is organized.

11. **Access is role-based `authority` arrays in route meta**, not `v-access` button codes. `views/app/` has zero `v-access` usages today.
