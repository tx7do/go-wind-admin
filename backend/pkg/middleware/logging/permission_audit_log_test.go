package logging

import (
	"context"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLastNumericPathSegment(t *testing.T) {
	assert.Equal(t, "5", lastNumericPathSegment("/admin/v1/roles/5"))
	assert.Equal(t, "12", lastNumericPathSegment("/admin/v1/users/12/roles"))
	assert.Equal(t, "", lastNumericPathSegment("/admin/v1/roles"))
	assert.Equal(t, "", lastNumericPathSegment("/admin/v1/roles/batch-delete"))
	assert.Equal(t, "9", lastNumericPathSegment("/admin/v1/dicts/3/entries/9"))
	assert.Equal(t, "", lastNumericPathSegment(""))
	assert.Equal(t, "", lastNumericPathSegment("/"))
}

func TestTargetNameFromBody(t *testing.T) {
	// CRUD 包装
	assert.Equal(t, "运维", targetNameFromBody([]byte(`{"data":{"name":"运维","code":"ops"}}`)))
	// 平铺对象 + username 候选
	assert.Equal(t, "bob", targetNameFromBody([]byte(`{"username":"bob","nickname":"Bob"}`)))
	// 无 name 只有 code
	assert.Equal(t, "ops", targetNameFromBody([]byte(`{"data":{"code":"ops"}}`)))
	// title
	assert.Equal(t, "经理", targetNameFromBody([]byte(`{"data":{"title":"经理"}}`)))
	// 空串视为无值，取下一个候选
	assert.Equal(t, "x", targetNameFromBody([]byte(`{"name":"  ","code":"x"}`)))
	// 无任何名称字段
	assert.Equal(t, "", targetNameFromBody([]byte(`{"data":{"ids":[1,2]}}`)))
	// protojson 驼峰：typeName/typeCode（字典类型）
	assert.Equal(t, "审计e2e测试类型", targetNameFromBody([]byte(`{"data":{"typeCode":"audit_e2e_test","typeName":"审计e2e测试类型"}}`)))
	// 兜底：未列举的 roleName/menuName 类键
	assert.Equal(t, "运维", targetNameFromBody([]byte(`{"data":{"roleName":"运维"}}`)))
	// 非 JSON / 空体
	assert.Equal(t, "", targetNameFromBody([]byte(`not-json`)))
	assert.Equal(t, "", targetNameFromBody(nil))
}

// TestSnapshotWriteBodyReplay 快照后 body 必须可被 handler 完整重读（含超限剩余部分）。
func TestSnapshotWriteBodyReplay(t *testing.T) {
	ctx := context.Background()

	// 小于上限：整体快照
	small := `{"data":{"name":"a"}}`
	req := &nethttp.Request{
		Method: "PUT",
		Body:   io.NopCloser(strings.NewReader(small)),
		Header: nethttp.Header{"Content-Type": []string{"application/json"}},
	}
	snapCtx := snapshotWriteBody(ctx, req)
	assert.Equal(t, []byte(small), bodySnapshotFromContext(snapCtx))
	got, _ := io.ReadAll(req.Body)
	assert.Equal(t, small, string(got))

	// 超过上限：快照截断为 maxBodySnapshot，body 重读仍完整
	big := strings.Repeat("x", maxBodySnapshot+1024)
	req2 := &nethttp.Request{
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader(big)),
		Header: nethttp.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
	}
	snapCtx2 := snapshotWriteBody(ctx, req2)
	assert.Len(t, bodySnapshotFromContext(snapCtx2), maxBodySnapshot)
	got2, _ := io.ReadAll(req2.Body)
	assert.Len(t, got2, len(big))

	// GET 不快照、不动 body
	req3 := &nethttp.Request{
		Method: "GET",
		Body:   io.NopCloser(strings.NewReader(small)),
	}
	snapCtx3 := snapshotWriteBody(ctx, req3)
	assert.Nil(t, bodySnapshotFromContext(snapCtx3))
	got3, _ := io.ReadAll(req3.Body)
	assert.Equal(t, small, string(got3))

	// 非 JSON（如 multipart 上传）不快照、不动 body
	req4 := &nethttp.Request{
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader(small)),
		Header: nethttp.Header{"Content-Type": []string{"multipart/form-data"}},
	}
	assert.Nil(t, bodySnapshotFromContext(snapshotWriteBody(ctx, req4)))
	got4, _ := io.ReadAll(req4.Body)
	assert.Equal(t, small, string(got4))
}
