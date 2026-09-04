package data

import (
	"context"
	"encoding/base64"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"
	passwordPolicy "go-wind-admin/pkg/password"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type UserCredentialRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper                  *mapper.CopierMapper[authenticationV1.UserCredential, ent.UserCredential]
	statusConverter         *mapper.EnumTypeConverter[authenticationV1.UserCredential_Status, usercredential.Status]
	identityTypeConverter   *mapper.EnumTypeConverter[authenticationV1.UserCredential_IdentityType, usercredential.IdentityType]
	credentialTypeConverter *mapper.EnumTypeConverter[authenticationV1.UserCredential_CredentialType, usercredential.CredentialType]

	passwordCrypto password.Crypto

	repository *entCrud.Repository[
		ent.UserCredentialQuery, ent.UserCredentialSelect,
		ent.UserCredentialCreate, ent.UserCredentialCreateBulk,
		ent.UserCredentialUpdate, ent.UserCredentialUpdateOne,
		ent.UserCredentialDelete,
		predicate.UserCredential,
		authenticationV1.UserCredential, ent.UserCredential,
	]
}

func NewUserCredentialRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client], passwordCrypto password.Crypto) *UserCredentialRepo {
	repo := &UserCredentialRepo{
		log:                     ctx.NewLoggerHelper("user-credentials/repo/admin-service"),
		entClient:               entClient,
		passwordCrypto:          passwordCrypto,
		mapper:                  mapper.NewCopierMapper[authenticationV1.UserCredential, ent.UserCredential](),
		statusConverter:         mapper.NewEnumTypeConverter[authenticationV1.UserCredential_Status, usercredential.Status](authenticationV1.UserCredential_Status_name, authenticationV1.UserCredential_Status_value),
		identityTypeConverter:   mapper.NewEnumTypeConverter[authenticationV1.UserCredential_IdentityType, usercredential.IdentityType](authenticationV1.UserCredential_IdentityType_name, authenticationV1.UserCredential_IdentityType_value),
		credentialTypeConverter: mapper.NewEnumTypeConverter[authenticationV1.UserCredential_CredentialType, usercredential.CredentialType](authenticationV1.UserCredential_CredentialType_name, authenticationV1.UserCredential_CredentialType_value),
	}

	repo.init()

	return repo
}

func (r *UserCredentialRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.UserCredentialQuery, ent.UserCredentialSelect,
		ent.UserCredentialCreate, ent.UserCredentialCreateBulk,
		ent.UserCredentialUpdate, ent.UserCredentialUpdateOne,
		ent.UserCredentialDelete,
		predicate.UserCredential,
		authenticationV1.UserCredential, ent.UserCredential,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.identityTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.credentialTypeConverter.NewConverterPair())
}

func (r *UserCredentialRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().UserCredential.Query().
		Where(usercredential.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, authenticationV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *UserCredentialRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().UserCredential.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, authenticationV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *UserCredentialRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*authenticationV1.ListUserCredentialResponse, error) {
	if req == nil {
		return nil, authenticationV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().UserCredential.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &authenticationV1.ListUserCredentialResponse{Total: 0, Items: nil}, nil
	}

	return &authenticationV1.ListUserCredentialResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *UserCredentialRepo) Create(ctx context.Context, req *authenticationV1.CreateUserCredentialRequest) (err error) {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.CreateWithTx(ctx, tx, req.GetData())
}

// createCredential 创建凭证。skipPolicy 为 true 时跳过口令复杂度校验，
// 仅供引导期默认管理员种子使用（见 CreateBootstrapCredential）。
func (r *UserCredentialRepo) createCredential(ctx context.Context, tx *ent.Tx, data *authenticationV1.UserCredential, skipPolicy bool) error {
	if data == nil {
		return authenticationV1.ErrorBadRequest("invalid request")
	}

	var err error

	if data.Credential != nil {
		var newCredential string
		newCredential, err = r.prepareCredential(r.credentialTypeConverter.ToEntity(data.CredentialType), data.GetCredential(), skipPolicy)
		if err != nil {
			r.log.Errorf(ctx, "prepare new credential failed: %s", err.Error())
			// 透传底层错误（如"密码过短/复杂度不足"），避免把真实原因吞成笼统的报错，
			// 否则前端只能看到无意义的 "prepare new credential failed"，难以对症处理。
			return err
		}
		data.Credential = trans.Ptr(newCredential)
	}

	builder := tx.UserCredential.Create()
	builder.
		SetUserID(data.GetUserId()).
		SetNillableTenantID(data.TenantId).
		SetNillableIdentityType(r.identityTypeConverter.ToEntity(data.IdentityType)).
		SetNillableIdentifier(data.Identifier).
		SetNillableCredentialType(r.credentialTypeConverter.ToEntity(data.CredentialType)).
		SetNillableCredential(data.Credential).
		SetNillableIsPrimary(data.IsPrimary).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableExtraInfo(data.ExtraInfo).
		SetNillableProvider(data.Provider).
		SetNillableProviderAccountID(data.ProviderAccountId).
		SetCreatedAt(time.Now())

	if err = builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert user credential failed: %s [%v]", err.Error(), data)
		return authenticationV1.ErrorInternalServerError("insert user credential failed")
	}

	return nil
}

