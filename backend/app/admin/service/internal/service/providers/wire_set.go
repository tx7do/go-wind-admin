//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"go-wind-admin/app/admin/service/internal/service"

	"github.com/google/wire"
)

// ProviderSet is the Wire provider set for service layer.
var ProviderSet = wire.NewSet(
	service.NewAuthenticationService,
	service.NewUserService,
	service.NewMenuService,
	service.NewRouterService,
	service.NewTaskService,
	service.NewRoleService,
	service.NewOrgUnitService,
	service.NewPositionService,
	service.NewDictService,
	service.NewAdminLoginLogService,
	service.NewAdminOperationLogService,
	service.NewOssService,
	service.NewUEditorService,
	service.NewFileService,
	service.NewTenantService,
	service.NewInternalMessageService,
	service.NewInternalMessageCategoryService,
	service.NewInternalMessageRecipientService,
	service.NewAdminLoginRestrictionService,
	service.NewUserProfileService,
	service.NewUserCredentialService,
	service.NewApiResourceService,
	service.NewPermissionService,
	service.NewPermissionGroupService,
	service.NewPolicyEvaluationLogService,
	service.NewPermissionAuditLogService,
)
