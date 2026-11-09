package task

// AgentRunTaskType 是 Agent 定时触发任务的类型常量。
// 用户经「任务管理」创建 PERIODIC 行（type_name="agent_run"，cron + payload），
// 系统到点执行 Agent 应用图（start 入参来自 payload.input），结果可按
// payload.notifyUserId 回传站内信。适合定期巡检/报告类编排
// （如"每天早上跑一遍数据巡检图"）。
//
// 执行身份：UserId=0（机器身份，台账 created_by=0），租户取应用归属租户——
// LLM/RAG 用量记该租户流水、AI_TOKENS 配额按该租户闸。
const AgentRunTaskType = "agent_run"

// AgentRunTaskData 定时触发 Agent 任务的载荷（sys_tasks.task_payload 的 JSON 形态）。
type AgentRunTaskData struct {
	AppId        uint64            `json:"appId"`                  // 要执行的 Agent 应用 ID
	Input        map[string]string `json:"input,omitempty"`        // start 节点声明的输入变量名 → 值
	NotifyUserId uint64            `json:"notifyUserId,omitempty"` // 结果站内信回传的用户（缺省不通知，只看运行台账）
}
