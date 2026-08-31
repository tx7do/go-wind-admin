---
name: add-crud-module
description: End-to-end guide for adding a new CRUD business module to the go-wind-admin monorepo — backend (Go + Kratos + Ent + Wire) plus one or more frontends (react / vue-element / vue-vben). Use whenever the user wants to add a new module, resource, entity, or management page; create a CRUD feature; scaffold a new admin page with list + create + edit + delete; or says things like "新增模块/新增CRUD/加一个 xxx 管理/新建管理页/新增资源/新增功能/add a module/create CRUD entity". Trigger even when the user only mentions the frontend page (the backend proto + generated types are a hard prerequisite, so this skill always coordinates both sides). Do NOT trigger for bug fixes, refactors, or changes to an existing module's logic.
---

# Add CRUD Module (go-wind-admin)

This skill scaffolds a **complete CRUD business module** across the backend and one or more frontends of the go-wind-admin monorepo, following the project's real (not documented-aspirational) patterns.

The monorepo has one backend and three frontends. They differ enough in their list/form/refresh mechanisms that each has its own reference. **This file is the orchestrator — it does not contain the per-framework code templates.** Read the matching reference file(s) before writing any code.

## The three-layer architecture in one breath

```
Backend (always first):  Proto (domain + BFF) → Service → Data/Repo → Server → Wire
Frontend (depends on generated types):  generated → hooks/composables → pages → router/i18n
```

**Hard rule:** the backend's `make api` step generates the `generated/` types that every frontend imports. The frontend cannot be completed before the backend proto + codegen is done. Always do backend first (or at least through Step 3 of the backend flow).

## Step 0 — Clarify requirements (once, up front)

Before writing anything, ask the user (use AskUserQuestion, batch into one call) for:

1. **Entity name** in three forms: PascalCase (`Product`), snake_case (`product`), and Chinese label (`商品`).
2. **Core fields** — a list of `{name, type, enum?, required?}`. Distinguish business fields from the audit fields (created_by/updated_by/deleted_by + timestamps) which are added automatically by Ent mixins and need not be listed.
3. **Functional domain** — does it belong to an existing domain (`permission`, `dict`, `identity`, `opm`, `system`, `tenant`, `log`, …) or is it a brand-new domain? This decides whether you edit an existing proto package / route file or create a new one.
4. **Which frontends** — react, vue-element, vue-vben (multi-select). Backend is always included.
5. **Module shape** — plain CRUD, tree (parent_id), master-detail (two linked tables like dict_type+dict_entry), or tenant-scoped. Plain CRUD is the default; the other shapes extend the reference templates.

Only proceed once you have these answers. Guessing leads to rework across all four ends.

## Step 1 — Execution order (the rule that prevents 80% of mistakes)

```
1. Backend  →  read references/backend.md, execute Steps 1–11
                 (ends with `make api` having produced generated/*.pb.go AND,
                  for the chosen frontend(s), regenerated frontend generated/ types)
2. Frontend →  for each chosen framework, read references/<framework>.md and execute its steps
                 frontend steps DEPEND on the regenerated apiClient.<entity>Service existing
```

If the user insists on frontend-only for an entity that has no backend proto yet, stop and explain the dependency: the frontend's `apiClient.<entity>Service` getter is *generated* from the backend proto, so there is nothing to call without it.

## Step 2 — Backend

Read **`references/backend.md`** in full before touching backend files. It covers the 10-step flow with real sample references:

```
1. domain proto  → 2. BFF proto  → 3. make api (+openapi)
4. ent schema    → 5. make ent
6. repo          → 7. service  → 8. register server
9. make register ENTITY=<entity> → 10. make build verify
```

Real samples to mirror: `backend/api/protos/dict/service/v1/dict_type.proto`, `backend/api/protos/admin/service/v1/i_api.proto`, `backend/app/admin/service/internal/data/api_repo.go`, `backend/app/admin/service/internal/service/dict_type_service.go`. **Read the matching sample file before writing your version** — copy its structure, change names/fields.

After backend Step 3 (`make api`), if a frontend is in scope, also regenerate the frontend generated types so the rest of this skill can proceed:
- react: `cd frontend/admin/react && pnpm generate:api`
- vue-element: regenerate via the backend's `protoc-gen-typescript-http` step (same generator)
- vue-vben: same generator; check `apps/admin/src/api/generated/admin/service/v1/index.ts` has the new `get <entity>Service()` getter

## Step 3 — Frontend(s)

For **each** chosen framework, read its reference and follow its step list. They are not interchangeable:

