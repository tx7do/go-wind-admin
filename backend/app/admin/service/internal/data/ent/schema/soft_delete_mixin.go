package schema

import (
	"context"
	"reflect"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/mixin"
)

// 本仓自带软删除 mixin：go-crud 的 mixin.SoftDelete 拦截器没有 context 旁路，
// 而回收站（租户停车）需要读已删行（trash 列表、purge 前置检查），故在此
// 提供一份带 SkipSoftDelete 旁路的等价实现。语义与 go-crud 版一致：
//   - deleted_at 列（NULL=在营）
//   - 查询拦截器自动注入 deleted_at IS NULL（fail-closed，注入失败报错而非放行）
//
// 注意：删除本身不是拦截器做的——业务侧 Delete 要显式 SetDeletedAt（停车）；
// 唯一键不会排除已删行，停车期间重名/重码会在 DB 层撞唯一索引（Create 侧
// 已把约束错误映射成友好提示）。

type softDeleteSkipKey struct{}

// SkipSoftDelete 给本次查询打上跳过软删除过滤的标记（回收站列表、purge
// 前置检查等需要读已删行的场景专用，勿在常规读路径使用）。
func SkipSoftDelete(ctx context.Context) context.Context {
	return context.WithValue(ctx, softDeleteSkipKey{}, true)
}

func softDeleteSkipped(ctx context.Context) bool {
	v, _ := ctx.Value(softDeleteSkipKey{}).(bool)
	return v
}

// SoftDelete 软删除拦截器 mixin：只挂查询过滤（可旁路）。
// deleted_at 列本身由 go-crud 的 mixin.TimeAt 提供（created/updated/deleted
// 三件套），无需在此重复声明字段。
type SoftDelete struct {
	mixin.Schema
}

func (SoftDelete) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		ent.InterceptFunc(func(next ent.Querier) ent.Querier {
			return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
				if !softDeleteSkipped(ctx) {
					if err := injectSoftDeleteWhere(query); err != nil {
						return nil, err
					}
				}
				return next.Query(ctx, query)
			})
		}),
	}
}

// injectSoftDeleteWhere 反射注入 deleted_at IS NULL 谓词。
// 与 go-crud 同构（fail-closed）：注入失败必须报错而非静默跳过，否则
// deleted_at 过滤悄悄消失、已删除数据被重新暴露。
func injectSoftDeleteWhere(query ent.Query) error {
	rv := reflect.ValueOf(query)
	mf := rv.MethodByName("Where")
	if !mf.IsValid() || mf.Kind() != reflect.Func {
		return &softDeleteInjectError{query: query}
	}

	mt := mf.Type()
	if !mt.IsVariadic() || mt.NumIn() != 1 {
		return &softDeleteInjectError{query: query}
	}

	elem := mt.In(0).Elem()
	selPtrType := reflect.TypeOf((*sql.Selector)(nil))
	if elem.Kind() != reflect.Func || elem.NumIn() < 1 || elem.In(0) != selPtrType {
		return &softDeleteInjectError{query: query}
	}

	fn := func(s *sql.Selector) {
		s.Where(sql.IsNull(s.C("deleted_at")))
	}
	valFn := reflect.ValueOf(fn)

	if valFn.Type() != elem {
		if valFn.Type().ConvertibleTo(elem) {
			valFn = valFn.Convert(elem)
		} else {
			valFn = reflect.MakeFunc(elem, func(in []reflect.Value) []reflect.Value {
				s := in[0].Interface().(*sql.Selector)
				fn(s)
				return nil
			})
		}
	}

	slice := reflect.MakeSlice(reflect.SliceOf(elem), 1, 1)
	slice.Index(0).Set(valFn)
	mf.CallSlice([]reflect.Value{slice})

	return nil
}

// softDeleteInjectError 软删过滤注入失败（fail-closed）。
type softDeleteInjectError struct {
	query ent.Query
}

func (e *softDeleteInjectError) Error() string {
	return "security: soft-delete where-injection failed (" + reflect.TypeOf(e.query).String() + ")"
}
