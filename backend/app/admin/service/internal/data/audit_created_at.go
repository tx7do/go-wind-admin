package data

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// auditCreatedAt 审计行落库时间：DTO 携带非空 CreatedAt 时以其为准（异步队列
// 消费侧回放事件时间，消除队列延迟造成的落库时间漂移）；否则取当前时间
// （同步路径行为不变——中间件不设置 DTO.CreatedAt）。
func auditCreatedAt(dtoValue *timestamppb.Timestamp) time.Time {
	if dtoValue != nil {
		return dtoValue.AsTime()
	}
	return time.Now()
}
