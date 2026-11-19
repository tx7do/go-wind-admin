// 本文件补 pkg/task 其余载荷结构体的线上契约测试：
// 这些结构体就是 sys_tasks.task_payload / asynq 消息的 wire format，
// 生产者与消费者分属两处代码，json tag 被误改会静默破坏消费侧（字段全为空值）。
// 对每个非空载荷做 Marshal 键名断言 + 反序列化往返，把 tag 钉死；
// 任务类型与 cron 常量一并钉死（asynq mux 注册、sys_tasks typeName 去重都依赖字面值，
// AiQueryRunTaskType 就曾因漏注册出过事故）。
package task

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- 载荷 JSON 契约 ---

func TestAgentRunTaskDataContract(t *testing.T) {
	in := AgentRunTaskData{
		AppId:        42,
		Input:        map[string]string{"topic": "巡检"},
		NotifyUserId: 7,
	}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"appId":42,"input":{"topic":"巡检"},"notifyUserId":7}`, string(raw))

	var out AgentRunTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, in, out)

	t.Run("Input与NotifyUserId的omitempty", func(t *testing.T) {
		raw, err := json.Marshal(AgentRunTaskData{AppId: 1})
		require.NoError(t, err)
		require.JSONEq(t, `{"appId":1}`, string(raw))
	})
}

func TestAiDocReindexTaskDataContract(t *testing.T) {
	raw, err := json.Marshal(AiDocReindexTaskData{BaseID: 9})
	require.NoError(t, err)
	require.JSONEq(t, `{"baseId":9}`, string(raw))

	var out AiDocReindexTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, int64(9), out.BaseID)

	t.Run("BaseID为0表示全量重建", func(t *testing.T) {
		var out AiDocReindexTaskData
		require.NoError(t, json.Unmarshal([]byte(`{"baseId":0}`), &out))
		require.Equal(t, int64(0), out.BaseID)
	})
}

func TestAiQueryRunTaskDataContract(t *testing.T) {
	in := AiQueryRunTaskData{Question: "昨日失败登录有哪些", Lang: "zh-CN"}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"question":"昨日失败登录有哪些","lang":"zh-CN"}`, string(raw))

	var out AiQueryRunTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, in, out)
}

func TestAuditLogWriteTaskDataContract(t *testing.T) {
	in := AuditLogWriteTaskData{
		Kind:       AuditLogKindApi,
		OccurredAt: "2026-11-19T10:00:00.123456789+08:00",
		Data:       map[string]any{"success": true, "traceId": "abc"},
	}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"kind":"api","occurredAt":"2026-11-19T10:00:00.123456789+08:00","data":{"success":true,"traceId":"abc"}}`, string(raw))

	var out AuditLogWriteTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, in, out)
}

func TestBackupTaskDataContract(t *testing.T) {
	raw, err := json.Marshal(BackupTaskData{Name: "nightly"})
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"nightly"}`, string(raw))

	var out BackupTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, "nightly", out.Name)
}

func TestBroadcastMessageTaskDataContract(t *testing.T) {
	// snake_case 命名（与 camelCase 的 AgentRunTaskData 风格不同），tag 误改会让
	// handler 读不到受众租户——TenantId 是受众范围的唯一依据，必须钉死。
	raw, err := json.Marshal(BroadcastMessageTaskData{MessageId: 11, TenantId: 3})
	require.NoError(t, err)
	require.JSONEq(t, `{"message_id":11,"tenant_id":3}`, string(raw))

	var out BroadcastMessageTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, int64(11), out.MessageId)
	require.Equal(t, int64(3), out.TenantId)
}

