package service

import (
	"context"
	"math"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	printV1 "go-wind-admin/api/gen/go/print/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/printrender"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// printRenderMaxHTML 服务端 PDF 渲染的 HTML 上限：前端内核产物（多行合并批量）通常
// 几百 KB 级，超限视为滥用直接拒绝，防 headless Chrome 被超大文档拖垮。
const printRenderMaxHTML = 8 << 20 // 8 MiB

// printRenderSem 渲染并发信号量（进程级）：每请求一个 Chrome tab，无并发限制时
// 循环提交大文档可耗尽 CPU/内存（60s 超时 × 无限并发）。
var printRenderSem = make(chan struct{}, 2)

func (s *PrintTemplateService) RenderPdf(ctx context.Context, req *printV1.RenderPrintTemplateRequest) (*printV1.RenderPrintTemplateResponse, error) {
	if req == nil || req.GetHtml() == "" {
		return nil, adminV1.ErrorBadRequest("invalid parameter: html is required")
	}
	if req.GetPaperWidthMm() <= 0 || req.GetPaperHeightMm() <= 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter: paper size (mm) is required")
	}
	if len(req.GetHtml()) > printRenderMaxHTML {
		return nil, adminV1.ErrorBadRequest("html too large")
	}

	// P1-11 修复：清洗后再送 Chromium——此前任意用户 HTML 直送 headless Chrome，
	// <iframe src=http://169.254.169.254/> 的响应会渲染进 PDF 返还（可读 SSRF 原语），
	// <script> 在无 CSP 的 about:blank 照常执行。清洗剥除脚本面并拒收私网/元数据
	// 资源 URL；发现被禁内容直接 400（不静默改写，避免渲染结果悄悄变样）。
	cleanHTML, err := sanitizePrintHTML(req.GetHtml())
	if err != nil {
		s.log.Warnf(ctx, "print template html blocked: %s", err.Error())
		return nil, adminV1.ErrorBadRequest("html contains blocked content: %v", err)
	}

	select {
	case printRenderSem <- struct{}{}:
		defer func() { <-printRenderSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	pdf, err := printrender.RenderPDF(ctx, printrender.Options{
		HTML:          cleanHTML,
		PaperWidthMm:  req.GetPaperWidthMm(),
		PaperHeightMm: req.GetPaperHeightMm(),
		Landscape:     req.GetLandscape(),
		Background:    req.GetBackground(),
	})
	if err != nil {
		s.log.Errorf(ctx, "render print template pdf failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("render pdf failed")
	}

	filename := req.GetFilename()
	if filename == "" {
		filename = "print"
	}
	filename += ".pdf"

	return &printV1.RenderPrintTemplateResponse{
		Pdf:      pdf,
		Filename: filename,
	}, nil
}

type PrintTemplateService struct {
	adminV1.PrintTemplateServiceHTTPServer

	log               *bLogger.Helper
	printTemplateRepo *data.PrintTemplateRepo
	aiProviderRepo    *data.AiProviderRepo
	aiUsageRepo       *data.AiUsageLogRepo
	tenantUsageRepo   *data.TenantUsageRepo
}

func NewPrintTemplateService(
	ctx *bootstrap.Context,
	printTemplateRepo *data.PrintTemplateRepo,
	aiProviderRepo *data.AiProviderRepo,
	aiUsageRepo *data.AiUsageLogRepo,
	tenantUsageRepo *data.TenantUsageRepo,
) *PrintTemplateService {
	svc := &PrintTemplateService{
		log:               ctx.NewLoggerHelper("print-template/service/admin-service"),
		printTemplateRepo: printTemplateRepo,
		aiProviderRepo:    aiProviderRepo,
		aiUsageRepo:       aiUsageRepo,
		tenantUsageRepo:   tenantUsageRepo,
	}

	svc.init()

	return svc
}

// init 播种内置打印模板（通用单据/标签基线版式）。与其他默认数据一致，
// 在服务构造（进程启动）时执行一次；SeedDefaults 按编码缺一补一、不覆盖既有行。
func (s *PrintTemplateService) init() {
	ctx := viewer.WithSystemContext(context.Background())
	if err := s.printTemplateRepo.SeedDefaults(ctx, constants.DefaultPrintTemplates); err != nil {
		s.log.Errorf(ctx, "seed default print templates failed: %s", err.Error())
	}
}

func (s *PrintTemplateService) List(ctx context.Context, req *paginationV1.PagingRequest) (*printV1.ListPrintTemplateResponse, error) {
	return s.printTemplateRepo.List(ctx, req)
}

func (s *PrintTemplateService) Get(ctx context.Context, req *printV1.GetPrintTemplateRequest) (*printV1.PrintTemplate, error) {
	return s.printTemplateRepo.Get(ctx, req)
}

func (s *PrintTemplateService) Create(ctx context.Context, req *printV1.CreatePrintTemplateRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 模板数配额检查（资源计数类，fail-open：仅租户上下文且配额已配置时生效）
	targetTenantID := operator.GetTenantId()
	if s.tenantUsageRepo != nil && targetTenantID > 0 {
		usage, guErr := s.tenantUsageRepo.GetUsage(ctx, targetTenantID)
		if guErr == nil {
			if limit, ok := quotaLimitFromUsages(usage.GetQuotas(), identityV1.PlanQuota_TEMPLATE_COUNT); ok {
				current := s.countTemplatesInTenant(ctx, targetTenantID)
				if uint64(current) >= limit {
					return nil, quotaExceededError("TEMPLATE_COUNT", limit, uint64(current))
				}
			}
		}
		// GetUsage 失败 → fail-open 放行
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.printTemplateRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// countTemplatesInTenant 统计指定租户的打印模板数（builtin 行 tenant_id=0 不计入）。
// 计数失败返回 math.MaxInt64 → 触发配额拒绝（fail-closed），但调用方只在 limit>0
// 时才走计数路径——未配置配额的场景不会调到这里。
func (s *PrintTemplateService) countTemplatesInTenant(ctx context.Context, tenantID uint32) int64 {
	n, err := s.printTemplateRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		s.log.Errorf(ctx, "count print templates for tenant [%d] failed: %s", tenantID, err.Error())
		return math.MaxInt64
	}
	return n
}

func (s *PrintTemplateService) Update(ctx context.Context, req *printV1.UpdatePrintTemplateRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.printTemplateRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PrintTemplateService) Delete(ctx context.Context, req *printV1.DeletePrintTemplateRequest) (*emptypb.Empty, error) {
	if err := s.printTemplateRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
