// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件收录 AI 会话/消息/Agent 应用/Agent 运行四仓的 ForTest 构造器
//     （ai_conversation / ai_message / ai_agent_app / ai_agent_run 对应的
//     *_repo_sqlite_test.go 使用；ai_provider 与 ai_usage_log 的 ForTest 构造器
//     已在 repo_testkit6.go 收录，此处不重复导出）。各构造器与对应 *_repo.go 的
//     生产构造器逐字段对齐（含 mapper/converter 初始化与 init() 调用），唯一差异
//     是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；entClient 由调用方
//     传入（测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentrun"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

// NewAiConversationRepoForTest 逐字段复刻 ai_conversation_repo.go 的
// NewAiConversationRepo（log 换 NopLogger）。
func NewAiConversationRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AiConversationRepo {
	repo := &AiConversationRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiConversation, ent.AiConversation](),
	}

	repo.init()

	return repo
}

// NewAiMessageRepoForTest 逐字段复刻 ai_message_repo.go 的 NewAiMessageRepo
// （log 换 NopLogger）。
func NewAiMessageRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AiMessageRepo {
	repo := &AiMessageRepo{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		entClient:     entClient,
		mapper:        mapper.NewCopierMapper[aiV1.AiMessage, ent.AiMessage](),
		roleConverter: mapper.NewEnumTypeConverter[aiV1.AiRole, aimessage.Role](aiV1.AiRole_name, aiV1.AiRole_value),
	}

	repo.init()

	return repo
}

// NewAiAgentAppRepoForTest 逐字段复刻 ai_agent_app_repo.go 的 NewAiAgentAppRepo
// （log 换 NopLogger）。
func NewAiAgentAppRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AiAgentAppRepo {
	repo := &AiAgentAppRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiAgentApp, ent.AiAgentApp](),
	}
	repo.init()
	return repo
}

// NewAiAgentRunRepoForTest 逐字段复刻 ai_agent_run_repo.go 的 NewAiAgentRunRepo
// （log 换 NopLogger）。
func NewAiAgentRunRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AiAgentRunRepo {
	repo := &AiAgentRunRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiAgentRun, ent.AiAgentRun](),
		statusConverter: mapper.NewEnumTypeConverter[aiV1.AiAgentRun_Status, aiagentrun.Status](
			aiV1.AiAgentRun_Status_name,
			aiV1.AiAgentRun_Status_value,
		),
	}
	repo.init()
	return repo
}

// NewAiAgentRevisionRepoForTest 逐字段复刻 ai_agent_revision_repo.go 的
// NewAiAgentRevisionRepo（log 换 NopLogger；薄仓无 mapper/converter）。
func NewAiAgentRevisionRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *AiAgentRevisionRepo {
	return &AiAgentRevisionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}
