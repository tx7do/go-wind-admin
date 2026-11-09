package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/task"
)

// AiDigestService 审计日报 AI 摘要：定时聚合昨日操作审计 → 按收件用户偏好语言分组
// 生成摘要（默认模型；失败降级纯统计）→ 站内信投递平台侧用户。
type AiDigestService struct {
	log *bLogger.Helper

	auditRepo        *data.OperationAuditLogRepo
	internalMessages *InternalMessageService
	internalMsgRepo  *data.InternalMessageRepo
	scriptRuntime    *ScriptRuntime
	entClient        *entCrud.EntClient[*ent.Client]
}

func NewAiDigestService(
	ctx *bootstrap.Context,
	auditRepo *data.OperationAuditLogRepo,
	internalMessages *InternalMessageService,
	internalMsgRepo *data.InternalMessageRepo,
	scriptRuntime *ScriptRuntime,
	entClient *entCrud.EntClient[*ent.Client],
) *AiDigestService {
	return &AiDigestService{
		log:              ctx.NewLoggerHelper("ai_digest/service/admin-service"),
		auditRepo:        auditRepo,
		internalMessages: internalMessages,
		internalMsgRepo:  internalMsgRepo,
		scriptRuntime:    scriptRuntime,
		entClient:        entClient,
	}
}

// AsyncAiAuditDigest 日报任务 handler：昨日审计统计 → 按收件语言分组渲染
// （LLM 摘要或纯统计兜底）→ 站内信投递平台用户。
func (s *AiDigestService) AsyncAiAuditDigest(taskType string, data *task.AiAuditDigestTaskData) error {
	// asynq ctx 不带 viewer；统计是平台视角 + 站内信落库都需要 viewer，用系统查看器。
	ctx := viewer.WithSystemContext(context.Background())

	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -1)
	today := yesterday.AddDate(0, 0, 1)
	dayLabel := yesterday.Format("2006-01-02")

	stats, err := s.auditRepo.DigestStats(ctx, yesterday, today)
	if err != nil {
		return err
	}
	if stats.Total == 0 {
		s.log.Infof(ctx, "audit digest: no operations on %s, skip", dayLabel)
		return nil
	}

	return s.deliverToPlatformUsers(ctx, dayLabel, stats)
}

// ---- 日报文案（站内信，按收件用户偏好语言渲染） ----
//
// 站内信不是事务性邮件，文案不进 pkg/mailtext（那是邮件的收口）；
// 语言标签归一复用 mailtext.Locale/LocaleOfTag，日报文案表留在本服务内、
// 是其唯一出口。收件语言取自用户个人中心的 locale 设置（user.locale），
// 未设置或未识别回落默认语言——与事务性邮件同一条链。
type digestCopy struct {
	title        string // %s=日期
	dateLine     string // 以下为喂给模型的统计事实清单各行
	totalLine    string
	failedLine   string
	usersLine    string
	actionsHead  string
	actionLine   string // %s=动作名 %d=次数
	plainDigest  string // LLM 不可用时的纯统计兜底（占位：日期/总数/失败/用户）
	plainAction  string // %s=动作名 %d=次数
	plainJoiner  string
	systemPrompt string
}

var digestTables = map[mailtext.Locale]digestCopy{
	mailtext.LocaleZhCN: {
		title:       "AI 审计日报（%s）",
		dateLine:    "日期：%s\n",
		totalLine:   "操作总数：%d\n",
		failedLine:  "失败操作数：%d\n",
		usersLine:   "涉及用户数：%d\n",
		actionsHead: "动作分布：\n",
		actionLine:  "  - %s: %d 次\n",
		plainDigest: "昨日（%s）操作审计：总数 %d，失败 %d，涉及用户 %d。\n动作分布：",
		plainAction: "%s %d 次",
		plainJoiner: "，",
		systemPrompt: "你是企业后台的运维审计分析助手。根据给出的昨日操作审计统计数据，" +
			"用中文写一段 150 字以内的日报摘要：先一句话概括总体活跃度，再指出值得关注的点" +
			"（如失败率偏高、DELETE/EXPORT 类敏感操作占比、异常活跃用户等，没有就说明整体平稳）。" +
			"只输出摘要正文，不要寒暄和标题。",
	},
	mailtext.LocaleEnUS: {
		title:       "AI Audit Digest (%s)",
		dateLine:    "Date: %s\n",
		totalLine:   "Total operations: %d\n",
		failedLine:  "Failed operations: %d\n",
		usersLine:   "Users involved: %d\n",
		actionsHead: "Action distribution:\n",
		actionLine:  "  - %s: %d\n",
		plainDigest: "Yesterday (%s) operation audit: total %d, failed %d, users involved %d.\nAction distribution: ",
		plainAction: "%s %d times",
		plainJoiner: ", ",
		systemPrompt: "You are an operations audit analysis assistant for an enterprise admin console. " +
			"Based on the previous day's operation audit statistics provided, write a digest of at most " +
			"150 words in English: open with one sentence summarizing overall activity, then point out " +
			"anything worth attention (e.g. elevated failure rate, a high share of sensitive operations " +
			"such as DELETE or EXPORT, unusually active users; if there is nothing notable, state that " +
			"the day was unremarkable). Output only the digest body — no greetings, no title.",
	},
}

