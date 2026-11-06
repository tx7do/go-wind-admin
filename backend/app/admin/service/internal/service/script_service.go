package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

// ScriptService 脚本管理服务（平台管理员）。
//
// 变更联动：Create/Update/Delete/启停成功后触发运行时 Resync，
// 使数据库变更即时生效（无需重启服务）。
type ScriptService struct {
	adminV1.ScriptServiceHTTPServer

	log     *bLogger.Helper
	repo    *data.ScriptRepo
	runtime *ScriptRuntime
}

func NewScriptService(
	ctx *bootstrap.Context,
	repo *data.ScriptRepo,
	runtime *ScriptRuntime,
) *ScriptService {
	return &ScriptService{
		log:     ctx.NewLoggerHelper("script/service/admin-service"),
		repo:    repo,
		runtime: runtime,
	}
}

func (s *ScriptService) List(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.ListScriptsResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *ScriptService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.CountScriptsResponse, error) {
	return s.repo.Count(ctx, req)
}

func (s *ScriptService) Get(ctx context.Context, req *scriptV1.GetScriptRequest) (*scriptV1.Script, error) {
	return s.repo.Get(ctx, req)
}

func (s *ScriptService) Create(ctx context.Context, req *scriptV1.CreateScriptRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.validateDraft(req.Data); err != nil {
		return nil, err
	}

	// 脚本名唯一
	name := req.Data.GetName()
	if exist, err := s.repo.IsNameExist(ctx, name, 0); err != nil {
		return nil, err
	} else if exist {
		return nil, scriptV1.ErrorConflict("script name already exists: %s", name)
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	s.resync(ctx)
	return &emptypb.Empty{}, nil
}

func (s *ScriptService) Update(ctx context.Context, req *scriptV1.UpdateScriptRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Data.Name != nil {
		if err := s.validateDraft(req.Data); err != nil {
			return nil, err
		}
		// 脚本名唯一（排除自身）
		if exist, err := s.repo.IsNameExist(ctx, req.Data.GetName(), req.GetId()); err != nil {
			return nil, err
		} else if exist {
			return nil, scriptV1.ErrorConflict("script name already exists: %s", req.Data.GetName())
		}
	}

	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}

	s.resync(ctx)
	return &emptypb.Empty{}, nil
}

func (s *ScriptService) Delete(ctx context.Context, req *scriptV1.DeleteScriptRequest) (*emptypb.Empty, error) {
	if req == nil || len(req.GetIds()) == 0 {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}

	s.resync(ctx)
	return &emptypb.Empty{}, nil
}

// TestRun 试运行脚本：已保存的按 id 取源码，或直接运行草稿。
// testRunSemaphore 试运行并发信号量（P3 加固）：TestRun 每次新建一次性引擎
// 不受全局串行化约束，并发 N 路各 5s 全速分配内存会形成内存尖峰 DoS。
var testRunSemaphore = make(chan struct{}, 4)

func (s *ScriptService) TestRun(ctx context.Context, req *scriptV1.TestRunScriptRequest) (*scriptV1.TestRunScriptResponse, error) {
	if req == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	select {
	case testRunSemaphore <- struct{}{}:
		defer func() { <-testRunSemaphore }()
	case <-ctx.Done():
		return nil, scriptV1.ErrorBadRequest("request cancelled")
	}

	var language, name, source string
	switch req.Target.(type) {
	case *scriptV1.TestRunScriptRequest_Id:
		dto, err := s.repo.Get(ctx, &scriptV1.GetScriptRequest{
			QueryBy: &scriptV1.GetScriptRequest_Id{Id: req.GetId()},
		})
		if err != nil {
			return nil, scriptV1.ErrorNotFound("script not found")
		}
		language = strings.ToLower(dto.GetLanguage().String())
		name = dto.GetName()
		source = dto.GetSource()
	case *scriptV1.TestRunScriptRequest_Draft:
		draft := req.GetDraft()
		if err := s.validateDraft(draft); err != nil {
			return nil, err
		}
		language = strings.ToLower(draft.GetLanguage().String())
		name = draft.GetName()
		source = draft.GetSource()
	default:
		return nil, scriptV1.ErrorBadRequest("target is required (id or draft)")
	}

	// 还原输入上下文（JSON 字符串 → Go 值）
	input := make(map[string]any, len(req.GetInput()))
	for k, raw := range req.GetInput() {
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			// 非 JSON 的原样作为字符串
			v = raw
		}
		input[k] = v
	}

	resp := &scriptV1.TestRunScriptResponse{Success: true}
	data, runErr := s.runtime.TestRun(ctx, language, name, source, input)
	if runErr != nil {
		resp.Success = false
		resp.Error = runErr.Error()
	}
	if data != nil {
		resp.Context = make(map[string]string, len(data))
		for k, v := range data {
			encoded, err := json.Marshal(v)
			if err != nil {
				encoded = []byte(fmt.Sprintf("%v", v))
			}
			resp.Context[k] = string(encoded)
		}
	}

	return resp, nil
}

// ListHookPoints 列出全部已注册钩子点及引擎支持的语言。
func (s *ScriptService) ListHookPoints(ctx context.Context, _ *emptypb.Empty) (*scriptV1.ListHookPointsResponse, error) {
	resp := &scriptV1.ListHookPointsResponse{
		Languages: s.runtime.Languages(),
	}

	hooks := s.runtime.HookPoints()
	resp.Items = make([]*scriptV1.HookPoint, 0, len(hooks))
	for _, h := range hooks {
		resp.Items = append(resp.Items, &scriptV1.HookPoint{
			Name:        h.Name,
			Description: h.Description,
			ScriptCount: uint32(h.ScriptCount + h.CallbackCount),
		})
	}
	return resp, nil
}

// validateDraft 校验脚本基础字段（名称/源码必填，语言受支持）。
func (s *ScriptService) validateDraft(draft *scriptV1.Script) error {
	if draft == nil {
		return scriptV1.ErrorBadRequest("invalid parameter")
	}
	if strings.TrimSpace(draft.GetName()) == "" {
		return scriptV1.ErrorBadRequest("script name is required")
	}
	if strings.TrimSpace(draft.GetSource()) == "" {
		return scriptV1.ErrorBadRequest("script source is required")
	}
	if s.runtime.EngineFor(strings.ToLower(draft.GetLanguage().String())) == nil {
		return scriptV1.ErrorBadRequest("unsupported script language: %s", draft.GetLanguage().String())
	}
	return nil
}

// resync 触发本实例全量重同步，并通知其他实例同样重同步
// （尽力而为：失败记日志不影响管理操作本身）。
func (s *ScriptService) resync(ctx context.Context) {
	if err := s.runtime.Resync(ctx); err != nil {
		s.log.Errorf(ctx, "script runtime resync after change failed: %v", err)
	}
	s.runtime.NotifyResync(ctx)
}