func TestNotificationDispatchTaskDataContract(t *testing.T) {
	in := NotificationDispatchTaskData{
		DeliveryId: 5,
		Target:     "user@example.com",
		Title:      "找回密码",
		Content:    "code",
		Params:     map[string]string{"code": "123456"},
	}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"delivery_id":5,"target":"user@example.com","title":"找回密码","content":"code","params":{"code":"123456"}}`, string(raw))

	var out NotificationDispatchTaskData
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Equal(t, in, out)

	t.Run("Params的omitempty", func(t *testing.T) {
		raw, err := json.Marshal(NotificationDispatchTaskData{DeliveryId: 1})
		require.NoError(t, err)
		require.JSONEq(t, `{"delivery_id":1,"target":"","title":"","content":""}`, string(raw))
	})
}

// --- 常量钉死（asynq mux 注册与 sys_tasks typeName 去重依赖字面值） ---

func TestTaskTypeConstantsPinned(t *testing.T) {
	require.Equal(t, "agent_run", AgentRunTaskType)
	require.Equal(t, "ai_audit_digest", AiAuditDigestTaskType)
	require.Equal(t, "ai_doc_reindex", AiDocReindexTaskType)
	require.Equal(t, "ai_query_run", AiQueryRunTaskType)
	require.Equal(t, "audit_log_archive", AuditLogArchiveTaskType)
	require.Equal(t, "audit_log_write", AuditLogWriteTaskType)
	require.Equal(t, "backup", BackupTaskType)
	require.Equal(t, "broadcast_message", BroadcastMessageTaskType)
	require.Equal(t, "monitor_alert_scan", MonitorAlertScanTaskType)
	require.Equal(t, "notification_archived_cleanup", NotificationArchivedCleanupTaskType)
	require.Equal(t, "notification_delivery_sweep", NotificationDeliverySweepTaskType)
	require.Equal(t, "notification_dispatch", NotificationDispatchTaskType)
	require.Equal(t, "plan_quota_watermark_scan", PlanQuotaWatermarkTaskType)
	require.Equal(t, "tenant_expiry_scan", TenantExpiryScanTaskType)
}

func TestCronSpecConstantsPinned(t *testing.T) {
	// 系统级常驻任务的 cron；改动即改变巡检节奏，须有意为之。
	require.Equal(t, "0 8 * * *", AiAuditDigestCronSpec)
	require.Equal(t, "30 3 * * *", AuditLogArchiveCronSpec)
	require.Equal(t, "*/5 * * * *", MonitorAlertScanCronSpec)
	require.Equal(t, "0 * * * *", NotificationArchivedCleanupCronSpec)
	require.Equal(t, "*/5 * * * *", NotificationDeliverySweepCronSpec)
	require.Equal(t, "0 9 * * *", PlanQuotaWatermarkCronSpec)
	require.Equal(t, "0 * * * *", TenantExpiryScanCronSpec)
}

func TestAuditLogKindConstantsPinned(t *testing.T) {
	// payload.kind 的取值域，消费侧据此路由到对应审计表。
	require.Equal(t, "api", AuditLogKindApi)
	require.Equal(t, "login", AuditLogKindLogin)
	require.Equal(t, "operation", AuditLogKindOperation)
	require.Equal(t, "permission", AuditLogKindPermission)
	require.Equal(t, "data_access", AuditLogKindDataAccess)
	require.Equal(t, "policy_evaluation", AuditLogKindPolicyEvaluation)
}

// --- 空载荷结构体的存在性（编排处会引用这些类型构造 payload） ---

func TestEmptyPayloadStructs(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  any
	}{
		{"AiAuditDigest", AiAuditDigestTaskData{}},
		{"AuditLogArchive", AuditLogArchiveTaskData{}},
		{"MonitorAlertScan", MonitorAlertScanTaskData{}},
		{"NotificationArchivedCleanup", NotificationArchivedCleanupTaskData{}},
		{"NotificationDeliverySweep", NotificationDeliverySweepTaskData{}},
		{"PlanQuotaWatermark", PlanQuotaWatermarkTaskData{}},
		{"TenantExpiryScan", TenantExpiryScanTaskData{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.raw)
			require.NoError(t, err)
			require.JSONEq(t, `{}`, string(raw))
		})
	}
}
