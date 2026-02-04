package data

import (
	"context"
	"encoding/base64"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

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

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type UserCredentialRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

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
		r.log.Errorf("query exist failed: %s", err.Error())
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
		r.log.Errorf("query count failed: %s", err.Error())
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
		r.log.Errorf("start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.CreateWithTx(ctx, tx, req.GetData())
}

func (r *UserCredentialRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *authenticationV1.UserCredential) error {
	if data == nil {
		return authenticationV1.ErrorBadRequest("invalid request")
	}

	var err error

	if data.Credential != nil {
		var newCredential string
		newCredential, err = r.prepareCredential(r.credentialTypeConverter.ToEntity(data.CredentialType), data.GetCredential())
		if err != nil {
			r.log.Errorf("prepare new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("prepare new credential failed")
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
		r.log.Errorf("insert user credential failed: %s [%v]", err.Error(), data)
		return authenticationV1.ErrorInternalServerError("insert user credential failed")
	}

	return nil
}

func (r *UserCredentialRepo) Update(ctx context.Context, req *authenticationV1.UpdateUserCredentialRequest) error {
	if req == nil || req.Data == nil {
		return authenticationV1.ErrorBadRequest("invalid request")
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
		newCredential, err = r.prepareCredential(r.credentialTypeConverter.ToEntity(req.Data.CredentialType), req.Data.GetCredential())
		if err != nil {
			r.log.Errorf("prepare new credential failed: %s", err.Error())
			return authenticationV1.ErrorBadRequest("prepare new credential failed")
		}
		req.Data.Credential = trans.Ptr(newCredential)
	}

	builder := r.entClient.Client().Debug().UserCredential.Update()
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

		r.log.Errorf("delete one data failed: %s", err.Error())

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

		r.log.Errorf("delete one data failed: %s", err.Error())

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
	if affected, err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

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

func (r *UserCredentialRepo) GetByIdentifier(ctx context.Context, req *authenticationV1.GetUserCredentialByIdentifierRequest) (*authenticationV1.UserCredential, error) {
	builder := r.entClient.Client().UserCredential.Query()

	builder.Where(
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	)

	entity, err := builder.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, authenticationV1.ErrorNotFound("user credential not found")
		}

		r.log.Errorf("query one data failed: %s", err.Error())

		return nil, authenticationV1.ErrorInternalServerError("query data failed")
	}

	return r.mapper.ToDTO(entity), nil
}

func (r *UserCredentialRepo) VerifyCredential(ctx context.Context, req *authenticationV1.VerifyCredentialRequest) (*authenticationV1.VerifyCredentialResponse, error) {
	if req.GetNeedDecrypt() {
		// 解密密码
		bytesPass, err := base64.StdEncoding.DecodeString(req.GetCredential())
		if err != nil {
			r.log.Errorf("decode base64 credential failed: %s", err.Error())
			return nil, authenticationV1.ErrorBadRequest("invalid credential format")
		}
		plainPassword, err := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		if err != nil {
			r.log.Errorf("decrypt credential failed: %s", err.Error())
			return nil, authenticationV1.ErrorBadRequest("decrypt credential failed")
		}

		req.Credential = string(plainPassword)
	}

	entity, err := r.entClient.Client().UserCredential.Query().
		Select(usercredential.FieldCredentialType, usercredential.FieldCredential, usercredential.FieldStatus).
		Where(
			usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
			usercredential.IdentifierEQ(req.GetIdentifier()),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, authenticationV1.ErrorUserNotFound("user not found")
		}

		r.log.Errorf("query one data failed: %s", err.Error())

		return nil, authenticationV1.ErrorServiceUnavailable("db error")
	}

	if *entity.Status != usercredential.StatusEnabled {
		return nil, authenticationV1.ErrorUserFreeze("account has freeze")
	}

	if !r.verifyCredential(entity.CredentialType, req.GetCredential(), *entity.Credential) {
		return nil, authenticationV1.ErrorIncorrectPassword("incorrect password")
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
			r.log.Errorf("verify password failed: %s", err.Error())
			return false
		}
		return ok
	default:
		return plainCredential == targetCredential
	}
}

