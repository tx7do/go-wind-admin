package data

import (
	"context"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/apiauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/dataaccessauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentryi18n"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	"go-wind-admin/app/admin/service/internal/data/ent/file"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessage"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagecategory"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/loginpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/membership"
	"go-wind-admin/app/admin/service/internal/data/ent/membershiporgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/membershipposition"
	"go-wind-admin/app/admin/service/internal/data/ent/membershiprole"
	"go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/plan"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/policyevaluationlog"
	"go-wind-admin/app/admin/service/internal/data/ent/position"
	"go-wind-admin/app/admin/service/internal/data/ent/role"
	"go-wind-admin/app/admin/service/internal/data/ent/rolemetadata"
	"go-wind-admin/app/admin/service/internal/data/ent/rolepermission"
	"go-wind-admin/app/admin/service/internal/data/ent/task"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"
	"go-wind-admin/app/admin/service/internal/data/ent/userorgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/userposition"
	"go-wind-admin/app/admin/service/internal/data/ent/userrole"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	appViewer "go-wind-admin/pkg/entgo/viewer"
)

// TenantUsageRepo 提供租户用量与配额的实时聚合查询，以及手动清理租户数据。
type TenantUsageRepo struct {
	entClient     *entCrud.EntClient[*ent.Client]
	authenticator *Authenticator
	log           *bLogger.Helper
}

func NewTenantUsageRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	authenticator *Authenticator,
) *TenantUsageRepo {
	return &TenantUsageRepo{
		entClient:     entClient,
		authenticator: authenticator,
		log:           ctx.NewLoggerHelper("tenant-usage/repo/admin-service"),
	}
}

// GetUsage 聚合查询指定租户的当前用量与套餐配额上限。
// 使用 SystemViewerContext 绕过租户隔离以跨表聚合。
func (r *TenantUsageRepo) GetUsage(ctx context.Context, tenantId uint32) (*identityV1.TenantUsage, error) {
	sysCtx := appViewer.NewSystemViewerContext(ctx)

	// 1. 查租户记录，WithPlan(WithQuotas) 预载套餐及其配额。
	t, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDEQ(tenantId)).
		WithPlan(func(q *ent.PlanQuery) { q.WithQuotas() }).
		Only(sysCtx)
	if err != nil || t == nil {
		r.log.Errorf(ctx, "get usage: tenant %d not found: %v", tenantId, err)
		return nil, adminV1.ErrorBadRequest("tenant not found")
	}

	usage := &identityV1.TenantUsage{
		TenantId: tenantId,
	}

	// 2. 从预载的套餐边读取套餐名称和配额上限。
	if t.Edges.Plan != nil {
		planId := t.Edges.Plan.ID
		usage.PlanId = &planId
		if t.Edges.Plan.Name != nil {
			usage.PlanName = t.Edges.Plan.Name
		}
		for _, q := range t.Edges.Plan.Edges.Quotas {
			if q.QuotaType != nil && q.QuotaValue != nil {
				usage.Quotas = append(usage.Quotas, &identityV1.QuotaUsage{
					QuotaType:  mapEntQuotaTypeToProto(*q.QuotaType),
					QuotaValue: *q.QuotaValue,
				})
			}
		}
	}

	// 3. 用户数统计
	userCount, uerr := r.entClient.Client().User.Query().
		Where(user.TenantIDEQ(tenantId)).
		Count(sysCtx)
	if uerr != nil {
		r.log.Errorf(ctx, "get usage: count users failed: %v", uerr)
		userCount = 0
	}
	usage.UserCount = uint64(userCount)

	// 4. 存储占用统计（SUM size，用 Scan 聚合）
	storageSum := uint64(0)
	var storageRows []struct {
		Total uint64 `sql:"total"`
	}
	if serr := r.entClient.Client().File.Query().
		Where(file.TenantIDEQ(tenantId)).
		Aggregate(ent.As(ent.Sum(file.FieldSize), "total")).
		Scan(sysCtx, &storageRows); serr != nil {
		r.log.Errorf(ctx, "get usage: sum file size failed: %v", serr)
	} else if len(storageRows) > 0 {
		storageSum = storageRows[0].Total
	}
	usage.StorageUsedBytes = storageSum

	// 5. API 调用量统计
	apiCount, aerr := r.entClient.Client().ApiAuditLog.Query().
		Where(apiauditlog.TenantIDEQ(tenantId)).
		Count(sysCtx)
	if aerr != nil {
		r.log.Errorf(ctx, "get usage: count api audit logs failed: %v", aerr)
		apiCount = 0
	}
	usage.ApiCallCount = uint64(apiCount)

	return usage, nil
}

