package service

import (
	"context"
	"fmt"
	"strings"

	"go-wind-admin/pkg/mailtext"

	"go-wind-admin/app/admin/service/internal/data"
)

// 事务性邮件的模板化覆写（可覆写，非强依赖）：
//
// 三类事务性出站（找回密码验证码 / 邮箱换绑验证码 / 渠道测试邮件）此前文案
// 硬编码在 pkg/mailtext。P3 第二片的模板管理（sys_notification_templates）
// 已具备渲染能力，本文件把两者接上——按**约定 code** 查模板：
//
//	密码重置 → pwd_reset_code
//	邮箱换绑 → contact_bind_code
//	渠道测试 → channel_test_email
//
// 存在且启用 → 用模板渲染（变量 code/user 等）；否则回落 mailtext 内置文案。
// 语义为"运营可改文案，不改变流程"：模板缺失/停用/渲染失败全部静默回落——
// 事务性邮件是认证链路的关键路径，绝不能因模板配置问题而发不出验证码。

// 事务性邮件的模板 code 约定。管理端按这些 code 建模板即可覆写对应邮件文案。
const (
	TplCodePwdReset     = "pwd_reset_code"
	TplCodeContactBind  = "contact_bind_code"
	TplCodeChannelTest  = "channel_test_email"
	TplCodeRuleTestNote = "rule_test_notification"
)

// TransactionMailer 事务性邮件渲染器（wiring 装配；模板渲染优先，mailtext 回落）。
// TemplateRepo 为 nil（旧测试装配）时直接走 mailtext，行为与接线前一致。
type TransactionMailer struct {
	TemplateRepo *data.NotificationTemplateRepo
}

// transactionalMailer 为内部别名：三个便捷入口的 receiver 走它，测试桩可直构。
type transactionalMailer = TransactionMailer

// render 事务性邮件文案：模板（按 code 命中且启用）渲染优先，回落 mailtext。
//
// vars 为模板变量集（code/user 等）；mailtextFallback 为内置文案生成函数。
// 渲染失败静默回落并返回 nil（由调用方记日志）——渲染错误不能阻断验证码发送。
func renderTransactional(
	ctx context.Context,
	mailer *transactionalMailer,
	tplCode string,
	vars map[string]string,
	mailtextFallback func() (string, string),
) (string, string) {
	if mailer == nil || mailer.TemplateRepo == nil {
		return mailtextFallback()
	}

	tpl, err := mailer.TemplateRepo.GetByCode(ctx, tplCode)
	if err != nil || tpl == nil || !tpl.GetIsEnabled() {
		return mailtextFallback()
	}

	title, err := renderTemplate(tpl.GetTitleTemplate(), vars)
	if err != nil {
		return mailtextFallback()
	}
	content, err := renderTemplate(tpl.GetContentTemplate(), vars)
	if err != nil {
		return mailtextFallback()
	}
	return title, content
}

// ---- 三个事务性场景的便捷入口（参数即各场景的模板变量） ----

// renderPwdResetCode 找回密码验证码邮件。模板 code：pwd_reset_code；
// 变量：{{code}}、{{codeSpaced}}（每字符空格分隔，便于人工抄写）。
func renderPwdResetCode(ctx context.Context, mailer *transactionalMailer, code string) (string, string) {
	return renderTransactional(ctx, mailer, TplCodePwdReset,
		map[string]string{"code": code, "codeSpaced": strings.Join(strings.Split(code, ""), " ")},
		func() (string, string) { return mailtext.PasswordResetCode(ctx, code) })
}

// renderContactBindCode 邮箱换绑验证码邮件。模板 code：contact_bind_code。
func renderContactBindCode(ctx context.Context, mailer *transactionalMailer, code string) (string, string) {
	return renderTransactional(ctx, mailer, TplCodeContactBind,
		map[string]string{"code": code, "codeSpaced": strings.Join(strings.Split(code, ""), " ")},
		func() (string, string) { return mailtext.ContactBindCode(ctx, code) })
}

// renderChannelTestEmail 渠道测试邮件。模板 code：channel_test_email；
// 变量：{{channelId}}、{{operatorId}}。
func renderChannelTestEmail(ctx context.Context, mailer *transactionalMailer, channelID, operatorID uint64) (string, string) {
	return renderTransactional(ctx, mailer, TplCodeChannelTest,
		map[string]string{
			"channelId":  fmtUint(channelID),
			"operatorId": fmtUint(operatorID),
		},
		func() (string, string) { return mailtext.ChannelTestEmail(ctx, channelID, operatorID) })
}

// renderRuleTestNotification 路由规则测试通知。模板 code：rule_test_notification。
func renderRuleTestNotification(ctx context.Context, mailer *transactionalMailer, ruleID uint64, eventType string) (string, string) {
	return renderTransactional(ctx, mailer, TplCodeRuleTestNote,
		map[string]string{"ruleId": fmtUint(ruleID), "eventType": eventType},
		func() (string, string) { return mailtext.RuleTestNotification(ctx, ruleID, eventType) })
}

func fmtUint(v uint64) string {
	return strings.TrimSpace(strings.Join(strings.Fields(fmt.Sprint(v)), ""))
}