func (r *UserCredentialRepo) prepareCredential(credentialType *usercredential.CredentialType, plainCredential string) (string, error) {
	var newCredential string
	switch *credentialType {
	case usercredential.CredentialTypePasswordHash:
		var err error
		// 加密明文密码
		newCredential, err = r.passwordCrypto.Encrypt(plainCredential)
		if err != nil {
			r.log.Errorf("hash new password failed: %s", err.Error())
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
		// 解密密码
		bytesPass, _ := base64.StdEncoding.DecodeString(req.GetOldCredential())
		plainPassword, _ := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		req.OldCredential = string(plainPassword)

		bytesPass, _ = base64.StdEncoding.DecodeString(req.GetNewCredential())
		plainPassword, _ = crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		req.NewCredential = string(plainPassword)
	}

	entity, err := r.entClient.Client().UserCredential.
		Query().
		Select(
			usercredential.FieldCredentialType,
			usercredential.FieldCredential,
		).
		Where(
			usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
			usercredential.IdentifierEQ(req.GetIdentifier()),
		).
		Only(ctx)
	if err != nil {
		r.log.Errorf("query one data failed: %s", err.Error())
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
	newCredential, err = r.prepareCredential(entity.CredentialType, req.GetOldCredential())
	if err != nil {
		r.log.Errorf("prepare new credential failed: %s", err.Error())
		return authenticationV1.ErrorBadRequest("prepare new credential failed")
	}

	if newCredential == "" {
		return authenticationV1.ErrorBadRequest("new credential cannot be empty")
	}

	builder := r.entClient.Client().UserCredential.Update()
	builder.Where(
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	)
	builder.
		SetCredential(newCredential).
		SetUpdatedAt(time.Now())
	if err = builder.Exec(ctx); err != nil {
		r.log.Errorf("update one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("update data failed")
	}

	return nil
}

// ResetCredential 修改认证信息
func (r *UserCredentialRepo) ResetCredential(ctx context.Context, req *authenticationV1.ResetCredentialRequest) error {
	if req.GetNeedDecrypt() {
		// 解密密码
		bytesPass, _ := base64.StdEncoding.DecodeString(req.GetNewCredential())
		plainPassword, _ := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
		req.NewCredential = string(plainPassword)
	}

	entity, err := r.entClient.Client().UserCredential.
		Query().
		Select(
			usercredential.FieldCredentialType,
		).
		Where(
			usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
			usercredential.IdentifierEQ(req.GetIdentifier()),
		).
		Only(ctx)
	if err != nil {
		r.log.Errorf("query one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("query one data failed")
	}

	if entity.CredentialType == nil {
		return authenticationV1.ErrorNotFound("user credential not found")
	}

	var newCredential string
	newCredential, err = r.prepareCredential(entity.CredentialType, req.GetNewCredential())
	if err != nil {
		r.log.Errorf("prepare new credential failed: %s", err.Error())
		return authenticationV1.ErrorBadRequest("prepare new credential failed")
	}

	if newCredential == "" {
		return authenticationV1.ErrorBadRequest("new credential cannot be empty")
	}

	builder := r.entClient.Client().UserCredential.Update()
	builder.Where(
		usercredential.IdentityTypeEQ(*r.identityTypeConverter.ToEntity(trans.Ptr(req.GetIdentityType()))),
		usercredential.IdentifierEQ(req.GetIdentifier()),
	)
	builder.
		SetCredential(newCredential).
		SetUpdatedAt(time.Now())
	if err = builder.Exec(ctx); err != nil {
		r.log.Errorf("update one data failed: %s", err.Error())
		return authenticationV1.ErrorInternalServerError("update data failed")
	}

	return nil
}
