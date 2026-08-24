package data

import (
	"context"
	"fmt"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/usermfafactor"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/pkg/crypto"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// EnrolledFactorInfo 是给管理面（ListEnrolledMethods/GetMFAStatus）返回的因子元信息。
// 不含 secret。
type EnrolledFactorInfo struct {
	ID          uint32
	Method      authenticationV1.MFAMethod
	DisplayName string
	Enabled     bool
	CreatedAt   *time.Time
	LastUsedAt  *time.Time
}

func toMFAMethod(m *usermfafactor.Method) authenticationV1.MFAMethod {
	if m == nil {
		return authenticationV1.MFAMethod_MFA_METHOD_UNSPECIFIED
	}
	switch *m {
	case usermfafactor.MethodTotp:
		return authenticationV1.MFAMethod_TOTP
	case usermfafactor.MethodSms:
		return authenticationV1.MFAMethod_SMS
	case usermfafactor.MethodEmail:
		return authenticationV1.MFAMethod_EMAIL
	case usermfafactor.MethodWebauthn:
		return authenticationV1.MFAMethod_WEBAUTHN
	}
	return authenticationV1.MFAMethod_MFA_METHOD_UNSPECIFIED
}

func toEnrolledEnabled(s *usermfafactor.Status) bool {
	if s == nil {
		return false
	}
	return *s == usermfafactor.StatusEnabled
}

// UserMfaFactorRepo 用户 MFA 因子仓储。
// TOTP secret 以 AES-GCM 密文落库（经 crypto.EncryptIfNeeded），校验时用
// crypto.DecryptIfNeeded 还原明文后交给 totp 库验证。
type UserMfaFactorRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
}

func NewUserMfaFactorRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserMfaFactorRepo {
	return &UserMfaFactorRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("user-mfa-factor/repo/admin-service"),
	}
}

// HasEnabledTotp 查询某用户在指定租户内是否绑定了 ENABLED 状态的 TOTP 因子。
// 供登录流程在密码校验通过后判断是否需进入 MFA 二次验证。
func (r *UserMfaFactorRepo) HasEnabledTotp(ctx context.Context, tenantID, userID uint32) (bool, error) {
	count, err := r.entClient.Client().UserMfaFactor.Query().
		Where(
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
			usermfafactor.MethodEQ(usermfafactor.MethodTotp),
			usermfafactor.StatusEQ(usermfafactor.StatusEnabled),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query has enabled totp failed: %s", err.Error())
		return false, fmt.Errorf("query mfa factor failed")
	}
	return count > 0, nil
}

// FindEnabledTotpForUser 取出某用户 ENABLED 的 TOTP 因子并解密 secret。
// 供 VerifyMFAChallenge 校验登录挑战用。
func (r *UserMfaFactorRepo) FindEnabledTotpForUser(ctx context.Context, tenantID, userID uint32) (factorID uint32, plainSecret string, err error) {
	entity, qerr := r.entClient.Client().UserMfaFactor.Query().
		Where(
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
			usermfafactor.MethodEQ(usermfafactor.MethodTotp),
			usermfafactor.StatusEQ(usermfafactor.StatusEnabled),
		).
		Only(ctx)
	if qerr != nil {
		if ent.IsNotFound(qerr) {
			return 0, "", fmt.Errorf("mfa factor not found")
		}
		r.log.Errorf(ctx, "query enabled totp failed: %s", qerr.Error())
		return 0, "", fmt.Errorf("query mfa factor failed")
	}
	if entity.SecretHash == nil {
		return 0, "", fmt.Errorf("mfa secret missing")
	}
	plain, derr := crypto.DecryptIfNeeded(*entity.SecretHash)
	if derr != nil {
		r.log.Errorf(ctx, "decrypt mfa secret failed: %s", derr.Error())
		return 0, "", fmt.Errorf("decrypt mfa secret failed")
	}
	return entity.ID, plain, nil
}

// ListByUser 列出某用户的全部 MFA 因子（不含 secret），供管理面展示。
func (r *UserMfaFactorRepo) ListByUser(ctx context.Context, tenantID, userID uint32) ([]EnrolledFactorInfo, error) {
	entities, err := r.entClient.Client().UserMfaFactor.Query().
		Where(
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list mfa factors failed: %s", err.Error())
		return nil, fmt.Errorf("list mfa factors failed")
	}
	infos := make([]EnrolledFactorInfo, 0, len(entities))
	for _, e := range entities {
		infos = append(infos, EnrolledFactorInfo{
			ID:          e.ID,
			Method:      toMFAMethod(e.Method),
			DisplayName: derefStr(e.DisplayName),
			Enabled:     toEnrolledEnabled(e.Status),
			CreatedAt:   e.CreatedAt,
			LastUsedAt:  e.LastUsedAt,
		})
	}
	return infos, nil
}

