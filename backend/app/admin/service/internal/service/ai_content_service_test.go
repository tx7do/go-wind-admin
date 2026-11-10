package service

import (
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/trans"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// TestCollectMenuIndexDocs 菜单收集语义：跳过无 id / 无 path / 无标题菜单；
// 标题取 meta.title，为空回退 name；输出与输入顺序一致、item_id 落真实菜单 id。
func TestCollectMenuIndexDocs(t *testing.T) {
	mk := func(id int64, path, name, title string) *permissionV1.Menu {
		m := &permissionV1.Menu{}
		if id != 0 {
			m.Id = trans.Ptr(id)
		}
		if path != "" {
			m.Path = trans.Ptr(path)
		}
		if name != "" {
			m.Name = trans.Ptr(name)
		}
		if title != "" {
			m.Meta = &permissionV1.MenuMeta{Title: trans.Ptr(title)}
		}
		return m
	}
	items := []*permissionV1.Menu{
		mk(1, "/a", "", "标题A"), // 正常：meta.title
		mk(2, "/b", "名字B", ""), // 标题回退：name
		mk(0, "/c", "x", "t"),  // 无 id → 跳过
		mk(3, "", "x", "t"),    // 无 path → 跳过
		mk(4, "/d", "", ""),    // 无标题 → 跳过
		nil,                    // nil（getter 零值 → 按 id=0 跳过）
	}
	got := collectMenuIndexDocs(items)
	if len(got) != 2 {
		t.Fatalf("collectMenuIndexDocs collected %d docs, want 2", len(got))
	}
	if got[0].menuId != 1 || got[0].title != "标题A" || got[0].route != "/a" {
		t.Fatalf("doc[0] mismatch: %+v", got[0])
	}
	if got[1].menuId != 2 || got[1].title != "名字B" || got[1].route != "/b" {
		t.Fatalf("doc[1] mismatch: %+v", got[1])
	}
}

// TestPlaceEmbeddingBatch 响应对位校验：合法响应按 index 对位写入；
// 越界 index、负 index、零维向量一律整体拒绝。
func TestPlaceEmbeddingBatch(t *testing.T) {
	vec := func(n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = 0.5
		}
		return v
	}

	// 合法：乱序响应按 index 对位（不按返回顺序）
	vectors := []string{"", "", ""}
	err := placeEmbeddingBatch(0, []openai.Embedding{
		{Index: 1, Embedding: vec(2)},
		{Index: 0, Embedding: vec(2)},
	}, vectors)
	if err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	if vectors[0] != "[0.5,0.5]" || vectors[1] != "[0.5,0.5]" {
		t.Fatalf("placement mismatch: %v", vectors)
	}
	if vectors[2] != "" {
		t.Fatalf("out-of-batch slot must stay empty: %q", vectors[2])
	}

	// 越界 index（batchStart + index == len）
	if err = placeEmbeddingBatch(0, []openai.Embedding{{Index: 3, Embedding: vec(2)}}, vectors); err == nil {
		t.Fatal("out-of-range index must be rejected")
	}
	// 负 index
	if err = placeEmbeddingBatch(1, []openai.Embedding{{Index: -1, Embedding: vec(2)}}, vectors); err == nil {
		t.Fatal("negative index must be rejected")
	}
	// 零维向量
	if err = placeEmbeddingBatch(0, []openai.Embedding{{Index: 0, Embedding: nil}}, vectors); err == nil {
		t.Fatal("zero-dim embedding must be rejected")
	}
}
