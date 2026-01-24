package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	fileV1 "go-wind-admin/api/gen/go/file/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/oss"
)

type FileService struct {
	adminV1.FileServiceHTTPServer

	log *log.Helper

	fileRepo *data.FileRepo
	mc       *oss.MinIOClient
}

func NewFileService(
	ctx *bootstrap.Context,
	fileRepo *data.FileRepo,
	mc *oss.MinIOClient,
) *FileService {
	return &FileService{
		log:      ctx.NewLoggerHelper("file/service/admin-service"),
		fileRepo: fileRepo,
		mc:       mc,
	}
}

func (s *FileService) List(ctx context.Context, req *paginationV1.PagingRequest) (*fileV1.ListFileResponse, error) {
	return s.fileRepo.List(ctx, req)
}

func (s *FileService) Get(ctx context.Context, req *fileV1.GetFileRequest) (*fileV1.File, error) {
	return s.fileRepo.Get(ctx, req)
}

func (s *FileService) Create(ctx context.Context, req *fileV1.CreateFileRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.fileRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *FileService) Update(ctx context.Context, req *fileV1.UpdateFileRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.fileRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *FileService) Delete(ctx context.Context, req *fileV1.DeleteFileRequest) (*emptypb.Empty, error) {
	f, err := s.fileRepo.Get(ctx, &fileV1.GetFileRequest{
		QueryBy: &fileV1.GetFileRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	if err = s.fileRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if err = s.mc.DeleteFile(ctx,
		f.GetBucketName(),
		f.GetFileDirectory()+"/"+f.GetSaveFileName(),
	); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
