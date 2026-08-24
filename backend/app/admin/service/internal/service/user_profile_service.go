package service

import (
	"context"
	"encoding/base64"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
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
	mc                 *oss.MinIOClient

	log *bLogger.Helper
}

func NewUserProfileService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	roleRepo *data.RoleRepo,
	userCredentialRepo *data.UserCredentialRepo,
	mc *oss.MinIOClient,
) *UserProfileService {
	return &UserProfileService{
		log:                ctx.NewLoggerHelper("user-profile/service/admin-service"),
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userCredentialRepo: userCredentialRepo,
		mc:                 mc,
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

func (s *UserProfileService) UpdateUser(ctx context.Context, req *identityV1.UpdateUserRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(operator.UserId)
	req.Id = operator.UserId

	if err = s.userRepo.Update(ctx, req); err != nil {
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

	err = s.userCredentialRepo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    operator.GetUsername(),
		OldCredential: req.GetOldPassword(),
		NewCredential: req.GetNewPassword(),
	})
	return &emptypb.Empty{}, err
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
		if int64(len(imageBytes)) > oss.MaxUploadSize {
			return nil, authenticationV1.ErrorBadRequest("avatar exceeds max size")
		}
		// 嗅探真实图片类型并校验白名单
		realMime, _ := oss.DetectFileType(imageBytes)
		if !oss.IsAllowedMimeType(realMime) || len(realMime) < 6 || realMime[:6] != "image/" {
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

// BindContact 绑定手机号码/邮箱
func (s *UserProfileService) BindContact(context.Context, *identityV1.BindContactRequest) (*emptypb.Empty, error) {
	return nil, nil
}

// VerifyContact 验证手机号码/邮箱
func (s *UserProfileService) VerifyContact(context.Context, *identityV1.VerifyContactRequest) (*emptypb.Empty, error) {
	return nil, nil
}
