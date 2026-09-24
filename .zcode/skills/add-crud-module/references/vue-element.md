# Vue Element frontend: add a CRUD module

Stack: Vue 3.5 + Vite + TypeScript + Element Plus 2 + vxe-table 4 + Pinia 3 + @tanstack/vue-query 5 + vue-router 5 + vue-i18n 11 + axios (gRPC-Web style). Path root: `frontend/admin/vue-element/src/`.

**Prerequisite:** backend proto + `protoc-gen-typescript-http` regeneration done (`cd backend && make ts`, or `buf generate --template buf.vue-element.admin.typescript.gen.yaml` from `backend/api/`), so `apiClient.<entity>Service` exists in `src/api/generated/admin/service/v1/index.ts`.

**Mirror these real samples — read them before writing:**
- Composables: `src/api/composables/position.ts` (canonical 5-hook + enum utils)
- List page: `src/pages/app/opm/position/index.vue` (ProPage config-driven)
- Drawer form: `src/pages/app/opm/position/position-drawer.vue` (ProModal + ElForm + useDrawerForm)
- Complex form (skip useDrawerForm): `src/pages/app/permission/role/role-drawer.vue` (tree + custom submit)
- Route: `src/router/routes/modules/app/opm.ts`

> **AGENTS.md divergence:** the project `AGENTS.md` mentions vee-validate + zod, but the actual CRUD forms use **Element Plus native `:rules` + `formRef.validate()`**. Follow the real code (ElForm rules), not the doc.

## Files you will create or edit

**Create (5):**
1. `src/api/composables/<entity>.ts` — hooks + enum utils
2. `src/pages/app/<group>/<entity>/index.vue` — list page
3. `src/pages/app/<group>/<entity>/<entity>-drawer.vue` — drawer form
4. `src/locales/zh-CN/pages/<entity>.json` — page copy
5. `src/locales/en-US/pages/<entity>.json` — English mirror

**Edit (4):**
6. `src/api/composables/index.ts` — add `export * from "./<entity>";` (only manual registration)
7. `src/locales/zh-CN/routes.json` + `en-US/routes.json` — add menu title key
8. `src/locales/zh-CN/enum.json` + `en-US/enum.json` — add per-module enum values (if any)
9. Route file (see Step 5)

`src/api/client.ts` is **never edited** — `apiClient.<entity>Service` is generated.

i18n and routes auto-discover via `import.meta.glob`:
- locales: path `zh-CN/pages/<entity>.json` → namespace `pages.<entity>.*`. Directory separators become dots.
- routes: `src/router/routes/modules/**/*.ts` → default export of `RouteRecordRaw[]`. No registration.

## Step 1 — Composables (hooks + enum utils)

Path: `src/api/composables/<entity>.ts`. Mirror `position.ts`. **Use `const t = i18n.global.t;` at top** (not `$t`) because composables run outside setup context but need reactive i18n for enum `computed`s.

```ts
import { computed } from 'vue';
import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from '@tanstack/vue-query';
import { i18n } from '@/core/i18n';
import { apiClient } from '@/api/client';
import { PaginationQuery, makeUpdateMask } from '@/core/transport/rest';
import { queryClient } from '@/plugins/vue-query';
import type { /* generated types */ } from '@/api/generated/admin/service/v1';

const t = i18n.global.t;

// 1. useListXxx — component-internal, reactive
export function useList<Entity>s(query: PaginationQuery, options?: UseQueryOptions<...>) {
  return useQuery({
    queryKey: ['list<Entity>s', query],
    queryFn: () => apiClient.<entity>Service.List(query.toRawParams()),
    ...options,
  });
}

// 2. fetchListXxx — for ProPage listAction, dropdowns, stores, guards
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
//       id, data: { ...values }, updateMask: makeUpdateMask(Object.keys(values ?? {}))
//    })
// 6. useDelete<Entity>

// Enum utilities (only if entity has enums). Each enum exports three things.
export const <entity>TypeList = computed(() => [
  { value: 'REGULAR', label: t('enum.<entity>.type.REGULAR') },
  { value: 'SPECIAL', label: t('enum.<entity>.type.SPECIAL') },
]);
export function <entity>TypeToName(type: string) { /* lookup label */ }
export function <entity>TypeToType(type: string): TagType { /* return element-plus tag type, e.g. 'success' | 'info' */ }
```

