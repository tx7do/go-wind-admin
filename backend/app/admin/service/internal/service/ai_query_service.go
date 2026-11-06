package service

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-crud/viewer"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── 智能问数安全边界 ────────────────────────────────────────────────

const (
	// aiQueryRowLimit 结果行硬上限（缺 LIMIT 自动补；超限的显式 LIMIT 也被改写）。
	aiQueryRowLimit = 100

	// aiQueryCallTimeout SQL 生成与结论生成的单次 LLM 调用兜底时长。
	aiQueryCallTimeout = 90 * time.Second
)

// aiQueryTableWhitelist 平台表白名单：NL→SQL 只允许触达这些表（平台用户）。
// 键为表名，值为给 LLM 的列说明（表名+列清单+备注一起拼进提示词当数据字典）。
var aiQueryTableWhitelist = map[string]string{
	"sys_users":                   "用户表：id, username(登录名), nickname(昵称), email, mobile, status(ON=启用/OFF=停用), created_at(注册时间)；用户与角色是多对多，经 sys_user_roles 关联，sys_users 表没有 role_id 列",
	"sys_roles":                   "角色表（平台+租户两级，经 tenant_id 区分）：id, name(角色名), code(角色编码), status, created_at",
	"sys_user_roles":              "用户-角色关联表：user_id, role_id（查每个角色的用户数：JOIN sys_user_roles ON sys_user_roles.role_id = sys_roles.id 再 JOIN sys_users ON sys_user_roles.user_id = sys_users.id）",
	"sys_tenants":                 "租户表：id, code(租户编码), name(租户名), plan_id(套餐ID), status, created_at",
	"sys_plans":                   "套餐表：id, name(套餐名)",
	"sys_operation_audit_logs":    "操作审计日志：id, user_id, username, action(CREATE/UPDATE/DELETE/READ/ASSIGN/UNASSIGN/EXPORT/IMPORT/OTHER), resource_type(资源类型), resource_id, success(bool), failure_reason, created_at(操作时间)",
	"sys_login_audit_logs":        "登录审计日志：id, user_id, username, status(SUCCESS/FAILED/LOCKED), ip_address, created_at",
	"sys_ai_usage_logs":           "AI 用量流水：id, user_id, tenant_id, model_name, prompt_tokens, completion_tokens, total_tokens, duration_ms, created_at",
	"internal_messages":           "站内信消息：id, title, status(PUBLISHED/…), created_at",
	"internal_message_recipients": "站内信收件记录：id, message_id, recipient_user_id(收件人), status(RECEIVED=未读/READ=已读), created_at",
}

// aiQueryTenantTableWhitelist 租户表白名单：只含带 tenant_id 列、可安全暴露给
// 租户自助分析的本租户数据表（平台级表如 sys_tenants/sys_plans 不在其中）。
var aiQueryTenantTableWhitelist = map[string]string{
	"sys_users":                   "用户表（本租户）：id, username(登录名), nickname(昵称), email, mobile, status(ON=启用/OFF=停用), created_at(注册时间)",
	"sys_roles":                   "角色表（本租户）：id, name(角色名), code(角色编码), status, created_at",
	"sys_operation_audit_logs":    "操作审计日志（本租户）：id, user_id, username, action(CREATE/UPDATE/DELETE/READ/…), resource_type(资源类型), success(bool), failure_reason, created_at(操作时间)",
	"sys_login_audit_logs":        "登录审计日志（本租户）：id, user_id, username, status(SUCCESS/FAILED/LOCKED), ip_address, created_at",
	"sys_ai_usage_logs":           "AI 用量流水（本租户）：id, user_id, model_name, prompt_tokens, completion_tokens, total_tokens, duration_ms, created_at",
	"internal_message_recipients": "站内信收件记录（本租户）：id, message_id, recipient_user_id(收件人), status(RECEIVED=未读/READ=已读), created_at",
}

