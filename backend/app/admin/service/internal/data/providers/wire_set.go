//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"github.com/google/wire"

	"go-wind-admin/app/admin/service/internal/data"
)

// ProviderSet is the Wire provider set for data layer.
var ProviderSet = wire.NewSet(
	data.NewRedisClient,
	data.NewEntClient,

	data.NewAuthorizerProvider,
	data.NewAuthenticator,
	data.NewAuthorizer,

	data.NewPasswordCrypto,

	data.NewMinIoClient,

	data.NewDictTypeRepo,
	data.NewDictEntryRepo,
	data.NewTaskRepo,
	data.NewLoginPolicyRepo,

	data.NewOrgUnitRepo,
	data.NewPositionRepo,
	data.NewTenantRepo,

	data.NewUserRepo,
	data.NewUserCredentialRepo,
	data.NewUserOrgUnitRepo,
	data.NewUserPositionRepo,
	data.NewUserRoleRepo,

	data.NewRoleRepo,
	data.NewRoleMetadataRepo,
	data.NewRolePermissionRepo,

	data.NewMembershipRepo,
	data.NewMembershipOrgUnitRepo,
	data.NewMembershipPositionRepo,
	data.NewMembershipRoleRepo,

	data.NewApiRepo,
	data.NewMenuRepo,

	data.NewPermissionRepo,
	data.NewPermissionGroupRepo,
	data.NewPermissionApiRepo,
	data.NewPermissionMenuRepo,
	data.NewPermissionAuditLogRepo,
	data.NewPolicyEvaluationLogRepo,

	data.NewLoginAuditLogRepo,
	data.NewApiAuditLogRepo,

	data.NewFileRepo,

	data.NewInternalMessageRepo,
	data.NewInternalMessageCategoryRepo,
	data.NewInternalMessageRecipientRepo,

	data.NewUserTokenRepo,
)