(Real naming: `position.ts` exports `positionTypeToName` + `positionTypeToType` (return type `TagType`, imported as `import type { TagType } from './shared';`); there is **no** `*ToColor` in this end — ElTag takes `:type`, not `:color`.)

**No `invalidateQueries` in mutations** — the project refreshes via `pageRef.value?.refresh()` after the drawer emits `success` (see Step 3). If you write invalidate, it won't fire because list data comes from `fetchQuery`, not a reactive `useQuery` subscription.

**Status/enable enums** are shared across modules in `composables/shared.ts` (`statusList`, `statusToName`, `statusToType`, `enableList`, … — note it is `statusToType`, returning `"success" | "info"`, not `statusToColor`) — reuse, don't redefine.

## Step 2 — Register composables export

Edit `src/api/composables/index.ts`, add under the right grouping:
```ts
export * from "./<entity>";
```
Forget this and `@/api/composables` won't expose your hooks OR your enum utils.

## Step 3 — List page

Path: `src/pages/app/<group>/<entity>/index.vue`. Mirror `position/index.vue`. Use the **`ProPage` config-driven** pattern (not hand-rolled vxe-table):

```vue
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <template #status="scope: any"><ElTag :type="statusToType(scope.row.status)">{{ statusToName(scope.row.status) }}</ElTag></template>
    </ProPage>
    <<Entity>Drawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ProPage, type ProPageConfig } from '@/components/Pro';
import { useAccess } from '@/core/access';   // optional
import { PaginationQuery, fetchList<Entity>s, useDelete<Entity>, statusList, statusToType, statusToName } from '@/api/composables';
import { $t } from '@/core/i18n';
import <Entity>Drawer from './<entity>-drawer.vue';

// `$t` is imported from '@/core/i18n' (see above) — `useI18n()` returns `t`, not `$t`.
const pageRef = ref();
const drawerRef = ref();
const { mutateAsync: delete<Entity> } = useDelete<Entity>();

const pageConfig: ProPageConfig = {
  search: {
    grid: true,
    fields: [
      { type: 'input', label: $t('pages.<entity>.name'), field: 'name',
        attrs: { placeholder: $t('common.placeholder.input'), clearable: true } },
      { type: 'select', label: $t('common.table.status'), field: 'status', options: statusList.value },
    ],
  },
  table: {
    listAction: async (query: any) => {                 // MUST use fetchListXxx, NOT useListXxx
      const { page, pageSize, ...queryParams } = query;
      const r = await fetchList<Entity>s(
        new PaginationQuery({                            // MUST use new
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        }),
      );
      return { items: r.items || [], total: r.total || 0 };
    },
    deleteAction: async (ids: string) => { await delete<Entity>({ id: ids as any }); },
    toolbarRight: ['add'],
    defaultToolbar: ['refresh', 'filter'],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: 'index', label: $t('common.table.seq'), width: 60 },
      { prop: 'name', label: $t('pages.<entity>.name'), minWidth: 120 },
      { prop: 'status', label: $t('common.table.status'), minWidth: 100, slotName: 'status' },
      { prop: 'createdAt', label: $t('common.table.createdAt'), minWidth: 160,
        cellType: 'date', dateFormat: 'YYYY-MM-DD HH:mm:ss' },
      { prop: 'action', label: $t('common.table.action'), fixed: 'right', width: 150, cellType: 'tool',
        buttons: [
          { name: 'edit', label: $t('common.button.edit'), icon: 'lucide:pen-line' },
          { name: 'delete', label: $t('common.button.delete'), icon: 'lucide:trash-2', attrs: { type: 'danger' } },
        ] },
    ],
  },
};

function handleAdd() { drawerRef.value?.open({ create: true }); }
function handleEdit(row) { drawerRef.value?.open({ create: false, row }); }
function handleSuccess() { pageRef.value?.refresh(); }      // refresh, NOT invalidate
</script>
```

