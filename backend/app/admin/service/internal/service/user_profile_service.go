package service

import (
	"context"
	"encoding/base64"

	"github.com/tx7do/go-utils/ossutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/oss"
)

type UserProfileService struct {
	adminV1.UserProfileServiceHTTPServer

	userRepo           data.UserRepo
	roleRepo           *data.RoleRepo
	userCredentialRepo *data.UserCredentialRepo
	authenticator      *data.Authenticator
	// notifier 是唯一的对外通知出口（邮箱绑定验证码邮件），渠道选择与 SMTP 细节不外泄。
	notifier   Notifier
	vcodeCache *data.VCodeCache
	mc         *oss.MinIOClient

	log    *bLogger.Helper
	mailer *TransactionMailer
}

func NewUserProfileService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	roleRepo *data.RoleRepo,
	userCredentialRepo *data.UserCredentialRepo,
	authenticator *data.Authenticator,
	notifier Notifier,
	vcodeCache *data.VCodeCache,
	mc *oss.MinIOClient,
	mailer *TransactionMailer,
) *UserProfileService {
	return &UserProfileService{
		log:                ctx.NewLoggerHelper("user-profile/service/admin-service"),
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userCredentialRepo: userCredentialRepo,
		authenticator:      authenticator,
		notifier:           notifier,
		vcodeCache:         vcodeCache,
		mc:                 mc,
		mailer:             mailer,
	}
}

func (s *UserProfileService) GetUser(ctx context.Context, _ *emptypb.Empty) (*identityV1.User, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "查询用户失败[%s]", err.Error())
		return nil, authenticationV1.ErrorNotFound("user not found")
	}

	roleCodes, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, user.GetRoleIds())
	if err != nil {
		s.log.Errorf(ctx, "get user role codes failed [%s]", err.Error())
	}
	if roleCodes != nil {
		user.Roles = roleCodes
	}

	return user, err
}

// UpdateUser 自服务资料更新（/me）。
//
// /me 是全仓唯一"任何登录用户都能调"的写入口，而它复用的 UpdateUserRequest.Data 是完整
// User 消息（含 RoleIds/Status/LockedUntil/TenantId 等特权字段），repo 层"无 field-mask
// 视为全量更新"（user_repo.go maskPathInUpdate）。此前无字段白名单，任意用户可通过
// {"data":{"roleId":1}} 给自己绑定任意角色（platform:admin 种子数据恒为 id=1），刷新令牌
// 后 fillAdminFlags 置 IsPlatformAdmin，全部平台闸门形同虚设。故这里显式重建仅含资料
// 字段的 Data 并强制 field-mask，关联/状态/租户字段一律剥离——对齐 DeleteAvatar/
// UploadAvatar 的显式 mask 写法。另有 repo 层 AssignUserRole(s) 的角色归属校验兜底
// （见 user_role_repo.go validateRoleBindingOwnership）。
func (s *UserProfileService) UpdateUser(ctx context.Context, req *identityV1.UpdateUserRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	data := &identityV1.User{
		Id: trans.Ptr(operator.UserId),
	}
	mask := &field_mask.FieldMask{}
	addField := func(path string, set bool) {
		if set {
			mask.Paths = append(mask.Paths, path)
		}
	}
	if src := req.GetData(); src != nil {
		data.Nickname = src.Nickname
		addField("nickname", src.Nickname != nil)
		data.Realname = src.Realname
		addField("realname", src.Realname != nil)
		data.Avatar = src.Avatar
		addField("avatar", src.Avatar != nil)
		data.Email = src.Email
		addField("email", src.Email != nil)
		data.Mobile = src.Mobile
		addField("mobile", src.Mobile != nil)
		data.Telephone = src.Telephone
		addField("telephone", src.Telephone != nil)
		data.Gender = src.Gender
		addField("gender", src.Gender != nil)
		data.Region = src.Region
		addField("region", src.Region != nil)
		data.Locale = src.Locale
		addField("locale", src.Locale != nil)
		data.Address = src.Address
		addField("address", src.Address != nil)
		data.Description = src.Description
		addField("description", src.Description != nil)
		data.Remark = src.Remark
		addField("remark", src.Remark != nil)
	}

	if err = s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Id:         operator.UserId,
		Data:       data,
		UpdateMask: mask,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *UserProfileService) ChangePassword(ctx context.Context, req *identityV1.ChangePasswordRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 前端与登录/注册一致走 AES 密文传输，必须解密后再校验/哈希——
	// 此前漏设 NeedDecrypt，密文串被当明文哈希入库（改完即锁号）且绕过复杂度校验。
	err = s.userCredentialRepo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    operator.GetUsername(),
		OldCredential: req.GetOldPassword(),
		NewCredential: req.GetNewPassword(),
		NeedDecrypt:   true,
	})
	if err != nil {
		return nil, err
	}

	// 改密成功后吊销本人全部客户端类型的令牌（含当前会话），
	// 防止凭据泄露后旧令牌继续可用；前端在成功回调中引导重新登录。
	if err = s.authenticator.RevokeUserTokenAllClientTypes(ctx, operator.GetUserId()); err != nil {
		s.log.Errorf(ctx, "revoke tokens after password change failed for user [%d]: %v", operator.GetUserId(), err)
	}

	return &emptypb.Empty{}, nil
}