// tenantIDPatternFor 构造"tenant_id = {id}"存在性校验的正则（容忍别名与空白）。
func tenantIDPatternFor(tenantID uint32) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\btenant_id\s*=\s*` + fmt.Sprintf("%d", tenantID) + `\b`)
}

// rejectDisjunctionOrComments 租户路径的析取/注释拒绝：
//   - 字符串字面量外的结构 OR（析取可让"tenant_id = 5"出现性检查形同虚设）；
//   - 任何 SQL 注释——"-- tenant_id = 5" 式注释能骗过存在性正则而谓词实际不生效。
func rejectDisjunctionOrComments(sqlText string) error {
	stripped, hasComment := stripSQLStringsAndComments(sqlText)
	if hasComment {
		return fmt.Errorf("sql comments are not allowed for tenant queries")
	}
	if regexp.MustCompile(`(?i)\bor\b`).MatchString(stripped) {
		return fmt.Errorf("disjunction (OR) is not allowed for tenant queries")
	}
	return nil
}

// stripSQLStringsAndComments 去掉单引号字符串字面量与注释，留下 SQL 结构骨架。
// 返回值 hasComment 标记原文是否含注释（注释本身是拒绝理由，不能只是剥掉）。
func stripSQLStringsAndComments(sqlText string) (string, bool) {
	var b strings.Builder
	inString := false
	hasComment := false
	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		switch {
		case inString:
			if ch == '\'' {
				if i+1 < len(sqlText) && sqlText[i+1] == '\'' {
					i++ // '' 转义仍在字符串内
					continue
				}
				inString = false
			}
		case ch == '\'':
			inString = true
		case ch == '-' && i+1 < len(sqlText) && sqlText[i+1] == '-':
			hasComment = true
			for i < len(sqlText) && sqlText[i] != '\n' {
				i++
			}
		case ch == '/' && i+1 < len(sqlText) && sqlText[i+1] == '*':
			hasComment = true
			i += 2
			for i+1 < len(sqlText) && !(sqlText[i] == '*' && sqlText[i+1] == '/') {
				i++
			}
			i++
		default:
			b.WriteByte(ch)
		}
	}
	return b.String(), hasComment
}

// aiQueryForbiddenPatterns 生成 SQL 的硬拒绝模式：写操作与危险语句。
var aiQueryForbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|TRUNCATE|ALTER|CREATE|GRANT|REVOKE|COPY|VACUUM|REINDEX|CALL|DO|SET)\b`),
	regexp.MustCompile(`(?i)\b(pg_read_file|pg_ls_dir|pg_sleep|lo_import|lo_export|dblink)\b`),
	regexp.MustCompile(`\;`),     // 多语句
	regexp.MustCompile(`--|/\*`), // 注释（防注释截断绕过）
}

// aiQuerySelectPattern 必须以 SELECT 或 WITH 开头（含前导空白/括号容忍）。
var aiQuerySelectPattern = regexp.MustCompile(`(?is)^\s*\(?\s*(SELECT|WITH)\b`)

// ── 服务 ───────────────────────────────────────────────────────────

// AiQueryService 智能问数：NL → 只读 SQL → 结构化结果 →（可选）自然语言结论。
// 平台管理员专属——第一版查询的是全平台数据，且 SQL 直触原生库表。
type AiQueryService struct {
	adminV1.AiQueryServiceHTTPServer
	log *bLogger.Helper

	providerRepo     *data.AiProviderRepo
	usageRepo        *data.AiUsageLogRepo
	entClient        *entCrud.EntClient[*ent.Client]
	internalMessages *InternalMessageService
	internalMsgRepo  *data.InternalMessageRepo
}

func NewAiQueryService(
	ctx *bootstrap.Context,
	providerRepo *data.AiProviderRepo,
	usageRepo *data.AiUsageLogRepo,
	entClient *entCrud.EntClient[*ent.Client],
	internalMessages *InternalMessageService,
	internalMsgRepo *data.InternalMessageRepo,
) *AiQueryService {
	return &AiQueryService{
		log:              ctx.NewLoggerHelper("ai_query/service/admin-service"),
		providerRepo:     providerRepo,
		usageRepo:        usageRepo,
		entClient:        entClient,
		internalMessages: internalMessages,
		internalMsgRepo:  internalMsgRepo,
	}
}

