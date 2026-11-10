package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateBackupTaskID 钉死备份任务 ID 的拼接契约：BackupTaskType + ":" + id。
func TestCreateBackupTaskID(t *testing.T) {
	cases := []struct {
		name   string
		taskId int64
		want   string
	}{
		{"zero", 0, "backup:0"},
		{"small", 1, "backup:1"},
		{"large", 4294967295, "backup:4294967295"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, CreateBackupTaskID(tc.taskId))
		})
	}
}

// TestBackupTaskTypeConstant 钉死 BackupTaskType 常量值，防止误改影响任务调度识别。
func TestBackupTaskTypeConstant(t *testing.T) {
	assert.Equal(t, "backup", BackupTaskType)
}