`ProPage` emits `add` (toolbar) and `edit`/`delete`/`view` (row buttons). Button `name: 'edit' | 'delete'` is auto-recognized; other names route through `@operate`. `cellType` renderers (`date`/`tag`/`switch`/`tool`/...) are registered in `src/components/Pro/ProTable/cellRendererRegistry.ts`.

## Step 4 — Drawer form

Path: `src/pages/app/<group>/<entity>/<entity>-drawer.vue`. Mirror `position-drawer.vue`. **Prefer `useDrawerForm`** — it wraps state, title generation, validate→mutate→toast→close. Only hand-roll (like `role-drawer.vue`) when the form needs transform logic (e.g. tree checkedKeys → leaf id list).

```vue
<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{ component: 'drawer', drawer: { size: drawer.drawerWidth, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="drawer.formData" :rules="formRules" label-width="120px">
      <ElDivider content-position="left">{{ $t('common.section.basic') }}</ElDivider>
      <ElFormItem :label="$t('pages.<entity>.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="$t('common.placeholder.input')" />
      </ElFormItem>
      <!-- more fields -->
    </ElForm>
    <template #footer>
      <ElButton @click="drawer.close">{{ $t('common.button.cancel') }}</ElButton>
      <ElButton type="primary" :loading="drawer.submitLoading.value"
        @click="drawer.handleSubmit(formRef, () => emit('success'))">
        {{ $t('common.button.confirm') }}
      </ElButton>
    </template>
  </ProModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ElForm, ElFormItem, ElInput, ElDivider, ElButton, type FormInstance, type FormRules } from 'element-plus';
import ProModal from '@/components/Pro/ProModal/index.vue';
import { useDrawerForm } from '@/components/Pro/composables/useDrawerForm';
import { useCreate<Entity>, useUpdate<Entity>, fetchListXxx } from '@/api/composables';
import { PaginationQuery } from '@/core/transport/rest';
import { $t } from '@/core/i18n';

const emit = defineEmits<{ success: [] }>();
const { mutateAsync: create<Entity> } = useCreate<Entity>();
const { mutateAsync: update<Entity> } = useUpdate<Entity>();
const formRef = ref<FormInstance>();

const drawer = useDrawerForm({
  moduleKey: 'pages.<entity>.moduleName',                  // drives auto title "新建/编辑 <moduleName>"
  defaults: { name: '', /* ... */ },
  createFn: create<Entity>,
  updateFn: (id, values) => update<Entity>({ id, values }),
  asyncSetup: async () => {                                 // optional: load dropdowns on open
    // const r = await fetchListXxx(new PaginationQuery({ formValues: { status: 'ON' } }));
  },
});

const formRules: FormRules = {
  name: [{ required: true, message: $t('common.validation.required'), trigger: 'blur' }],
};

defineExpose({ open: drawer.open });
</script>
```

`useDrawerForm` handles: `visible`/`isCreate`/`currentId`/`pageLoading`/`submitLoading` state, auto title (`common.modal.create/update` + moduleName), `open`, `resetForm`, `handleSubmit` (validate → create/update → `ElMessage.success(common.notification.createSuccess)` → onSuccess → close, with error fallback). **Validation uses Element Plus `:rules`** — not vee-validate.

For complex forms (tree selection, multi-step), hand-roll state and `handleSubmit` like `role-drawer.vue`. In your catch, check `if (error !== false)` before toasting error — `validate()` rejects with `false` on validation failure, and you don't want a "创建失败" toast for a missing required field.