// Ask 智能问数主链路：NL→SQL（第一次调用）→ 护栏校验 → 只读执行 → NL 结论（第二次调用）。
// 两次调用都复用脚本 ai 模块的 provider 解析与用量记账。
func (s *AiQueryService) Ask(ctx context.Context, req *aiV1.AskAiQueryRequest) (*aiV1.AskAiQueryResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// P1 修复：Ask 的两次 LLM 调用此前不记用量、不受 AI_TOKENS 配额约束
	//（chat/agent 有闸而问数没有，配额与计费口径被绕过）。接入同款月度配额。
	if err = checkAiTokenQuota(ctx, s.usageRepo, s.log, operator); err != nil {
		return nil, err
	}

	// 双白名单：平台用户查全平台表；租户用户查本租户数据表子集，
	// 且生成的 SQL 必须自带 tenant_id = 自身租户 过滤（后置校验 fail-closed）。
	tenantID := operator.GetTenantId()
	whitelist := aiQueryTableWhitelist
	tenantPattern := (*regexp.Regexp)(nil)
	if tenantID != 0 {
		whitelist = aiQueryTenantTableWhitelist
		tenantPattern = tenantIDPatternFor(tenantID)
	}
	question := strings.TrimSpace(req.GetQuestion())
	if question == "" {
		return nil, adminV1.ErrorBadRequest("question is required")
	}

	provider, err := s.providerRepo.GetEnabledDefault(ctx)
	if err != nil {
		return nil, err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	lang := req.GetLang()
	if lang == "" {
		lang = "zh-CN"
	}

	// 1. NL → SQL（携带对话历史以消解追问里的指代，如"那只看 admin 的"）
	// P1 修复：历史 Sql 原样进 assistant 消息可被攻击者当作指令样本——故意提交
	// "…WHERE tenant_id = 5 OR 1=1" 的历史让模型模仿该形态绕过护栏。历史 SQL
	// 过同款护栏，不合格的剔除 SQL 只留问题与结果摘要。
	history := make([]aiV1.AiQueryHistoryItem, 0, len(req.GetHistory()))
	for _, h := range req.GetHistory() {
		hSql := h.GetSql()
		if hSql != "" {
			if _, serr := sanitizeSQLForTenant(hSql, whitelist, tenantPattern, tenantID); serr != nil {
				hSql = ""
			}
		}
		history = append(history, aiV1.AiQueryHistoryItem{
			Question:      h.GetQuestion(),
			Sql:           hSql,
			ResultSummary: h.GetResultSummary(),
		})
	}
	sqlGenerated, genTokens, err := s.generateSQL(ctx, client, provider, question, lang, history, whitelist, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 四重护栏：只读语句 / 白名单表 / LIMIT 钳制 / 二次正则复核
	cleaned, err := sanitizeSQLForTenant(sqlGenerated, whitelist, tenantPattern, tenantID)
	if err != nil {
		return nil, adminV1.ErrorBadRequest("generated sql rejected: %v", err)
	}

	// 3. 只读事务执行
	columns, rows, execErr := s.executeReadOnly(ctx, cleaned)

	resp := &aiV1.AskAiQueryResponse{
		Sql:     cleaned,
		Columns: columns,
		Rows:    rows,
	}
	if execErr != nil {
		// 执行失败把数据库错误回给前端（便于用户改问题重试），SQL 照样返回
		s.log.Errorf(ctx, "ai query execute failed: %v", execErr)
		resp.ErrorMessage = trans.Ptr(execErr.Error())
		return resp, nil
	}
	resp.RowCount = uint32(len(rows))

	// 4.（可选）结果 → 自然语言结论
	if !req.GetWithAnswer() || len(rows) == 0 {
		return resp, nil
	}
	answer, answerTokens, err := s.generateAnswer(ctx, client, provider, question, cleaned, columns, rows, lang)
	if err != nil {
		s.log.Errorf(ctx, "ai query answer failed, returning rows only: %v", err)
	} else {
		resp.Answer = trans.Ptr(answer)
	}

	resp.TotalTokens = genTokens + answerTokens

	// P1 修复：用量落库（此前 usageRepo 注入但零调用，Ask 全程不记用量）。
	if s.usageRepo != nil && resp.TotalTokens > 0 {
		modelName := ptrStrOr(provider.ModelName, "gpt-4o-mini")
		totalTokens := uint32(resp.TotalTokens)
		if uerr := s.usageRepo.Create(ctx, &aiV1.AiUsageLog{
			UserId:      trans.Ptr(operator.GetUserId()),
			TenantId:    trans.Ptr(operator.GetTenantId()),
			ModelName:   &modelName,
			TotalTokens: &totalTokens,
		}); uerr != nil {
			s.log.Errorf(ctx, "record ai query usage failed: %v", uerr)
		}
	}

	return resp, nil
}

// generateSQL 数据字典 + 对话历史 + 问题 → 只读 SQL。
// 历史轮次作为 user/assistant 交替消息传入，让模型消解追问里的指代
// （如"那只看 admin 的"指上一轮的查询对象）；护栏在本函数之外统一执行。
func (s *AiQueryService) generateSQL(ctx context.Context, client *openai.Client, provider *ent.AiProvider, question, lang string, history []aiV1.AiQueryHistoryItem, whitelist map[string]string, tenantID uint32) (string, uint32, error) {
	var sb strings.Builder
	sb.WriteString("你是本系统的只读数据分析引擎。用户的提问永远是对上面列出的数据库表发起查询——哪怕措辞听起来像个人账号问题（如\"我的登录失败记录有哪些\"问的也是表里的数据），一律直接生成 SQL，绝不拒绝、绝不建议用户去其他平台自查。\n")
	sb.WriteString("根据表结构与用户问题，生成一条 Postgres 只读 SELECT 语句。\n")
	sb.WriteString("规则：\n")
	sb.WriteString("1. 只输出 SQL 本身，不要任何解释、不要 Markdown 代码块标记；\n")
	sb.WriteString("2. 只能使用下面列出的表；涉及多表时用 JOIN；\n")
	sb.WriteString("3. 必须带 LIMIT（最大 100）；时间比较用 created_at 与 NOW()；\n")
	sb.WriteString(fmt.Sprintf("4. 结论文字用%s；\n", langName(lang)))
	sb.WriteString("5. 用户问题可能包含对上一轮查询的指代（如\"那只看…的\"、\"换成按天分组\"），结合对话历史理解其完整含义后生成独立可执行的 SQL。\n\n")
	sb.WriteString("可用表：\n")
	for table, desc := range whitelist {
		sb.WriteString("- " + table + "：" + desc + "\n")
	}
	if tenantID != 0 {
		// 租户路径：强制谓词写进提示词（后置校验兜底，缺失即拒绝）
		sb.WriteString(fmt.Sprintf("强制规则：所有查询必须包含 tenant_id = %d 过滤条件（本租户数据边界），缺失视为无效输出。\n\n", tenantID))
	}

	// few-shot 示例放真实消息序列（上下文示例对行为的约束力远强于 system 指令）——
	// DeepSeek 实测：示例只在 system 里时，"登录失败记录"类措辞仍触发安全拒答。
	// 租户路径的示例本身带 tenant_id 谓词，教模型输出带边界的 SQL。
	exampleSQL := "SELECT id, username, ip_address, created_at FROM sys_login_audit_logs " +
		"WHERE status = 'FAILED' AND created_at >= NOW() - INTERVAL '24 hours' " +
		"ORDER BY created_at DESC LIMIT 100"
	if tenantID != 0 {
		exampleSQL = "SELECT id, username, ip_address, created_at FROM sys_login_audit_logs " +
			fmt.Sprintf("WHERE tenant_id = %d AND status = 'FAILED' AND created_at >= NOW() - INTERVAL '24 hours' ", tenantID) +
			"ORDER BY created_at DESC LIMIT 100"
	}
	messages := make([]openai.ChatCompletionMessage, 0, len(history)*2+6)
	// few-shot 1：登录失败记录（最易触发模型拒答的措辞，用示例锚定"这是查表"）
	messages = append(messages,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "最近 24 小时登录失败的记录有哪些？",
		},
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: exampleSQL,
		},
	)
	// few-shot 2：用户-角色是多对多，统计"每个角色的用户数"必须经 sys_user_roles 关联
	// （模型会臆造 sys_users.role_id 列——这是真实实测的高频错误，用示例锚定）
	messages = append(messages,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "统计每个角色的用户数量",
		},
		openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant,
			Content: "SELECT r.id AS role_id, r.name AS role_name, COUNT(ur.user_id) AS user_count " +
				"FROM sys_roles r " +
				"LEFT JOIN sys_user_roles ur ON ur.role_id = r.id " +
				"GROUP BY r.id, r.name ORDER BY user_count DESC LIMIT 100",
		},
	)
	// 下标访问而非按值 range：AiQueryHistoryItem 内嵌 protoimpl.MessageState
	// （含 sync.Mutex），按值拷贝会连带复制锁。
	for i := range history {
		// SQL 被护栏剔空的 history 项不发 assistant 消息（空 content 会被
		// provider 以 400 拒绝）——仅保留用户问题轮次，模型无前文可模仿。
		if strings.TrimSpace(history[i].GetSql()) == "" {
			messages = append(messages,
				openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: history[i].GetQuestion()},
			)
			continue
		}
		messages = append(messages,
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: history[i].GetQuestion()},
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: history[i].GetSql()},
		)
	}
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: question})

	qCtx, cancel := context.WithTimeout(ctx, aiQueryCallTimeout)
	defer cancel()
	resp, err := client.CreateChatCompletion(qCtx, openai.ChatCompletionRequest{
		Model:    ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		Messages: messages,
	})
	if err != nil {
		return "", 0, adminV1.ErrorInternalServerError("ai query generate failed: %v", err)
	}
	tokens := uint32(resp.Usage.TotalTokens)
	if len(resp.Choices) == 0 {
		return "", tokens, adminV1.ErrorInternalServerError("ai query generate empty")
	}
	raw := resp.Choices[0].Message.Content
	s.log.Infof(ctx, "ai query raw model output: %s", raw)
	return raw, tokens, nil
}

