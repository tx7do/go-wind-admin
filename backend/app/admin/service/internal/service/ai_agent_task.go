package service

import (
	"context"
	"fmt"
	"time"

	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/go-crud/viewer"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"
)

// Agent 定时触发任务桥（M3「定时触发」，docs/ai_agent_orchestration.md §M3）：
// 「任务管理」建 PERIODIC 行（type_name="agent_run"，payload 带 appId/input/notifyUserId），
// 调度器到点直投本类型，handler 经内部 Run 执行图并把结果回传站内信。
// 形态对齐 ai_query_run 的"任务桥 + 站内信回传"先例（其 asynq 注册已补齐，见
// asynq_server.go——此前注册缺失使其成为死代码，类型经 H8 守卫必 400）。

// AsyncAgentRun 定时触发任务 handler：执行 Agent 应用图 → 结果回传站内信。
//
// 执行身份：UserId=0（机器身份，运行台账 created_by=0、SSE 流 ID="0"），租户取
// 应用归属租户——LLM/RAG 用量记该租户流水、AI_TOKENS 配额按该租户闸。
// 错误语义：图执行失败已落 FAILED 台账（确定性结论，不重试）；仅基础设施错误
// （DB/校验外的意外失败）返回 error 交 asynq 重试。
func (s *AiAgentService) AsyncAgentRun(taskType string, data *task.AgentRunTaskData) error {
	ctx := viewer.WithSystemContext(context.Background())
	if data == nil || data.AppId == 0 {
		return fmt.Errorf("appId is required")
	}

	app, err := s.appRepo.GetEntityByID(ctx, data.AppId)
	if err != nil {
		return err
	}
	tenantID := int64(0)
	if app.TenantID != nil {
		tenantID = *app.TenantID
	}
	if !app.Enabled {
		// 停用的应用本轮跳过：管理员停用不应引发 asynq 重试风暴，也不算一次执行（不落台账）。
		s.log.Warnf(ctx, "AsyncAgentRun: app [%d] disabled, skip scheduled run", data.AppId)
		return nil
	}

	// 机器身份执行：Run 内部照常做静态校验/配额/台账/SSE（StreamID="0" 无订阅者，帧自然丢弃）。
	operator := &authenticationV1.UserTokenPayload{
		UserId:   0,
		TenantId: trans.Ptr(tenantID),
		Username: trans.Ptr(fmt.Sprintf("agent:scheduled:%d", data.AppId)),
		Roles:    []string{"machine"},
	}
	runCtx := auth.NewContext(ctx, operator)
	run, err := s.Run(runCtx, &aiV1.RunAiAgentAppRequest{Id: data.AppId, Input: data.Input})
	if err != nil {
		return err
	}

	if data.NotifyUserId > 0 {
		s.notifyAgentRun(ctx, run, data.NotifyUserId, tenantID)
	}
	return nil
}

// notifyAgentRun 把定时触发的执行结论以站内信回传给 notifyUserId
// （发送失败仅记日志——通知是辅功能，不该让 asynq 重试一次已完成的执行）。
func (s *AiAgentService) notifyAgentRun(ctx context.Context, run *aiV1.AiAgentRun, notifyUserID, tenantID int64) {
	if s.internalMessages == nil || s.internalMsgRepo == nil {
		return
	}

	statusText := "成功"
	detail := run.GetOutput()
	if run.GetStatus().String() != "SUCCEEDED" {
		statusText = "失败"
		detail = run.GetError()
	}
	title := fmt.Sprintf("Agent 定时执行%s（应用 #%d）", statusText, run.GetAppId())
	content := fmt.Sprintf("运行 #%d 状态：%s\n运行ID：%d\n", run.GetId(), statusText, run.GetId())
	if len(detail) > 600 {
		content += "产出/错误（截断）：\n" + detail[:600]
	} else if detail != "" {
		content += "产出/错误：\n" + detail
	} else {
		content += "（无产出）"
	}

	now := time.Now()
	msg, err := s.internalMsgRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:     &title,
			Content:   trans.Ptr(content),
			Status:    trans.Ptr(internalMessageV1.InternalMessage_PUBLISHED),
			Type:      trans.Ptr(internalMessageV1.InternalMessage_NOTIFICATION),
			CreatedBy: trans.Ptr(int64(0)),
			CreatedAt: timeutil.TimeToTimestamppb(&now),
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "notify agent run: create message failed: %s", err.Error())
		return
	}
	_ = tenantID
	if err = s.internalMessages.sendNotification(ctx, msg.GetId(), notifyUserID, 0, &now, title, content); err != nil {
		s.log.Errorf(ctx, "notify agent run: send to user [%d] failed: %s", notifyUserID, err.Error())
	}
}
