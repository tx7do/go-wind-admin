package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-transport/transport/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"

	"go-wind-admin/pkg/sseevent"
)

// recordingPublisher 抓下 TryPublish 的入参，用来断言 SSE 帧真正的事件名与 data 形状。
type recordingPublisher struct {
	streamID sse.StreamID
	event    *sse.Event
}

func (r *recordingPublisher) Publish(_ context.Context, _ sse.StreamID, _ *sse.Event) {}

func (r *recordingPublisher) TryPublish(_ context.Context, streamID sse.StreamID, event *sse.Event) bool {
	r.streamID, r.event = streamID, event
	return true
}

// TestInternalMessageSsePayloadContract 钉住 notification 事件的线上格式。
//
// 这条测试存在的理由是一个静默故障：收件记录若用 encoding/json 序列化，键是结构体 tag 上的
// snake_case（message_id / created_at），而三端通知面板按 REST 的 camelCase 读字段。
// vue-element 的 handleSseNotification 开头就是 `if (!data.id || !data.messageId) return`，
// 于是实时提醒、未读自增、下拉插入全部不触发，且没有任何报错——只有落库的收件箱能看出消息来过。
// 因此这里断言的是"前端读得到"，而不是"序列化没出错"。
func TestInternalMessageSsePayloadContract(t *testing.T) {
	svc := &InternalMessageService{log: bLogger.NewHelper(bLogger.NopLogger())}
	rec := &recordingPublisher{}
	svc.RegisterInternalMessagePublisher(rec)

	now := time.Now()
	recipient := newMessageRecipient(11, 22, 33, 44, &now, "标题", "正文")

	svc.publishNotification(context.Background(), recipient)

	require.NotNil(t, rec.event, "TryPublish 未被调用")
	assert.Equal(t, sseevent.Notification, string(rec.event.Event), "事件名必须来自 pkg/sseevent 注册表")
	assert.Equal(t, sse.StreamID("22"), rec.streamID, "streamID 用收件用户 ID")
	require.NotEmpty(t, rec.event.ID, "事件 id 用于客户端去重")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.event.Data, &payload))

	assert.Equal(t, float64(11), payload["messageId"])
	assert.Equal(t, float64(22), payload["recipientUserId"])
	assert.Equal(t, float64(44), payload["tenantId"])
	assert.Equal(t, "标题", payload["title"])
	assert.Equal(t, "正文", payload["content"])
	// status 必须是枚举名：面板按 item.status === 'READ' 判断已读，拿到 varint 会永远为 false
	assert.Equal(t, internalMessageV1.InternalMessageRecipient_RECEIVED.String(), payload["status"])
	assert.NotEmpty(t, payload["createdAt"], "dateUtil(createdAt).fromNow() 依赖该字段")

	for _, snake := range []string{"message_id", "recipient_user_id", "created_at", "tenant_id"} {
		assert.NotContains(t, payload, snake, "SSE data 必须是 protojson 的 camelCase 形状")
	}
}