// sanitizeSQL 四重护栏（平台路径）：SQL 提取 → 只读校验 → 危险模式拒绝 → 白名单表校验 → LIMIT 钳制。
func sanitizeSQL(raw string) (string, error) {
	return sanitizeSQLFor(raw, aiQueryTableWhitelist, nil)
}

// sanitizeSQLFor 带白名单与租户谓词校验的护栏。tenantPattern 非 nil 时要求
// SQL 中出现 tenant_id = {租户ID}（租户路径 fail-closed）。
func sanitizeSQLFor(raw string, whitelist map[string]string, tenantPattern *regexp.Regexp) (string, error) {
	return sanitizeSQLForTenant(raw, whitelist, tenantPattern, 0)
}

// sanitizeSQLForTenant 同 sanitizeSQLFor，租户路径额外携带租户 ID 供结构化
// 逐块强制校验（P1-5 深层：条件数须覆盖每块白名单表引用数）。
func sanitizeSQLForTenant(raw string, whitelist map[string]string, tenantPattern *regexp.Regexp, tenantID uint32) (string, error) {
	sqlText := extractSQL(raw)
	if sqlText == "" {
		return "", fmt.Errorf("no SQL statement found in model output")
	}

	if !aiQuerySelectPattern.MatchString(sqlText) {
		return "", fmt.Errorf("only SELECT/WITH statements are allowed")
	}
	for _, pat := range aiQueryForbiddenPatterns {
		if pat.MatchString(sqlText) {
			return "", fmt.Errorf("forbidden pattern detected")
		}
	}
	// 白名单：提取 FROM/JOIN 后的表名逐一校验；模型偶发丢 sys_ 前缀或写单复数，
	// 依候选序列纠正（sys_ 前缀 / 复数 s / 两者组合）
	tablePattern := regexp.MustCompile(`(?i)\b(FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := tablePattern.FindAllStringSubmatch(sqlText, -1)
	for _, m := range matches {
		table := strings.ToLower(m[2])
		if _, ok := whitelist[table]; ok {
			continue
		}
		corrected, ok := correctTableName(table, whitelist)
		if !ok {
			return "", fmt.Errorf("table %q is not in the whitelist", m[2])
		}
		// 按词边界改写（\b 视下划线为词字符，不会误伤 sys_roles 内部的 roles）
		fixPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(table) + `\b`)
		sqlText = fixPattern.ReplaceAllStringFunc(sqlText, func(string) string {
			return corrected
		})
	}
	if tenantPattern != nil {
		if !tenantPattern.MatchString(sqlText) {
			return "", fmt.Errorf("missing mandatory tenant_id filter")
		}
		// P1 修复：租户路径仅允许合取谓词——"tenant_id = 5 OR 1=1" 这类析取让
		// 出现性检查形同虚设；注释（-- 与块注释）可把谓词移出有效过滤面，一并拒绝。
		if err := rejectDisjunctionOrComments(sqlText); err != nil {
			return "", err
		}
		// P1-5 深层：结构化逐块强制——多表 JOIN 只过滤一张、子查询内表不过滤
		// 等出现性穿透形态在此拒绝（每块条件数须覆盖该块白名单表引用数）。
		stripped, _ := stripSQLStringsAndComments(sqlText)
		if err := enforcePerBlockTenantFilter(stripped, whitelist, tenantID); err != nil {
			return "", err
		}
	}
	// LIMIT 钳制：无 LIMIT 补 aiQueryRowLimit；LIMIT N>N 上限改写
	lower := strings.ToLower(sqlText)
	if !regexp.MustCompile(`(?i)\blimit\s+\d+`).MatchString(lower) {
		sqlText = fmt.Sprintf("SELECT * FROM (%s) AS _q LIMIT %d", sqlText, aiQueryRowLimit)
	} else {
		reLimit := regexp.MustCompile(`(?i)\blimit\s+(\d+)`)
		sqlText = reLimit.ReplaceAllStringFunc(sqlText, func(m string) string {
			n := 0
			fmt.Sscanf(strings.TrimSpace(m[5:]), "%d", &n)
			if n > aiQueryRowLimit {
				return fmt.Sprintf("LIMIT %d", aiQueryRowLimit)
			}
			return m
		})
	}
	return sqlText, nil
}

// executeReadOnly 只读事务执行（数据库层兜底：READ ONLY 事务里写操作必败）。
func (s *AiQueryService) executeReadOnly(ctx context.Context, sqlText string) ([]string, []*aiV1.AiQueryRow, error) {
	db := s.entClient.DB()
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(txCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, fmt.Errorf("begin read-only tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	resultRows, err := tx.QueryContext(txCtx, sqlText)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resultRows.Close() }()

	columns, err := resultRows.Columns()
	if err != nil {
		return nil, nil, err
	}

	out := make([]*aiV1.AiQueryRow, 0, 16)
	for resultRows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err = resultRows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := &aiV1.AiQueryRow{Values: make([]string, 0, len(columns))}
		for _, v := range values {
			row.Values = append(row.Values, stringifyCellValue(v))
		}
		out = append(out, row)
	}
	if err = resultRows.Err(); err != nil {
		return nil, nil, err
	}
	return columns, out, nil
}

// generateAnswer 结果集 → 自然语言结论（第二次调用）。
func (s *AiQueryService) generateAnswer(ctx context.Context, client *openai.Client, provider *ent.AiProvider, question, sqlText string, columns []string, rows []*aiV1.AiQueryRow, lang string) (string, uint32, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("用户问题：%s\n\n执行的 SQL：%s\n\n查询结果（列：%s）：\n", question, sqlText, strings.Join(columns, ", ")))
	maxRows := len(rows)
	if maxRows > 20 {
		maxRows = 20 // 结论只看前 20 行，避免超长
	}
	for _, r := range rows[:maxRows] {
		sb.WriteString(strings.Join(r.Values, " | "))
		sb.WriteString("\n")
	}

	langLine := "用中文回答。"
	if strings.HasPrefix(lang, "en") {
		langLine = "Answer in English."
	}

	aCtx, cancel := context.WithTimeout(ctx, aiQueryCallTimeout)
	defer cancel()
	resp, err := client.CreateChatCompletion(aCtx, openai.ChatCompletionRequest{
		Model: ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "你是数据分析助手。根据用户问题、SQL 与查询结果，" +
					"给出简明的结论：直接回答问题本身，引用具体数字；" +
					"结果为空就明确说明没有匹配数据。" + langLine,
			},
			{Role: openai.ChatMessageRoleUser, Content: sb.String()},
		},
	})
	if err != nil {
		return "", 0, err
	}
	tokens := uint32(resp.Usage.TotalTokens)
	if len(resp.Choices) == 0 {
		return "", tokens, fmt.Errorf("empty answer")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), tokens, nil
}

// aiQueryExtractPattern 提取用：非锚定定位第一条 SELECT/WITH（模型输出可能带说明前缀）。
var aiQueryExtractPattern = regexp.MustCompile(`(?is)\b(SELECT|WITH)\b`)

// extractSQL 从模型输出中提取 SQL：模型偶发会在 SQL 前后带说明文字或围栏，
// 这里定位第一条 SELECT/WITH 起、到末尾分号（或文本尾）止的片段。
// 提取之后再过锚定的只读校验与危险模式正则，被篡改的语句仍会被拒。
func extractSQL(raw string) string {
	s := stripSQLFence(raw)
	loc := aiQueryExtractPattern.FindStringIndex(s)
	if loc == nil {
		return ""
	}
	s = s[loc[0]:]
	// 若以分号结尾（或其后还有说明文字），截到第一个分号
	if idx := strings.Index(s, ";"); idx >= 0 {
		s = s[:idx]
	}
	// 剥离行注释（LLM 爱加 -- 说明；注释里可能含被拒关键字）——
	// 逐行删除 "--" 起的内容，字符串字面量中的 "--" 场景对分析型 SQL 可忽略
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	return strings.TrimSpace(s)
}

// stripSQLFence 剥掉模型可能输出的 Markdown 代码块围栏。
func stripSQLFence(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```sql", "```SQL", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			if idx := strings.LastIndex(s, "```"); idx >= 0 {
				s = s[:idx]
			}
			break
		}
	}
	return strings.TrimSpace(s)
}

