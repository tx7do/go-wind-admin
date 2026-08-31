# Backend: add a CRUD resource (Go + Kratos + Ent, hand-written wiring)

The backend is the source of truth. Frontends depend on the types generated here. Follow the 10 steps in order; each `make` target is a verifiable checkpoint.

**Mirror these real samples — read them before writing your version:**
- Cleanest CRUD: `backend/api/protos/dict/service/v1/dict_type.proto` (domain) + `backend/api/protos/admin/service/v1/i_dict_type.proto` (BFF) + `backend/app/admin/service/internal/service/dict_type_service.go` + `backend/app/admin/service/internal/data/dict_type_repo.go`
- Repo with enum converter: `backend/app/admin/service/internal/data/api_repo.go`
- Schema with mixin + index: `backend/app/admin/service/internal/data/ent/schema/api.go`

## Files you will create or edit

**Create (6):**
1. `backend/api/protos/<domain>/service/v1/<entity>.proto` — domain messages + domain service
2. `backend/api/protos/admin/service/v1/i_<entity>.proto` — BFF REST service
3. `backend/app/admin/service/internal/data/ent/schema/<entity>.go` — Ent schema
4. `backend/app/admin/service/internal/data/<entity>_repo.go` — Repository
5. `backend/app/admin/service/internal/service/<entity>_service.go` — Service
6. *(optional)* `backend/api/protos/<domain>/service/v1/<entity>_error.proto` — domain error enum (only if you want domain-specific errors; otherwise reuse `admin_error`)

**Edit (2, or 0 with `make register`):**
7. `backend/app/admin/service/cmd/server/wiring_ent.go` — `make register ENTITY=<entity>` (Step 9) injects the repo line, service line, and `NewRestServer` arg at the `register:*` markers; or add them by hand
8. `backend/app/admin/service/internal/server/rest_server.go` — param + `Register<Entity>HTTPServer` call; also injected by `make register`
9. `backend/api/buf.gen.yaml` — **only if** you created a brand-new domain (not permission/dict/identity/...). Add a `go_package` override entry.

**Never hand-edit (regenerated):** `api/gen/go/**`, `internal/data/ent/**` (except `schema/`). Dependency wiring in `cmd/server/wiring_ent.go` is hand-written — no codegen step exists for it; a missing wire-up is a compile error at the call site.

## Step 1 — Domain proto

Path: `backend/api/protos/<domain>/service/v1/<entity>.proto`. `package <domain>.service.v1;`. The domain service has **no** `google.api.http` annotations.

Mirror `dict_type.proto`. Key conventions:

- **Every field is `optional`** (matches Ent's `Optional().Nillable()`).
- **Audit field numbers are fixed:** operator IDs at `100` (created_by) / `101` (updated_by) / `102` (deleted_by); timestamps at `200` (created_at) / `201` (updated_at) / `202` (deleted_at). Business fields use numbers 1..N.
- **json_name uses camelCase** (`type_code` → `typeCode`).
- **Get request** uses `oneof query_by` + `optional google.protobuf.FieldMask view_mask` (lets callers trim returned fields).
- **Update request** has `id` + `data` + `update_mask` + `optional bool allow_missing` (allow_missing = upsert).
- **Delete request** — single delete uses `oneof query_by { uint32 id = 1; }`; batch delete uses `repeated uint32 ids = 1;`. Pick one; BFF can also pick differently.

Standard 5 RPCs in the domain service, all with empty `{}` bodies:
```proto
service <Entity>Service {
  rpc List   (pagination.PagingRequest)        returns (List<Entity>Response) {}
  rpc Count  (pagination.PagingRequest)        returns (Count<Entity>Response) {}
  rpc Get    (Get<Entity>Request)              returns (<Entity>) {}
  rpc Create (Create<Entity>Request)           returns (google.protobuf.Empty) {}
  rpc Update (Update<Entity>Request)           returns (google.protobuf.Empty) {}
  rpc Delete (Delete<Entity>Request)           returns (google.protobuf.Empty) {}
}
```
(Count is optional in BFF; keep it in domain if you might need it.)

Imports you'll typically need: `gnostic/openapi/v3/annotations.proto`, `google/protobuf/empty.proto`, `google/protobuf/timestamp.proto`, `google/protobuf/field_mask.proto`, `pagination/v1/pagination.proto`. Add OpenAPI descriptions via `(gnostic.openapi.v3.property)={description:"..."}`.

For an **enum field**, nest the enum inside the entity message (see `permission/service/v1/api.proto` `enum Scope`).

## Step 2 — BFF proto

Path: `backend/api/protos/admin/service/v1/i_<entity>.proto`. `package admin.service.v1;`. **Defines no messages** — import the domain proto and reference its types fully-qualified.

Mirror `i_api.proto` (5-method) or `i_dict_type.proto` (with `additional_bindings` for multi-dimension query + batch delete). Standard shape:

```proto
import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "pagination/v1/pagination.proto";
import "<domain>/service/v1/<entity>.proto";

service <Entity>Service {
  rpc List   (pagination.PagingRequest)              returns (<domain>.service.v1.List<Entity>Response) {
    option (google.api.http) = { get: "/admin/v1/<entities>" };
  }
  rpc Get    (<domain>.service.v1.Get<Entity>Request)    returns (<domain>.service.v1.<Entity>) {
    option (google.api.http) = { get: "/admin/v1/<entities>/{id}" };
  }
  rpc Create (<domain>.service.v1.Create<Entity>Request) returns (google.protobuf.Empty) {
    option (google.api.http) = { post: "/admin/v1/<entities>" body: "*" };
  }
  rpc Update (<domain>.service.v1.Update<Entity>Request) returns (google.protobuf.Empty) {
    option (google.api.http) = { put: "/admin/v1/<entities>/{id}" body: "*" };
  }
  rpc Delete (<domain>.service.v1.Delete<Entity>Request) returns (google.protobuf.Empty) {
    option (google.api.http) = { delete: "/admin/v1/<entities>/{id}" };
  }
}
```

Route prefix is always `/admin/v1/<entities>`. For batch delete the path drops `{id}` (see `i_dict_type.proto`). The BFF can expose fewer RPCs than the domain (e.g. omit Count).

**If you introduced a NEW domain** (one not already present in `buf.gen.yaml`'s overrides), add a `go_package` override entry there pointing `<domain>/service/v1` → `go-wind-admin/api/gen/go/<domain>/service/v1;<domain>pb`. Otherwise `make api` won't know where to emit the domain package.

## Step 3 — Generate proto code

```bash
cd backend && make api && make openapi
```
Produces `api/gen/go/<domain>/service/v1/*.pb.go` and `api/gen/go/admin/service/v1/i_<entity>*.pb.go` (the latter contains `Register<Entity>ServiceHTTPServer`). If a frontend is in scope, also regenerate the frontend generated types now (see SKILL.md Step 2).

## Step 4 — Ent schema

Path: `backend/app/admin/service/internal/data/ent/schema/<entity>.go`. Mirror `schema/api.go` (plain) or `schema/dict_type.go` (with tenant + edges).

```go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/tx7do/go-crud/entgo/mixin"
)

type <Entity> struct{ ent.Schema }

func (<Entity>) Annotations() []schema.Annotation {
    return []schema.Annotation{
        entsql.Annotation{Table: "sys_<entities>", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
        entsql.WithComments(true),
        schema.Comment("<中文表说明>"),
    }
}

func (<Entity>) Fields() []ent.Field {
    return []ent.Field{
        // Business fields — all Optional().Nillable() to match proto optional.
        field.String("name").Comment("名称").Optional().Nillable(),
        // enum field:
        // field.Enum("scope").NamedValues("Admin","ADMIN","App","APP").Default("ADMIN").Optional().Nillable(),
    }
}

func (<Entity>) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixin.AutoIncrementId{},     // uint32 id (matches proto optional uint32 id = 1)
        mixin.TimeAt{},              // created_at + updated_at + deleted_at
        mixin.OperatorID{},          // created_by + updated_by + deleted_by (uint32, nillable)
        mixin.SwitchStatus{},        // status enum (ON/OFF, default ON) — drop if not needed
        // Optional extras:
        // mixin.IsEnabled{},         // bool is_enabled
        // mixin.SortOrder{},         // uint32 sort_order
        // mixin.TenantID[uint32]{},  // tenant_id (multi-tenant)
    }
}

func (<Entity>) Indexes() []ent.Index {
    return []ent.Index{
        // index.Fields("name").Unique().StorageKey("idx_sys_<entity>_name"),
        index.Fields("created_at").StorageKey("idx_sys_<entity>_created_at"),
    }
}
```

**Mixin field names (verified from `go-crud/entgo/mixin/`):** `OperatorID` → `created_by`/`updated_by`/`deleted_by` (uint32 nillable); `TimeAt` → `created_at`/`updated_at`/`deleted_at` (Time nillable); `SwitchStatus` → `status` enum default ON. These match the proto audit fields (100/101/102, 200/201/202) — the `CopierMapper` wires them by name.

For tree shape: `edge.To("children", <Entity>.Type).From("parent").Field("parent_id")` plus a `parent_id` uint32 field.

## Step 5 — Generate Ent code

```bash
cd backend/app/admin/service && make ent
```
Must run from the service directory (schema path is relative). Produces `internal/data/ent/<entity>/`, `internal/data/ent/<entity>_create.go`, etc., and the `predicate.<Entity>` type — all referenced by the repo's 10 generic params.

## Step 6 — Repository

Path: `backend/app/admin/service/internal/data/<entity>_repo.go`. **Read `api_repo.go` in full before writing.** It is the canonical example (with enum converter). `dict_type_repo.go` is the simpler scalar-only variant and also shows batch delete + UpdateOne.

Structure (10 generic type params — order is fixed, do not swap):

```go
package data

import (
    "context"
    "time"

    "entgo.io/ent/dialect/sql"
    entCrud "github.com/tx7do/go-crud/entgo"
    "github.com/tx7do/go-crud/entgo/mapper"
    "github.com/tx7do/go-utils/mapper/copierutil"
    "github.com/go-kratos/kratos/v2/log"

    "go-wind-admin/api/gen/go/<domain>/service/v1"   // domain types, alias as <domain>V1
    "go-wind-admin/app/admin/service/internal/data/ent"
    "go-wind-admin/app/admin/service/internal/data/ent/<entity>"  // predicates + field consts
    "go-wind-admin/app/admin/service/internal/data/ent/predicate"
    "go-wind-admin/pkg/bootstrap"
)

type <Entity>Repo struct {
    entClient *entCrud.EntClient[*ent.Client]
    log       *log.Helper
    mapper    *mapper.CopierMapper[<domain>V1.<Entity>, ent.<Entity>]
    // enumConverter *mapper.EnumTypeConverter[<domain>V1.<Entity>_Scope, <entity>.Scope] // only if enum field
    repository *entCrud.Repository[
        ent.<Entity>Query, ent.<Entity>Select,
        ent.<Entity>Create, ent.<Entity>CreateBulk,
        ent.<Entity>Update, ent.<Entity>UpdateOne,
        ent.<Entity>Delete,
        predicate.<Entity>,
        <domain>V1.<Entity>, ent.<Entity>,
    ]
}

func New<Entity>Repo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *<Entity>Repo {
    r := &<Entity>Repo{
        log:       ctx.NewLoggerHelper("<entity>/repo/admin-service"),
        entClient: entClient,
        mapper:    mapper.NewCopierMapper[<domain>V1.<Entity>, ent.<Entity>](),
    }
    r.init()
    return r
}

func (r *<Entity>Repo) init() {
    r.repository = entCrud.NewRepository[ /* same 10 params as the struct field */ ](r.mapper)
    // REQUIRED for every repo — time <-> timestamppb conversion:
    r.mapper.AppendConverters(copierutil.NewTimeStringConvertPair())
    r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
    // For each enum field:
    // r.mapper.AppendConverters(r.enumConverter.NewConverterPair())
}
```

Five methods (mirror `api_repo.go` lines noted):

- **List** — `r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)`. Note you pass `builder` AND `builder.Clone()` — the clone is used for count.
- **Get** — `switch req.QueryBy.(type)` to build `[]func(s *sql.Selector)` where-conds, then `r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)`. The view_mask controls which fields are returned.
- **Create** — build via `r.entClient.Client().<Entity>.Create().SetNillable*(...).SetNillableCreatedBy(...).SetCreatedAt(time.Now()).Exec(ctx)`. `CreatedBy` is injected by the Service from `auth.FromContext`.
- **Update** — `r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(), setCallback, whereCallback)`. The setCallback applies FieldMask-passed fields via `SetNillable*`; whereCallback sets `sql.EQ(<entity>.FieldID, req.GetId())`. `allow_missing` triggers a Create-if-absent (upsert) inside UpdateX.
- **Delete** — `r.repository.Delete(ctx, builder, whereCallback)`. Batch variant: `r.entClient.Client().<Entity>.Delete().Where(<entity>.IDIn(ids...)).Exec(ctx)`.

**Error returns in repo:** prefer domain errors (`<domain>V1.ErrorInternalServerError(...)`) — though many repos mix in `adminV1`. The Service layer standardizes on `adminV1`. Be consistent within the file.

## Step 7 — Service

Path: `backend/app/admin/service/internal/service/<entity>_service.go`. Mirror `dict_type_service.go`.

```go
package service

import (
    "context"
    "github.com/go-kratos/kratos/v2/log"
    "google.golang.org/protobuf/types/known/emptypb"

    adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
    <domain>V1 "go-wind-admin/api/gen/go/<domain>/service/v1"
    paginationV1 "go-wind-admin/api/gen/go/pagination/v1"
    "go-wind-admin/app/admin/service/internal/data"
    "go-wind-admin/pkg/bootstrap"
    "go-wind-admin/pkg/middleware/auth"
    "go-wind-admin/pkg/utils/trans"
)

type <Entity>Service struct {
    adminV1.<Entity>ServiceHTTPServer   // embed the BFF-generated HTTP server interface
    log  *log.Helper
    repo *data.<Entity>Repo
}

func New<Entity>Service(ctx *bootstrap.Context, repo *data.<Entity>Repo) *<Entity>Service {
    return &<Entity>Service{
        log:  ctx.NewLoggerHelper("<entity>/service/admin-service"),
        repo: repo,
    }
}

// List / Get — delegate straight through
func (s *<Entity>Service) List(ctx context.Context, req *paginationV1.PagingRequest) (*<domain>V1.List<Entity>Response, error) {
    return s.repo.List(ctx, req)
}
func (s *<Entity>Service) Get(ctx context.Context, req *<domain>V1.Get<Entity>Request) (*<domain>V1.<Entity>, error) {
    return s.repo.Get(ctx, req)
}

// Create — pull operator from JWT, inject createdBy
func (s *<Entity>Service) Create(ctx context.Context, req *<domain>V1.Create<Entity>Request) (*emptypb.Empty, error) {
    if req.Data == nil {
        return nil, adminV1.ErrorBadRequest("invalid parameter")
    }
    operator, err := auth.FromContext(ctx)
    if err != nil {
        return nil, err
    }
    req.Data.CreatedBy = trans.Ptr(operator.UserId)
    if err = s.repo.Create(ctx, req); err != nil {
        return nil, err
    }
    return &emptypb.Empty{}, nil
}

// Update — inject id + updatedBy, force updated_by into the mask
func (s *<Entity>Service) Update(ctx context.Context, req *<domain>V1.Update<Entity>Request) (*emptypb.Empty, error) {
    if req.Data == nil {
        return nil, adminV1.ErrorBadRequest("invalid parameter")
    }
    operator, err := auth.FromContext(ctx)
    if err != nil {
        return nil, err
    }
    req.Data.Id = trans.Ptr(req.GetId())
    req.Data.UpdatedBy = trans.Ptr(operator.UserId)
    if req.UpdateMask != nil {
        req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
    }
    if err = s.repo.Update(ctx, req); err != nil {
        return nil, err
    }
    return &emptypb.Empty{}, nil
}

// Delete — delegate (or s.repo.BatchDelete(ctx, req.GetIds()) for batch)
func (s *<Entity>Service) Delete(ctx context.Context, req *<domain>V1.Delete<Entity>Request) (*emptypb.Empty, error) {
    if err := s.repo.Delete(ctx, req); err != nil {
        return nil, err
    }
    return &emptypb.Empty{}, nil
}
```

`auth.FromContext` returns `*authenticationV1.UserTokenPayload`; `operator.UserId` is `uint32`, hence `trans.Ptr`. Validate `req.Data == nil` at entry. Use `adminV1.ErrorXxx` for errors (from `protos/admin/service/v1/admin_error.proto`) — not `errors.New`.

## Step 8 — Register on the server

Edit `backend/app/admin/service/internal/server/rest_server.go`:

1. Add a parameter to `NewRestServer`'s signature:
   ```go
   <entity>Service *service.<Entity>Service,
   ```
2. Inside the function body, register the HTTP handler:
   ```go
   adminV1.Register<Entity>ServiceHTTPServer(srv, <entity>Service)
   ```
   The register function name is `Register<BFFServiceName>HTTPServer` — it is generated by `protoc-gen-go-http` from the BFF proto. Don't use the redaction wrapper that `UserService` uses; that's a special case for sensitive user fields.

## Step 9 — Register the module (make register)

Dependency injection is hand-written — there is no wire/codegen step. Registration is automated:

```bash
cd backend && make register ENTITY=product
```

This injects all five registration points in one shot, anchored on `register:*` marker comments (idempotent, safe to re-run):

- `cmd/server/wiring_ent.go` — repo constructor line, service constructor line, `productService,` arg in the `NewRestServer(...)` call
- `internal/server/rest_server.go` — param in the `NewRestServer` signature, `RegisterProductServiceHTTPServer(srv, productService)` route call

The tool only emits the standard CRUD shape (`data.New<Entity>Repo(ctx, entClient)` / `service.New<Entity>Service(ctx, <entity>Repo)`). For services with extra dependencies, edit the injected lines by hand — the markers show exactly where things belong.

A missing wire-up is a compile error at the call site — if `go build` fails, fix the wiring, not the generated code.

## Step 10 — Build & verify

```bash
cd backend/app/admin/service && make build
# or full: cd backend && make gen
```
Run and check Swagger at `http://localhost:7788/docs` for the new resource. Hit a route via curl/the frontend to confirm.

## Backend-specific pitfalls

1. **`gow` vs `make`.** The `gow` CLI (from `go-wind-toolkit`) is documented in `backend/AGENTS.md` as `gow api` / `gow ent`, but the project's `Makefile` actually wires `make api/ent/gen`. Prefer `make` targets — they are the source of truth for what really runs. `gow` is fine if installed, just don't assume targets exist.

2. **The repo's generic signature has 10 type params, not 9.** Order: Query, Select, Create, CreateBulk, Update, UpdateOne, Delete, Predicate, DTO, Entity. Swapping any two breaks compilation in confusing ways. Copy the exact order from `api_repo.go`.

3. **Time converters are mandatory in `init()`.** Without `copierutil.NewTimeStringConvertPair()` + `NewTimeTimestamppbConverterPair()`, Ent's `time.Time` won't map to proto's `google.protobuf.Timestamp`, and you'll see silent zero values or runtime panics on Create/Update of audit fields.

4. **`allow_missing` upsert in `UpdateX`.** When the proto carries `allow_missing: true`, UpdateX may Create if the row doesn't exist. Make sure your Create path is correct before relying on this.

5. **Audit field numbers are a hard convention.** Don't renumber `created_by` to e.g. `5` — the BFF/domain proto pair, the Ent mixin, and the CopierMapper all agree on 100/101/102 + 200/201/202 by name, and renumbering risks collision with business fields.

6. **Never edit `api/gen/go/` or `internal/data/ent/` (except `schema/`).** If a generated symbol is wrong, fix the source (proto / schema) and regenerate. Dependency wiring is hand-written in `cmd/server/wiring_ent.go` — edit it directly when constructors change.