// mapEntQuotaTypeToProto 将 ent planquota.QuotaType 字符串枚举映射到 proto PlanQuota_QuotaType。
func mapEntQuotaTypeToProto(qt planquota.QuotaType) identityV1.PlanQuota_QuotaType {
	switch qt {
	case planquota.QuotaTypeUserLimit:
		return identityV1.PlanQuota_USER_LIMIT
	case planquota.QuotaTypeStorage:
		return identityV1.PlanQuota_STORAGE
	case planquota.QuotaTypeApiCall:
		return identityV1.PlanQuota_API_CALL
	default:
		return identityV1.PlanQuota_PLAN_QUOTA_TYPE_UNSPECIFIED
	}
}

// CleanupTenantData 手动清理指定租户的全部业务数据。
//
// 在一个事务中硬删所有带 tenant_id 的业务表数据，保留 sys_tenants 记录（status 改为 OFF），
// 事务提交后吊销该租户全部用户的在线令牌（admin+app 双 ClientType）。
// 使用 SystemViewerContext 绕过租户隔离以删除跨表数据。
func (r *TenantUsageRepo) CleanupTenantData(ctx context.Context, tenantId uint32) error {
	sysCtx := appViewer.NewSystemViewerContext(ctx)

	tx, err := r.entClient.Client().Tx(sysCtx)
	if err != nil {
		r.log.Errorf(ctx, "cleanup tenant %d: start tx failed: %s", tenantId, err.Error())
		return adminV1.ErrorInternalServerError("start transaction failed")
	}

	// 收集租户用户 ID 列表，用于事务提交后吊销令牌。
	// 必须在删除用户记录之前查询，否则删除后无法获取 ID。
	userIds, err := tx.User.Query().
		Where(user.TenantIDEQ(tenantId)).
		IDs(sysCtx)
	if err != nil {
		r.log.Errorf(ctx, "cleanup tenant %d: query user ids failed: %v", tenantId, err)
		_ = tx.Rollback()
		return err
	}

	// 以下每个闭包删除一张带 tenant_id 的业务表。
	// 列表对齐 ent.Client/Tx 中所有拥有 TenantIDEQ 谓词的包（共 29 张表）。
	// 注意：Tx 层的 Delete.Exec 返回 (int, error)，需丢弃 int 仅返回 error。
	deleteFns := []func() error{
		func() error { _, e := tx.ApiAuditLog.Delete().Where(apiauditlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.DataAccessAuditLog.Delete().Where(dataaccessauditlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.DictEntry.Delete().Where(dictentry.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.DictEntryI18n.Delete().Where(dictentryi18n.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.DictType.Delete().Where(dicttype.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.File.Delete().Where(file.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.InternalMessage.Delete().Where(internalmessage.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.InternalMessageCategory.Delete().Where(internalmessagecategory.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.InternalMessageRecipient.Delete().Where(internalmessagerecipient.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.LoginAuditLog.Delete().Where(loginauditlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.LoginPolicy.Delete().Where(loginpolicy.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.Membership.Delete().Where(membership.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.MembershipOrgUnit.Delete().Where(membershiporgunit.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.MembershipPosition.Delete().Where(membershipposition.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.MembershipRole.Delete().Where(membershiprole.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.OperationAuditLog.Delete().Where(operationauditlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.OrgUnit.Delete().Where(orgunit.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.PermissionAuditLog.Delete().Where(permissionauditlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.PolicyEvaluationLog.Delete().Where(policyevaluationlog.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.Position.Delete().Where(position.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.Role.Delete().Where(role.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.RoleMetadata.Delete().Where(rolemetadata.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.RolePermission.Delete().Where(rolepermission.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.Task.Delete().Where(task.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.User.Delete().Where(user.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.UserCredential.Delete().Where(usercredential.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.UserOrgUnit.Delete().Where(userorgunit.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.UserPosition.Delete().Where(userposition.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
		func() error { _, e := tx.UserRole.Delete().Where(userrole.TenantIDEQ(tenantId)).Exec(sysCtx); return e },
	}

	for _, fn := range deleteFns {
		if err = fn(); err != nil {
			r.log.Errorf(ctx, "cleanup tenant %d: delete table data failed: %v", tenantId, err)
			_ = tx.Rollback()
			return err
		}
	}

	// 保留租户记录，将状态改为 OFF 以阻断后续访问（TenantAccessChecker 对 status!=ON 直接 403）。
	if err = tx.Tenant.UpdateOneID(tenantId).
		SetStatus(tenant.StatusOff).
		Exec(sysCtx); err != nil {
		r.log.Errorf(ctx, "cleanup tenant %d: set status OFF failed: %v", tenantId, err)
		_ = tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		r.log.Errorf(ctx, "cleanup tenant %d: commit failed: %s", tenantId, err.Error())
		return adminV1.ErrorInternalServerError("transaction commit failed")
	}

	r.log.Infof(ctx, "cleanup tenant %d: committed deletion of %d tables, set status OFF", tenantId, len(deleteFns))

	// 事务提交后吊销该租户全部用户的在线令牌（admin+app 双 ClientType）。
	// 即使部分吊销失败，status==OFF 也会让 TenantAccessChecker 阻断后续请求。
	if r.authenticator != nil {
		for _, uid := range userIds {
			_ = r.authenticator.RevokeUserToken(ctx, authenticationV1.ClientType_admin, uid)
			_ = r.authenticator.RevokeUserToken(ctx, authenticationV1.ClientType_app, uid)
		}
		r.log.Infof(ctx, "cleanup tenant %d: revoked tokens for %d users", tenantId, len(userIds))
	} else {
		r.log.Warnf(ctx, "cleanup tenant %d: authenticator is nil, skipped token revocation for %d users", tenantId, len(userIds))
	}

	return nil
}

// EnforceExpiryPolicies 扫描所有 status==ON 且 expired_at<=now 的租户，
// 按其套餐的 expiry_policy 修改租户状态，并吊销该租户全部用户的在线令牌。
//
// 策略映射：
//   - BLOCK_LOGIN → status=EXPIRED（后续请求被 TenantAccessChecker 以 status!=ON 拦截）
//   - FREEZE     → status=FREEZE（同上）
//   - READONLY   → 保持 ON，读写拦截交给 TenantAccessChecker 按过期+只读判定
//
// 使用 SystemViewerContext 跨租户扫描。返回被改状态的租户数量。
func (r *TenantUsageRepo) EnforceExpiryPolicies(ctx context.Context) (int, error) {
	sysCtx := appViewer.NewSystemViewerContext(ctx)
	now := time.Now()

	// 查询所有 status==ON 且 expired_at<=now 的租户，WithPlan 预载套餐以读取 expiry_policy。
	tenants, err := r.entClient.Client().Tenant.Query().
		Where(
			tenant.StatusEQ(tenant.StatusOn),
			tenant.ExpiredAtLTE(now),
		).
		WithPlan().
		All(sysCtx)
	if err != nil {
		r.log.Errorf(ctx, "expiry scan: query tenants failed: %v", err)
		return 0, err
	}

	enforcedCount := 0
	for _, t := range tenants {
		// 该租户必须有关联套餐才能判定策略。
		if t.Edges.Plan == nil {
			// 无套餐关联：无法判定策略，跳过（保持 ON）。
			r.log.Warnf(ctx, "expiry scan: tenant %d expired but has no plan, skipped", t.ID)
			continue
		}
		planId := t.Edges.Plan.ID
		expiryPolicy := plan.ExpiryPolicyReadonly
		if t.Edges.Plan.ExpiryPolicy != nil {
			expiryPolicy = *t.Edges.Plan.ExpiryPolicy
		}

		var newStatus tenant.Status
		switch expiryPolicy {
		case plan.ExpiryPolicyBlockLogin:
			newStatus = tenant.StatusExpired
		case plan.ExpiryPolicyFreeze:
			newStatus = tenant.StatusFreeze
		case plan.ExpiryPolicyReadonly:
			// READONLY：保持 ON，读写拦截交给中间件，此处不改状态。
			r.log.Infof(ctx, "expiry scan: tenant %d expired with READONLY policy, kept ON", t.ID)
			continue
		default:
			r.log.Warnf(ctx, "expiry scan: tenant %d plan %d has unknown expiry policy %q, skipped", t.ID, planId, expiryPolicy)
			continue
		}

		// 修改租户状态。
		if err = r.entClient.Client().Tenant.UpdateOneID(t.ID).
			SetStatus(newStatus).
			Exec(sysCtx); err != nil {
			r.log.Errorf(ctx, "expiry scan: tenant %d set status %s failed: %v", t.ID, newStatus, err)
			continue
		}
		enforcedCount++

		// 吊销该租户全部用户的在线令牌（admin+app 双 ClientType）。
		if r.authenticator != nil {
			userIds, uerr := r.entClient.Client().User.Query().
				Where(user.TenantIDEQ(t.ID)).
				IDs(sysCtx)
			if uerr != nil {
				r.log.Errorf(ctx, "expiry scan: tenant %d query user ids failed: %v", t.ID, uerr)
				continue
			}
			for _, uid := range userIds {
				_ = r.authenticator.RevokeUserToken(ctx, authenticationV1.ClientType_admin, uid)
				_ = r.authenticator.RevokeUserToken(ctx, authenticationV1.ClientType_app, uid)
			}
			r.log.Infof(ctx, "expiry scan: tenant %d set status=%s, revoked tokens for %d users", t.ID, newStatus, len(userIds))
		}
	}

	r.log.Infof(ctx, "expiry scan: completed, %d tenants enforced", enforcedCount)
	return enforcedCount, nil
}