| Framework | Reference | List component | Form | Refresh mechanism |
|---|---|---|---|---|
| react | `references/react.md` | ProTable | DrawerForm + formRef | `queryClient.invalidateQueries(['listXxx'])` |
| vue-element | `references/vue-element.md` | ProPage (config-driven) | ProModal + ElForm + useDrawerForm | `pageRef.value?.refresh()` (NOT invalidate) |
| vue-vben | `references/vue-vben.md` | VxeGrid + proxyConfig | useVbenDrawer + useVbenForm | `gridApi.reload()` on drawer close |

**Critical:** each reference documents the *actual* project pattern, including places where the repo's `AGENTS.md` is aspirational/out-of-date. Trust the reference over the AGENTS.md when they disagree (e.g. vue-element uses ElForm `:rules`, not vee-validate+zod; vue-vben routes are grouped by functional domain in one file, not one file per module).

## Step 4 — Completion checklist

Before declaring done, verify cross-cutting concerns:

- [ ] Backend compiles: `cd backend && make build` (or `make gen`)
- [ ] Swagger at `http://localhost:7788/docs` shows the new resource with correct routes
- [ ] For each frontend: list loads, search filters, pagination works, create/edit form validates and persists, update actually changes fields, delete confirms and refreshes
- [ ] No edits to any `generated/` directory (these are regenerated, not hand-edited); dependency wiring lives in `cmd/server/wiring.go` (hand-written, no codegen)
- [ ] `apiClient.<entity>Service` getter exists in the frontend generated index (proves proto round-trip worked)
- [ ] i18n: no hardcoded Chinese/English in pages — everything goes through `$t`/`t` with keys in the locale JSONs
- [ ] Update operations carry `updateMask` (frontend `useUpdateXxx` does this automatically via `makeUpdateMask`; do not hand-build the mask)

## Step 5 — Cross-framework pitfalls (memorize these)

These recur on every module regardless of framework. Read them once here, then the per-framework reference adds framework-specific ones.

1. **`PaginationQuery` must be instantiated with `new`.** All three frontends have a `PaginationQuery` class whose `.toRawParams()` depends on instance getters. Passing a plain object literal and calling `.toRawParams()` crashes. Always `new PaginationQuery({ paging, formValues })`.

2. **Every Update call must carry `updateMask`.** The backend's `UpdateX` honors `google.protobuf.FieldMask` — without the mask, no fields are written. All three frontends provide a `useUpdateXxx` mutation that calls `makeUpdateMask(Object.keys(values))` internally. Use it; do not call `apiClient.<entity>Service.Update` directly with a hand-built mask.

3. **Never hand-edit `generated/`.** Backend `api/gen/go/` and each frontend's `api/generated/` are produced by codegen. If a type or service is missing, the fix is to regenerate (fix the proto, rerun `make api` / `pnpm generate:api`), not to edit the generated file.

4. **Backend error codes: Service layer uses `adminV1.ErrorXxx`.** Repo layer may use domain errors (`permissionV1.ErrorXxx`, `dictV1.ErrorXxx`), but the Service (BFF) layer standardizes on `adminV1.ErrorBadRequest` / `adminV1.ErrorInternalServerError` from `protos/admin/service/v1/admin_error.proto`. Don't use `errors.New`.

5. **Refresh mechanism is framework-specific — do not mix them up.**
   - react: `queryClient.invalidateQueries({ queryKey: ['listXxx'] })` (prefix match)
   - vue-element: `pageRef.value?.refresh()` — invalidate does nothing here because the list uses `fetchQuery`, not a reactive `useQuery` subscription
   - vue-vben: `gridApi.reload()` (typically wired into the drawer's `onOpenChange`)

6. **i18n files auto-register via `import.meta.glob` — do not manually register them.** Adding a new locale JSON under the right folder is enough. But the *route menu title* lives in a separate `routes.json` (react/vue-element) or `menu.json` (vue-vben), and route `meta.title` must point at it with the correct prefix (`'routes:xxx'` for react/vue-element, `$t('menu.xxx')` for vue-vben).

## When to read which reference

| Situation | Read |
|---|---|
| Any backend work | `references/backend.md` |
| User chose react | `references/react.md` |
| User chose vue-element | `references/vue-element.md` |
| User chose vue-vben | `references/vue-vben.md` |
| Tree / master-detail / tenant shape | The base reference for that framework, plus inspect a real sample of that shape (e.g. `system/dict/` for master-detail, menu module for tree) |