func (r *UserCredentialRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *authenticationV1.UserCredential) error {
	return r.createCredential(ctx, tx, data, false)
}

// CreateBootstrapCredential 引导期创建默认管理员（超级用户）口令。
// 该口令为文档约定的初始引导凭据（admin/admin），仅在全新空库初始化时写入，
// 属系统引导账号，故跳过复杂度策略；登录后应尽快改密以符合等保口令要求。
func (r *UserCredentialRepo) CreateBootstrapCredential(ctx context.Context, data *authenticationV1.UserCredential) (err error) {
	if data == nil {
		return authenticationV1.ErrorBadRequest("invalid request")
	}

	var tx *ent.Tx
	if tx, err = r.entClient.Client().Tx(ctx); err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = authenticationV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.createCredential(ctx, tx, data, true)
}

func (r *UserCredentialRepo) Update(ctx context.Context, req *authenticationV1.UpdateUserCredentialRequest) error {
	if req == nil || req.Data == nil {
		return authenticationV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return authenticationV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			err = r.Create(ctx, &authenticationV1.CreateUserCredentialRequest{Data: req.Data})
			return err
		}
	}

	var err error

	if req.Data.Credential != nil {
		var newCredential string
		newCredential, err = r.prepareCredential(r.credentialTypeConverter.ToEntity(req.Data.CredentialType), req.Data.GetCredential(), false)
		if err != nil {
			r.log.Errorf(ctx, "prepare new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("prepare new credential failed")
		}
		req.Data.Credential = trans.Ptr(newCredential)
	}

	builder := r.entClient.Client().UserCredential.Update()
	err = r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *authenticationV1.UserCredential) {
			builder.
				SetNillableIdentityType(r.identityTypeConverter.ToEntity(req.Data.IdentityType)).
				SetNillableIdentifier(req.Data.Identifier).
				SetNillableCredentialType(r.credentialTypeConverter.ToEntity(req.Data.CredentialType)).
				SetNillableCredential(req.Data.Credential).
				SetNillableIsPrimary(req.Data.IsPrimary).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableExtraInfo(req.Data.ExtraInfo).
				SetNillableProvider(req.Data.Provider).
				SetNillableProviderAccountID(req.Data.ProviderAccountId).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(usercredential.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *UserCredentialRepo) Delete(ctx context.Context, id uint32) error {
	builder := r.entClient.Client().UserCredential.Delete()
	builder.Where(usercredential.IDEQ(id))
	if affected, err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return authenticationV1.ErrorInternalServerError("delete one data failed")
	} else {
		if affected == 0 {
			return authenticationV1.ErrorNotFound("user credential not found")
		} else {
			return nil
		}
	}
}

func (r *UserCredentialRepo) DeleteByUserId(ctx context.Context, userId uint32) error {
	builder := r.entClient.Client().UserCredential.Delete()
	builder.Where(usercredential.UserIDEQ(userId))
	if affected, err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return authenticationV1.ErrorInternalServerError("delete one data failed")
	} else {
		if affected == 0 {
			return authenticationV1.ErrorNotFound("user credential not found")
		} else {
			return nil
		}
	}
}

func (r *UserCredentialRepo) DeleteByIdentifier(ctx context.Context, identityType authenticationV1.UserCredential_IdentityType, identifier string) error {
	builder := r.entClient.Client().UserCredential.Delete()
	builder.Where(
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(&identityType)),
		usercredential.IdentifierEQ(identifier),
	)
	// 租户范围过滤（平台管理员 tenant_id=0 不限定）
	if tid, hasTenant := maybeTenantFromViewer(ctx); hasTenant {
		builder.Where(usercredential.TenantIDEQ(tid))
	}
	if affected, err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return authenticationV1.ErrorInternalServerError("delete one data failed")
	} else {
		if affected == 0 {
			return authenticationV1.ErrorNotFound("user credential not found")
		} else {
			return nil
		}
	}
}

