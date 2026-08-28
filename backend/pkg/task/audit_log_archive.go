package task

// AuditLogArchiveTaskType 是审计日志归档任务的类型常量。
// 系统级常驻定时任务（不写入 sys_tasks 表），由 asynq 调度器按固定 cron 周期触发，
// handler 为 TaskService.AsyncAuditLogArchive。
// 等保要求：审计日志留存不少于 6 个月。本任务把超过保留期（默认 180 天）的
// 审计行导出为本地 JSONL 归档文件后从库中删除，库瘦身、日志留痕两不误。
const AuditLogArchiveTaskType = "audit_log_archive"

// AuditLogArchiveCronSpec 审计日志归档的 cron（每天 03:30 低峰执行）。
const AuditLogArchiveCronSpec = "30 3 * * *"

// AuditLogArchiveTaskData 审计日志归档任务的载荷（参数走环境变量：
// AUDIT_ARCHIVE_DIR 归档目录，默认 ./data/audit-archive；
// AUDIT_RETENTION_DAYS 库内保留天数，默认 180）。
type AuditLogArchiveTaskData struct{}
