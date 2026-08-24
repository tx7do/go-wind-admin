package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type UserCredentialService struct {
	authenticationV1.UnimplementedUserCredentialServiceServer

	log *bLogger.Helper

	userCredentialsRepo *data.UserCredentialRepo
}

func NewUserCredentialService(
	ctx *bootstrap.Context,
	userCredentialRepo *data.UserCredentialRepo,
) *UserCredentialService {
	return &UserCredentialService{
		log:                 ctx.NewLoggerHelper("user-credential/service/admin-service"),
		userCredentialsRepo: userCredentialRepo,
	}
}

func (s *UserCredentialService) List(ctx context.Context, req *paginationV1.PagingRequest) (*authenticationV1.ListUserCredentialResponse, error) {
	return s.userCredentialsRepo.List(ctx, req)
}

func (s *UserCredentialService) Get(ctx context.Context, req *authenticationV1.GetUserCredentialRequest) (*authenticationV1.UserCredential, error) {
	return s.userCredentialsRepo.Get(ctx, req)
}

func (s *UserCredentialService) GetByIdentifier(ctx context.Context, req *authenticationV1.GetUserCredentialByIdentifierRequest) (*authenticationV1.UserCredential, error) {
	return s.userCredentialsRepo.GetByIdentifier(ctx, req)
}

func (s *UserCredentialService) Create(ctx context.Context, req *authenticationV1.CreateUserCredentialRequest) (*emptypb.Empty, error) {
	if err := s.userCredentialsRepo.Create(ctx, req); err != nil {
		// s.log.Info(ctx, err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *UserCredentialService) Update(ctx context.Context, req *authenticationV1.UpdateUserCredentialRequest) (*emptypb.Empty, error) {
	if err := s.userCredentialsRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *UserCredentialService) Delete(ctx context.Context, req *authenticationV1.DeleteUserCredentialRequest) (*emptypb.Empty, error) {
	if err := s.userCredentialsRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *UserCredentialService) VerifyCredential(ctx context.Context, req *authenticationV1.VerifyCredentialRequest) (*authenticationV1.VerifyCredentialResponse, error) {
	return s.userCredentialsRepo.VerifyCredential(ctx, req)
}

func (s *UserCredentialService) ChangeCredential(ctx context.Context, req *authenticationV1.ChangeCredentialRequest) (*emptypb.Empty, error) {
	err := s.userCredentialsRepo.ChangeCredential(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *UserCredentialService) ResetCredential(ctx context.Context, req *authenticationV1.ResetCredentialRequest) (*emptypb.Empty, error) {
	err := s.userCredentialsRepo.ResetCredential(ctx, req)
	return &emptypb.Empty{}, err
}