func (r *UserCredentialRepo) Get(ctx context.Context, req *authenticationV1.GetUserCredentialRequest) (*authenticationV1.UserCredential, error) {
	if req == nil {
		return nil, authenticationV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().UserCredential.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *authenticationV1.GetUserCredentialRequest_Id:
		whereCond = append(whereCond, usercredential.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// maybeTenantFromViewer returns the viewer's tenant id if present and >0, plus whether a viewer context exists.
func maybeTenantFromViewer(ctx context.Context) (tenantID uint32, hasTenant bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist {
		return 0, false
	}
	tid := vc.TenantID()
	if tid == 0 {
		return 0, false
	}
	return uint32(tid), true
}

func (r *UserCredentialRepo) GetByIdentifier(ctx context.Context, req *authenticationV1.GetUserCredentialByIdentifierRequest) (*authenticationV1.UserCredential, error) {
	builder := r.entClient.Client().UserCredential.Query()

	var whereConds []predicate.UserCredential
	whereConds = append(whereConds,
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	)

	// 租户范围过滤（平台管理员 tenant_id=0 不限定）
	if tid, hasTenant := maybeTenantFromViewer(ctx); hasTenant {
		whereConds = append(whereConds, usercredential.TenantIDEQ(tid))
	}

	builder.Where(whereConds...)

	entity, err := builder.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf(ctx, "query one data failed: %s", err.Error())

		return nil, authenticationV1.ErrorInternalServerError("query data failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// dummyPasswordHash 是一个合法的 bcrypt 哈希，用于在用户不存在时执行一次假校验，
// 让"用户不存在"与"密码错误"两条路径耗时一致，抹掉用户名枚举的计时侧信道。
// 明文不可逆，仅作为恒定耗时的目标。
const dummyPasswordHash = "$2a$10$1sbpKmhQDpXLHnDnEQ1nLe3oOnYyP2bUJyqHcX2T0Fq1qfyoXOrPm"

// performDummyVerify 执行一次假的密码校验，用于抹平计时差异。
// 返回值始终为 false，且从不抛错。
func (r *UserCredentialRepo) performDummyVerify(plainCredential string) {
	_, _ = r.passwordCrypto.Verify(plainCredential, dummyPasswordHash)
}

// FindUserCredential 在指定租户范围内根据身份类型+标识符查询单条凭证并校验密码。
// tenantID 由调用方解析确定（登录时按 tenant_code 解析；留空视为平台 tenant 0）。
// (tenant_id, identity_type, identifier) 在该 tenant 内唯一，无需跨租户消歧。
// needDecrypt 为 true 时，plainCredential 应为 base64+AES 加密密文，会自动解密。
// 返回匹配到的用户 ID，调用方可据此通过 ID 查找用户。
func (r *UserCredentialRepo) FindUserCredential(ctx context.Context, tenantID uint32, identityType authenticationV1.UserCredential_IdentityType, identifier, plainCredential string, needDecrypt bool) (uint32, error) {
	if needDecrypt {
		bytesPass, err := base64.StdEncoding.DecodeString(plainCredential)
		if err != nil {
			r.log.Errorf(ctx, "decode base64 credential failed: %s", err.Error())
			return 0, authenticationV1.ErrorBadRequest("invalid credential format")
		}
		decrypted, err := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		if err != nil {
			r.log.Errorf(ctx, "decrypt credential failed: %s", err.Error())
			return 0, authenticationV1.ErrorBadRequest("decrypt credential failed")
		}
		plainCredential = string(decrypted)
	}

	entity, err := r.entClient.Client().UserCredential.Query().
		Select(
			usercredential.FieldUserID,
			usercredential.FieldCredentialType,
			usercredential.FieldCredential,
			usercredential.FieldStatus,
			usercredential.FieldUpdatedAt,
			usercredential.FieldExtraInfo,
		).
		Where(
			usercredential.TenantIDEQ(tenantID),
			usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(identityType))),
			usercredential.IdentifierEQ(identifier),
		).
		Only(ctx)
	if err != nil {
		// 恒定时间防护：用户不存在时也跑一次 bcrypt 校验，避免被计时攻击枚举用户名
		r.performDummyVerify(plainCredential)
		if ent.IsNotFound(err) {
			return 0, authenticationV1.ErrorUserNotFound("user not found")
		}
		r.log.Errorf(ctx, "query credential failed: %s", err.Error())
		return 0, authenticationV1.ErrorServiceUnavailable("db error")
	}

	if entity.Status == nil || entity.CredentialType == nil || entity.Credential == nil || entity.UserID == nil {
		r.performDummyVerify(plainCredential)
		return 0, authenticationV1.ErrorUserNotFound("user not found")
	}
	if *entity.Status != usercredential.StatusEnabled {
		r.performDummyVerify(plainCredential)
		return 0, authenticationV1.ErrorUserNotFound("user not found")
	}

	if r.verifyCredential(entity.CredentialType, plainCredential, *entity.Credential) {
		// 等保口令策略：有效期检查——超期拒绝登录，用户走重置/改密流程换新口令
		if maxAge := passwordPolicy.MaxAgeDays(); maxAge > 0 && *entity.CredentialType == usercredential.CredentialTypePasswordHash {
			if entity.UpdatedAt != nil && time.Since(*entity.UpdatedAt) > time.Duration(maxAge)*24*time.Hour {
				r.log.Warnf(ctx, "password expired for user [%d] (age > %dd), login denied", *entity.UserID, maxAge)
				return 0, authenticationV1.ErrorBadRequest("password expired, please reset your password")
			}
		}
		return *entity.UserID, nil
	}

	return 0, authenticationV1.ErrorInvalidPassword("incorrect password")
}

func (r *UserCredentialRepo) VerifyCredential(ctx context.Context, req *authenticationV1.VerifyCredentialRequest) (*authenticationV1.VerifyCredentialResponse, error) {
	// 该内部 RPC 请求未携带租户信息，按平台（tenant 0）范围校验。
	if _, err := r.FindUserCredential(ctx, 0, req.GetIdentityType(), req.GetIdentifier(), req.GetCredential(), req.GetNeedDecrypt()); err != nil {
		return nil, err
	}

	return &authenticationV1.VerifyCredentialResponse{
		Success: true,
	}, nil
}

func (r *UserCredentialRepo) verifyCredential(credentialType *usercredential.CredentialType, plainCredential, targetCredential string) bool {
	if credentialType == nil || plainCredential == "" {
		return false
	}

	switch *credentialType {
	case usercredential.CredentialTypePasswordHash:
		ok, err := r.passwordCrypto.Verify(plainCredential, targetCredential)
		if err != nil {
			r.log.Errorf(context.Background(), "verify password failed: %s", err.Error())
			return false
		}
		return ok
	default:
		return plainCredential == targetCredential
	}
}

func (r *UserCredentialRepo) prepareCredential(credentialType *usercredential.CredentialType, plainCredential string, skipComplexity bool) (string, error) {
	var newCredential string
	switch *credentialType {
	case usercredential.CredentialTypePasswordHash:
		// 等保口令策略：哈希前对明文做复杂度校验（覆盖创建/修改/重置全部路径）。
		// 引导期默认管理员种子口令（admin/admin）通过 skipComplexity 跳过。
		if !skipComplexity {
			if err := passwordPolicy.ValidateComplexity(plainCredential); err != nil {
				return "", authenticationV1.ErrorBadRequest("%s", err.Error())
			}
		}
		var err error
		// 加密明文密码
		newCredential, err = r.passwordCrypto.Encrypt(plainCredential)
		if err != nil {
			r.log.Errorf(context.Background(), "hash new password failed: %s", err.Error())
			return "", authenticationV1.ErrorBadRequest("hash new password failed")
		}

	default:
		newCredential = plainCredential
	}

	return newCredential, nil
}

// ChangeCredential 修改认证信息
func (r *UserCredentialRepo) ChangeCredential(ctx context.Context, req *authenticationV1.ChangeCredentialRequest) error {
	if req.GetNeedDecrypt() {
		// 解密旧密码
		oldBytes, err := base64.StdEncoding.DecodeString(req.GetOldCredential())
		if err != nil {
			r.log.Errorf(ctx, "decode base64 old credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("invalid old credential format")
		}
		oldPlain, err := crypto.AesDecrypt(oldBytes, crypto.DefaultAESKey, nil)
		if err != nil {
			r.log.Errorf(ctx, "decrypt old credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("decrypt old credential failed")
		}
		req.OldCredential = string(oldPlain)

		// 解密新密码。此前 base64/AES 错误被吞掉（_），解密失败会把空串/垃圾串当成新口令
		// 哈希存储，导致后续登录恒失败且难以排查。这里与 VerifyCredential 对齐：解密失败即报错。
		newBytes, err := base64.StdEncoding.DecodeString(req.GetNewCredential())
		if err != nil {
			r.log.Errorf(ctx, "decode base64 new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("invalid new credential format")
		}
		newPlain, err := crypto.AesDecrypt(newBytes, crypto.DefaultAESKey, nil)
		if err != nil {
			r.log.Errorf(ctx, "decrypt new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("decrypt new credential failed")
		}
		req.NewCredential = string(newPlain)
	}

	// 租户范围 where 条件（平台管理员 tenant_id=0 不限定）
	tenantWhere := []predicate.UserCredential{
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	}
	if tid, hasTenant := maybeTenantFromViewer(ctx); hasTenant {
		tenantWhere = append(tenantWhere, usercredential.TenantIDEQ(tid))
	}

	entity, err := r.entClient.Client().UserCredential.
		Query().
		Select(
			usercredential.FieldCredentialType,
			usercredential.FieldCredential,
			usercredential.FieldExtraInfo,
		).
		Where(tenantWhere...).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}
		r.log.Errorf(ctx, "query one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("query one data failed")
	}

	if entity.CredentialType == nil {
		return authenticationV1.ErrorNotFound("user credential not found")
	}

	// 验证旧认证信息
	if !r.verifyCredential(entity.CredentialType, req.GetOldCredential(), *entity.Credential) {
		return authenticationV1.ErrorBadRequest("invalid old password")
	}

	var newCredential string
	newCredential, err = r.prepareCredential(entity.CredentialType, req.GetNewCredential(), false)
	if err != nil {
		// 口令策略（复杂度等）错误原样透传，便于前端给出可操作的提示
		r.log.Warnf(ctx, "prepare new credential rejected: %s", err.Error())
		return err
	}

	if newCredential == "" {
		return authenticationV1.ErrorBadRequest("new credential cannot be empty")
	}

	// 等保口令策略：历史口令检查——新明文不得与最近 N 条历史哈希重复
	if err := r.checkPasswordHistory(ctx, entity, req.GetNewCredential()); err != nil {
		return err
	}

	extraInfo := appendPasswordHistory(entity.ExtraInfo, *entity.Credential, passwordPolicy.HistoryCount())

	builder := r.entClient.Client().UserCredential.Update()
	builder.Where(tenantWhere...)
	builder.
		SetCredential(newCredential).
		SetNillableExtraInfo(extraInfo).
		SetUpdatedAt(time.Now())
	if err = builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "update one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("update data failed")
	}

	return nil
}

// ResetCredential 修改认证信息
func (r *UserCredentialRepo) ResetCredential(ctx context.Context, req *authenticationV1.ResetCredentialRequest) error {
	if req.GetNeedDecrypt() {
		// 解密新密码（base64/AES 错误不可吞：吞错会把空串/垃圾串哈希存为新口令）
		bytesPass, err := base64.StdEncoding.DecodeString(req.GetNewCredential())
		if err != nil {
			r.log.Errorf(ctx, "decode base64 new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("invalid new credential format")
		}
		plainPassword, err := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		if err != nil {
			r.log.Errorf(ctx, "decrypt new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("decrypt new credential failed")
		}
		req.NewCredential = string(plainPassword)
	}

	// 租户范围 where 条件（平台管理员 tenant_id=0 不限定）
	tenantWhere := []predicate.UserCredential{
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	}
	if tid, hasTenant := maybeTenantFromViewer(ctx); hasTenant {
		tenantWhere = append(tenantWhere, usercredential.TenantIDEQ(tid))
	}

	entity, err := r.entClient.Client().UserCredential.
		Query().
		Select(
			usercredential.FieldCredentialType,
		).
		Where(tenantWhere...).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}
		r.log.Errorf(ctx, "query one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("query one data failed")
	}

	if entity.CredentialType == nil {
		return authenticationV1.ErrorNotFound("user credential not found")
	}

	var newCredential string
	newCredential, err = r.prepareCredential(entity.CredentialType, req.GetNewCredential(), false)
	if err != nil {
		// 口令策略（复杂度等）错误原样透传，便于前端给出可操作的提示
		r.log.Warnf(ctx, "prepare new credential rejected: %s", err.Error())
		return err
	}

	if newCredential == "" {
		return authenticationV1.ErrorBadRequest("new credential cannot be empty")
	}

	// 等保口令策略：历史口令检查（管理员重置同样不得与近期历史重复）
	if err := r.checkPasswordHistory(ctx, entity, req.GetNewCredential()); err != nil {
		return err
	}

	extraInfo := appendPasswordHistory(entity.ExtraInfo, *entity.Credential, passwordPolicy.HistoryCount())

	builder := r.entClient.Client().UserCredential.Update()
	builder.Where(tenantWhere...)
	builder.
		SetCredential(newCredential).
		SetNillableExtraInfo(extraInfo).
		SetUpdatedAt(time.Now())
	if err = builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "update one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("update data failed")
	}

	return nil
}