func digestCopyOf(locale mailtext.Locale) digestCopy {
	if cp, ok := digestTables[locale]; ok {
		return cp
	}
	return digestTables[mailtext.LocaleZhCN]
}

func digestTitle(locale mailtext.Locale, dayLabel string) string {
	return fmt.Sprintf(digestCopyOf(locale).title, dayLabel)
}

func digestFacts(locale mailtext.Locale, dayLabel string, stats *data.AuditDigestStats) string {
	cp := digestCopyOf(locale)
	var facts strings.Builder
	facts.WriteString(fmt.Sprintf(cp.dateLine, dayLabel))
	facts.WriteString(fmt.Sprintf(cp.totalLine, stats.Total))
	facts.WriteString(fmt.Sprintf(cp.failedLine, stats.Failed))
	facts.WriteString(fmt.Sprintf(cp.usersLine, stats.Users))
	facts.WriteString(cp.actionsHead)
	for _, a := range stats.TopActions {
		facts.WriteString(fmt.Sprintf(cp.actionLine, a.Action, a.Count))
	}
	return facts.String()
}

func buildPlainDigest(locale mailtext.Locale, dayLabel string, stats *data.AuditDigestStats) string {
	cp := digestCopyOf(locale)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(cp.plainDigest, dayLabel, stats.Total, stats.Failed, stats.Users))
	for i, a := range stats.TopActions {
		if i > 0 {
			sb.WriteString(cp.plainJoiner)
		}
		sb.WriteString(fmt.Sprintf(cp.plainAction, a.Action, a.Count))
	}
	return sb.String()
}

// summarize 拼装统计事实清单并调默认模型生成该语言的摘要。
func (s *AiDigestService) summarize(ctx context.Context, locale mailtext.Locale, dayLabel string, stats *data.AuditDigestStats) (string, error) {
	return s.scriptRuntime.ChatForScript(ctx, 0, digestCopyOf(locale).systemPrompt,
		digestFacts(locale, dayLabel, stats))
}

// deliverToPlatformUsers 按收件用户的偏好语言分组渲染并投递：每个语言组一条
// 消息行 + 组内用户收件行。单组失败记日志后继续其余组、不向上报错重试——
// asynq 重试会整任务重跑，重复投递已成功的组；全部组都失败才报错（此时无任何
// 收件行落库，重试无重复副作用）。
func (s *AiDigestService) deliverToPlatformUsers(ctx context.Context, dayLabel string, stats *data.AuditDigestStats) error {
	users, err := s.entClient.Client().User.Query().
		Where(user.TenantIDEQ(0), user.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		s.log.Errorf(ctx, "audit digest: query platform users failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("query platform users failed")
	}
	if len(users) == 0 {
		s.log.Infof(ctx, "audit digest: no platform users to deliver")
		return nil
	}

	groups := make(map[mailtext.Locale][]*ent.User)
	for _, u := range users {
		locale := mailtext.LocaleZhCN
		if u.Locale != nil {
			if l, ok := mailtext.LocaleOfTag(*u.Locale); ok {
				locale = l
			}
		}
		groups[locale] = append(groups[locale], u)
	}

	failedGroups := 0
	for locale, group := range groups {
		summary, err := s.summarize(ctx, locale, dayLabel, stats)
		if err != nil {
			// 摘要失败降级为纯统计文本，日报照发（AI 不可用不应吞掉日报）
			s.log.Errorf(ctx, "audit digest: llm summarize failed for locale %s, fallback to plain stats: %v", locale, err)
			summary = buildPlainDigest(locale, dayLabel, stats)
		}
		if err := s.deliverMessage(ctx, group, digestTitle(locale, dayLabel), summary); err != nil {
			s.log.Errorf(ctx, "audit digest: deliver locale %s group failed: %v", locale, err)
			failedGroups++
		}
	}
	if failedGroups > 0 && failedGroups == len(groups) {
		return fmt.Errorf("audit digest: all %d locale groups failed to deliver", failedGroups)
	}
	return nil
}

// deliverMessage 建一条站内信行并向给定用户投递收件行。
func (s *AiDigestService) deliverMessage(ctx context.Context, users []*ent.User, title, content string) error {
	now := time.Now()
	msg, err := s.internalMsgRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:     &title,
			Content:   trans.Ptr(content),
			Status:    trans.Ptr(internalMessageV1.InternalMessage_PUBLISHED),
			Type:      trans.Ptr(internalMessageV1.InternalMessage_NOTIFICATION),
			CreatedBy: trans.Ptr(uint64(0)),
			CreatedAt: timeutil.TimeToTimestamppb(&now),
		},
	})
	if err != nil {
		return err
	}

	for _, u := range users {
		if err = s.internalMessages.sendNotification(ctx, msg.GetId(), u.ID, 0, &now, title, content); err != nil {
			// 单人投递失败不阻断其余收件人
			s.log.Errorf(ctx, "audit digest: deliver to user %d failed: %v", u.ID, err)
		}
	}
	return nil
}
