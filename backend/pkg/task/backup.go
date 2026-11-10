package task

import "fmt"

const (
	BackupTaskType = "backup"
)

type BackupTaskData struct {
	Name string `json:"name"`
}

// CreateBackupTaskID creates a unique task ID for a backup task based on the lottery details.
func CreateBackupTaskID(taskId int64) string {
	return fmt.Sprintf("%s:%d",
		BackupTaskType, taskId,
	)
}