// DeleteAvatar 删除头像
func (s *UserProfileService) DeleteAvatar(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Data: &identityV1.User{
			Id:     trans.Ptr(operator.UserId),
			Avatar: trans.Ptr(""),
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"avatar"},
		},
	}); err != nil {
		s.log.Errorf(ctx, "delete user avatar failed [%s]", err.Error())
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// UploadAvatar 上传头像
func (s *UserProfileService) UploadAvatar(ctx context.Context, req *identityV1.UploadAvatarRequest) (*identityV1.UploadAvatarResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var avatarURL string
	switch req.GetSource().(type) {
	case *identityV1.UploadAvatarRequest_ImageBase64:
		// 解码 base64 图片数据
		imageBytes, derr := base64.StdEncoding.DecodeString(req.GetImageBase64())
		if derr != nil {
			s.log.Errorf(ctx, "decode avatar base64 failed [%s]", derr.Error())
			return nil, authenticationV1.ErrorBadRequest("invalid avatar base64 data")
		}
		if len(imageBytes) == 0 {
			return nil, authenticationV1.ErrorBadRequest("empty avatar data")
		}
		// 校验图片大小（复用上传限制，头像不应超过该上限）
		if int64(len(imageBytes)) > ossutil.MaxUploadSize {
			return nil, authenticationV1.ErrorBadRequest("avatar exceeds max size")
		}
		// 嗅探真实图片类型并校验白名单
		realMime, _ := ossutil.DetectFileType(imageBytes)
		if !ossutil.IsAllowedMimeType(realMime) || len(realMime) < 6 || realMime[:6] != "image/" {
			return nil, authenticationV1.ErrorBadRequest("only image files are allowed for avatar")
		}
		// 上传到 OSS（mc 自动嗅探 MIME/桶/对象名，头像统一进 images 桶）
		_, _, downloadUrl, uerr := s.mc.UploadFile(ctx, "", "", realMime, imageBytes)
		if uerr != nil {
			s.log.Errorf(ctx, "upload avatar to oss failed [%s]", uerr.Error())
			return nil, authenticationV1.ErrorInternalServerError("upload avatar failed")
		}
		avatarURL = downloadUrl
	case *identityV1.UploadAvatarRequest_ImageUrl:
		avatarURL = req.GetImageUrl()
	default:
		s.log.Errorf(ctx, "upload avatar failed, invalid avatar source")
		return nil, authenticationV1.ErrorBadRequest("invalid avatar source")
	}

	if err = s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Data: &identityV1.User{
			Id:     trans.Ptr(operator.UserId),
			Avatar: trans.Ptr(avatarURL),
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"avatar"},
		},
	}); err != nil {
		s.log.Errorf(ctx, "delete user avatar failed [%s]", err.Error())
		return nil, err
	}

	return &identityV1.UploadAvatarResponse{
		Url: avatarURL,
	}, nil
}
