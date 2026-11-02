// 审计日志六仓（api/data_access/login/operation/permission/policy_evaluation）的
// 跨包（service 层）测试装配：与 repo_testkit.go 同契约——逐字段复刻各生产构造器，
// 唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；
// entClient 由调用方传入（测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
//
// 这些仓此前只有 data 包内的白盒 struct literal 装配（*_repo_sqlite_test.go），
// service 层测试无法复用（非导出字段），故补导出构造器。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/dataaccessauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// NewApiAuditLogRepoForTest 逐字段复刻 api_audit_log_repo.go 的 NewApiAuditLogRepo。
func NewApiAuditLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *ApiAuditLogRepo {
	repo := &ApiAuditLogRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.ApiAuditLog, ent.ApiAuditLog](),
	}

	repo.init()

	return repo
}

// NewDataAccessAuditLogRepoForTest 逐字段复刻 data_access_audit_log_repo.go 的
// NewDataAccessAuditLogRepo。
func NewDataAccessAuditLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *DataAccessAuditLogRepo {
	repo := &DataAccessAuditLogRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.DataAccessAuditLog, ent.DataAccessAuditLog](),
		accessTypeConverter: mapper.NewEnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, dataaccessauditlog.AccessType](
			auditV1.DataAccessAuditLog_AccessType_name, auditV1.DataAccessAuditLog_AccessType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, dataaccessauditlog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}

	repo.init()

	return repo
}

// NewLoginAuditLogRepoForTest 逐字段复刻 login_audit_log_repo.go 的 NewLoginAuditLogRepo。
func NewLoginAuditLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *LoginAuditLogRepo {
	repo := &LoginAuditLogRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.LoginAuditLog, ent.LoginAuditLog](),
		statusConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_Status, loginauditlog.Status](
			auditV1.LoginAuditLog_Status_name, auditV1.LoginAuditLog_Status_value,
		),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_ActionType, loginauditlog.ActionType](
			auditV1.LoginAuditLog_ActionType_name, auditV1.LoginAuditLog_ActionType_value,
		),
		riskLevelConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, loginauditlog.RiskLevel](
			auditV1.LoginAuditLog_RiskLevel_name, auditV1.LoginAuditLog_RiskLevel_value,
		),
		loginMethodConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, loginauditlog.LoginMethod](
			auditV1.LoginAuditLog_LoginMethod_name, auditV1.LoginAuditLog_LoginMethod_value,
		),
	}

	repo.init()

	return repo
}

// NewOperationAuditLogRepoForTest 逐字段复刻 operation_audit_log_repo.go 的
// NewOperationAuditLogRepo。
func NewOperationAuditLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *OperationAuditLogRepo {
	repo := &OperationAuditLogRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.OperationAuditLog, ent.OperationAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.OperationAuditLog_ActionType, operationauditlog.Action](
			auditV1.OperationAuditLog_ActionType_name, auditV1.OperationAuditLog_ActionType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, operationauditlog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}

	repo.init()

	return repo
}

// NewPermissionAuditLogRepoForTest 逐字段复刻 permission_audit_log_repo.go 的
// NewPermissionAuditLogRepo。
func NewPermissionAuditLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PermissionAuditLogRepo {
	repo := &PermissionAuditLogRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.PermissionAuditLog, ent.PermissionAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.PermissionAuditLog_ActionType, permissionauditlog.Action](
			auditV1.PermissionAuditLog_ActionType_name, auditV1.PermissionAuditLog_ActionType_value,
		),
	}

	repo.init()

	return repo
}

// NewPolicyEvaluationLogRepoForTest 逐字段复刻 policy_evaluation_log_repo.go 的
// NewPolicyEvaluationLogRepo。
func NewPolicyEvaluationLogRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PolicyEvaluationLogRepo {
	repo := &PolicyEvaluationLogRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog](),
	}

	repo.init()

	return repo
}