// stringifyCellValue 数据库原始值 → 展示字符串（time/[]byte/nil 特判）。
func stringifyCellValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// langName BCP-47 → 提示词里的语言名。
func langName(lang string) string {
	if strings.HasPrefix(lang, "en") {
		return "English"
	}
	return "中文"
}

// correctTableName 表名纠正：依候选序列匹配白名单（sys_ 前缀 / 复数 s / 组合）。
func correctTableName(table string, whitelist map[string]string) (string, bool) {
	if _, ok := whitelist[table]; ok {
		return table, true
	}
	for _, candidate := range []string{
		"sys_" + table,
		table + "s",
		"sys_" + table + "s",
		strings.TrimSuffix(table, "s"),
		"sys_" + strings.TrimSuffix(table, "s"),
	} {
		if _, ok := whitelist[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

// ── 定时问数（ai_query_run） ────────────────────────────────────────

// RunScheduledQuery 执行一次问数并返回格式化的文本结果（平台白名单）。
func (s *AiQueryService) RunScheduledQuery(ctx context.Context, question, lang string) (string, error) {
	provider, err := s.providerRepo.GetEnabledDefault(ctx)
	if err != nil {
		return "", err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return "", err
	}

	var history []aiV1.AiQueryHistoryItem
	sqlText, _, err := s.generateSQL(ctx, client, provider, question, lang, history, aiQueryTableWhitelist, 0)
	if err != nil {
		return "", err
	}
	cleaned, err := sanitizeSQLFor(sqlText, aiQueryTableWhitelist, nil)
	if err != nil {
		return "", err
	}
	_, rows, execErr := s.executeReadOnly(ctx, cleaned)
	if execErr != nil {
		return "", fmt.Errorf("execute: %w", execErr)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("问题：%s\n", question))
	sb.WriteString(fmt.Sprintf("SQL：%s\n", cleaned))
	if len(rows) == 0 {
		sb.WriteString("结果：无匹配数据")
	} else {
		sb.WriteString(fmt.Sprintf("结果（%d 行）：\n", len(rows)))
		for _, r := range rows {
			sb.WriteString("  " + strings.Join(r.Values, " | ") + "\n")
		}
	}
	return sb.String(), nil
}

// AsyncAiQueryRun 定时问数任务 handler：执行问题 → 站内信推送结果给任务创建人。
func (s *AiQueryService) AsyncAiQueryRun(taskType string, data *task.AiQueryRunTaskData) error {
	ctx := viewer.WithSystemContext(context.Background())
	question := data.Question
	if question == "" {
		return fmt.Errorf("question is required")
	}

	result, err := s.RunScheduledQuery(ctx, question, "zh-CN")
	if err != nil {
		return err
	}

	title := "AI 问数结果：" + question
	if err = s.pushInternalMessage(ctx, title, result); err != nil {
		s.log.Errorf(ctx, "ai query run: push result failed: %v", err)
	}
	return nil
}

// pushInternalMessage 创建站内信并投递给平台侧用户。
func (s *AiQueryService) pushInternalMessage(ctx context.Context, title, content string) error {
	now := time.Now()
	msg, err := s.internalMsgRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:     &title,
			Content:   &content,
			Status:    trans.Ptr(internalMessageV1.InternalMessage_PUBLISHED),
			Type:      trans.Ptr(internalMessageV1.InternalMessage_NOTIFICATION),
			CreatedBy: trans.Ptr(uint32(0)),
			CreatedAt: timestamppb.New(now),
		},
	})
	if err != nil {
		return err
	}

	// 投递给平台侧（tenant_id=0）用户
	users, err := s.entClient.Client().User.Query().
		Where(user.TenantIDEQ(0), user.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return err
	}
	for _, u := range users {
		if err = s.internalMessages.sendNotification(ctx, msg.GetId(), u.ID, 0, &now, title, content); err != nil {
			s.log.Errorf(ctx, "push to user %d failed: %v", u.ID, err)
		}
	}
	return nil
}