## Step 5 — i18n + route

**i18n** — `src/locales/zh-CN/pages/<entity>.json`:
```json
{
  "moduleName": "<中文>",
  "name": "名称",
  "code": "编码",
  ...
}
```
Namespace = `pages.<entity>` (path-derived). `$t('pages.<entity>.name')` reads it. Page `<script setup>` and `<template>` use `$t`; composables use `t = i18n.global.t`.

Edit `src/locales/zh-CN/routes.json` (and en-US): under the appropriate domain object, add the menu key (e.g. `"opm": { ..., "<entity>": "<中文>管理" }`). Edit `enum.json` similarly for any new enum values, under `enum.<entity>.<type>.<VALUE>`.

`common.json` already has `button.*`, `table.*`, `placeholder.*`, `validation.*`, `notification.*`, `modal.*`, `section.*` — reuse, don't redefine.

**Route** — `src/router/routes/modules/app/<group>.ts`. **Group by functional domain in ONE file** (opm.ts holds org-units + positions + users; permission.ts holds roles + permissions + apis). Don't create one file per module unless it's a brand-new domain.

Add a child entry:
```ts
{
  path: '<entities>',                       // relative to parent /<group>
  name: '<Entity>Management',               // globally unique
  meta: {
    order: 3,
    icon: 'lucide:box',
    title: 'routes.<group>.<entity>',       // matches the routes.json key
    authority: ['sys:platform_admin', 'sys:tenant_manager'],   // role codes, ARRAY
  },
  component: () => import('@/pages/app/<group>/<entity>/index.vue'),
}
```
Top-level domain entry wraps with `component: Layout`, `redirect`, `meta.order` (large, e.g. 2001/2002 for menu ordering), `meta.keepAlive`. Auto-discovered via glob — no registration. Detail pages use `meta.hideInMenu: true`.

## vue-element-specific pitfalls

1. **Refresh with `pageRef.value?.refresh()`, NOT `invalidateQueries`.** The list pulls via `fetchQuery` (command-style), not a reactive `useQuery` subscription, so invalidation does nothing. The drawer emits `success`, the parent calls `refresh()`.

2. **`listAction` MUST call `fetchListXxx`, not `useListXxx`.** `useQuery` must run synchronously at setup top-level; it can't be called from an async callback.

3. **Update MUST carry `updateMask`.** `useUpdate<Entity>` builds it via `makeUpdateMask(Object.keys(values))`. The backend filters the DTO down to the masked fields; an omitted mask means NO filtering — every populated field in the submitted DTO is written (stale form data corrupts untouched columns).

4. **`PaginationQuery` must be `new`-ed.** Plain object literal breaks `.toRawParams()`. It auto-strips empty values from `formValues`.

5. **Form validation is Element Plus `:rules` + `formRef.validate()`**, despite `AGENTS.md` mentioning vee-validate+zod. Follow the real code.

6. **In the drawer's catch, guard `if (error !== false)` before error toast.** `validate()` rejects with `false`; without the guard you'd show "创建失败" on a missing required field.

7. **`composables/index.ts` export is mandatory** — without it, neither hooks nor enum utils resolve.

8. **Routes are grouped by functional domain** in one file (`opm.ts`, `permission.ts`, `system.ts`, `tenant.ts`, `log.ts`, `internal_message.ts`). Add to the right file; only create a new file for a brand-new domain.

9. **`$t` vs `t`:** `$t` in `.vue` `<template>` and `<script setup>`; `t = i18n.global.t` in composables (no setup context, but enum `computed`s need reactive i18n).

10. **`isTenantUser` flag on `PaginationQuery`** filters out tenant fields from the query — use it for cross-tenant dropdown data sources where you don't want `tenant_id` in the filter.

11. **`ProPage` is config-driven.** Don't hand-roll `<vxe-table>` for a standard CRUD list — describe it via `pageConfig` and let ProPage wire search/table/pagination/toolbar.