// CreateTotpFactor 创建一条 ENABLED 的 TOTP 因子。secret 会被加密后落库。
// 返回新因子 ID。
func (r *UserMfaFactorRepo) CreateTotpFactor(ctx context.Context, tenantID, userID uint32, plainSecret, displayName string) (uint32, error) {
	encSecret, err := crypto.EncryptIfNeeded(plainSecret)
	if err != nil {
		r.log.Errorf(ctx, "encrypt mfa secret failed: %s", err.Error())
		return 0, fmt.Errorf("encrypt mfa secret failed")
	}
	created, cerr := r.entClient.Client().UserMfaFactor.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetMethod(usermfafactor.MethodTotp).
		SetSecretHash(encSecret).
		SetDisplayName(displayName).
		SetStatus(usermfafactor.StatusEnabled).
		Save(ctx)
	if cerr != nil {
		r.log.Errorf(ctx, "create mfa factor failed: %s", cerr.Error())
		return 0, fmt.Errorf("create mfa factor failed")
	}
	return created.ID, nil
}

// DeleteForUser 按 factorID 删除因子，强制校验 (tenantID, userID) 归属，防越权删他人因子。
// 返回是否确实删除了一行。
func (r *UserMfaFactorRepo) DeleteForUser(ctx context.Context, tenantID, userID, factorID uint32) (bool, error) {
	n, err := r.entClient.Client().UserMfaFactor.Delete().
		Where(
			usermfafactor.IDEQ(factorID),
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "delete mfa factor failed: %s", err.Error())
		return false, fmt.Errorf("delete mfa factor failed")
	}
	return n > 0, nil
}

// DeleteAllByUserMethod 清空某用户指定方法的全部因子（管理端救援重置用）。
// 返回删除行数。
func (r *UserMfaFactorRepo) DeleteAllByUserMethod(ctx context.Context, tenantID, userID uint32, method usermfafactor.Method) (int, error) {
	n, err := r.entClient.Client().UserMfaFactor.Delete().
		Where(
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
			usermfafactor.MethodEQ(method),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "delete mfa factors by method failed: %s", err.Error())
		return 0, fmt.Errorf("delete mfa factors failed")
	}
	return n, nil
}

// GetFactorById 按 id 取因子归属（管理端定位目标用户用），不含 secret。
func (r *UserMfaFactorRepo) GetFactorById(ctx context.Context, factorID uint32) (tenantID, userID uint32, found bool, err error) {
	entity, qerr := r.entClient.Client().UserMfaFactor.Query().
		Select(usermfafactor.FieldTenantID, usermfafactor.FieldUserID, usermfafactor.FieldMethod).
		Where(usermfafactor.IDEQ(factorID)).
		Only(ctx)
	if qerr != nil {
		if ent.IsNotFound(qerr) {
			return 0, 0, false, nil
		}
		r.log.Errorf(ctx, "get mfa factor failed: %s", qerr.Error())
		return 0, 0, false, fmt.Errorf("get mfa factor failed")
	}
	return derefUint32(entity.TenantID), derefUint32(entity.UserID), true, nil
}

// FindFirstByUser 跨租户按用户+方法找首行因子归属（平台管理员救援重置定位 tenant 用）。
// 调用前提：ctx 为平台管理员 viewer（IsPlatformContext 放行全量）或已注入 Allow。
func (r *UserMfaFactorRepo) FindFirstByUser(ctx context.Context, userID uint32, method usermfafactor.Method) (tenantID, uid uint32, found bool, err error) {
	entity, qerr := r.entClient.Client().UserMfaFactor.Query().
		Select(usermfafactor.FieldTenantID, usermfafactor.FieldUserID).
		Where(
			usermfafactor.UserIDEQ(userID),
			usermfafactor.MethodEQ(method),
		).
		First(ctx)
	if qerr != nil {
		if ent.IsNotFound(qerr) {
			return 0, 0, false, nil
		}
		r.log.Errorf(ctx, "find mfa factor by user failed: %s", qerr.Error())
		return 0, 0, false, fmt.Errorf("find mfa factor failed")
	}
	return derefUint32(entity.TenantID), derefUint32(entity.UserID), true, nil
}

// UpdateLastUsed 更新因子最近使用时间。
func (r *UserMfaFactorRepo) UpdateLastUsed(ctx context.Context, tenantID, userID, factorID uint32, at time.Time) error {
	n, err := r.entClient.Client().UserMfaFactor.Update().
		Where(
			usermfafactor.IDEQ(factorID),
			usermfafactor.TenantIDEQ(tenantID),
			usermfafactor.UserIDEQ(userID),
		).
		SetLastUsedAt(at).
		Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "update mfa last_used_at failed: %s", err.Error())
		return fmt.Errorf("update mfa last_used_at failed")
	}
	_ = n
	return nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
