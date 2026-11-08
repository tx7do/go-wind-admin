package service

import (
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestShouldDeleteArchivedNotification(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.Local)
	retention := 24 * time.Hour
	cutoff := now.Add(-retention)

	oldDispatch := &asynq.TaskInfo{
		ID:           "task-old",
		Type:         "notification_dispatch",
		Queue:        "default",
		LastFailedAt: now.Add(-48 * time.Hour),
	}
	freshDispatch := &asynq.TaskInfo{
		ID:           "task-fresh",
		Type:         "notification_dispatch",
		Queue:        "default",
		LastFailedAt: now.Add(-2 * time.Hour),
	}
	otherType := &asynq.TaskInfo{
		ID:           "task-other",
		Type:         "audit_log_archive",
		Queue:        "default",
		LastFailedAt: now.Add(-72 * time.Hour),
	}
	noFailTime := &asynq.TaskInfo{
		ID:    "task-nofail",
		Type:  "notification_dispatch",
		Queue: "default",
	}

	require.True(t, shouldDeleteArchivedNotification(oldDispatch, "notification_dispatch", cutoff), "超保留期的派发任务应删")
	require.False(t, shouldDeleteArchivedNotification(freshDispatch, "notification_dispatch", cutoff), "保留期内的不删")
	require.False(t, shouldDeleteArchivedNotification(otherType, "notification_dispatch", cutoff), "非通知派发类型不越界代删")
	require.False(t, shouldDeleteArchivedNotification(noFailTime, "notification_dispatch", cutoff), "无失败时间的异常态不删（宁可漏删不误删）")
	require.False(t, shouldDeleteArchivedNotification(nil, "notification_dispatch", cutoff), "nil 条目安全跳过")
}

func TestAsynqQueueName(t *testing.T) {
	require.Equal(t, "default", asynqQueueName(nil), "无配置兜底 default")
	require.Equal(t, "default", asynqQueueName(map[string]int32{}), "空配置兜底 default")
	require.Equal(t, "default", asynqQueueName(map[string]int32{"critical": 10, "default": 5, "low": 1}), "含 default 时直接取")
	require.Equal(t, "critical", asynqQueueName(map[string]int32{"critical": 10, "low": 1}), "队列改名场景取权重最高的一个")
}
