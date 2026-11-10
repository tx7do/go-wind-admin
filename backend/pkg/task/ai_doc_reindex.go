package task

// AiDocReindexTaskType 是知识库向量重索引任务的类型常量。
// 非系统常驻 cron：经「任务管理」按需创建（cron 任意，通常一次性或低频），
// 用于更换知识库的 embedding 模型后对既有切片全量重算向量——
// 切片文本不变（切片在入库时已定），只换 embedding 列。
const AiDocReindexTaskType = "ai_doc_reindex"

// AiDocReindexTaskData 重索引任务载荷。
// BaseID=0 表示对全部知识库重建；>0 只重建指定库。
type AiDocReindexTaskData struct {
	BaseID int64 `json:"baseId"`
}
