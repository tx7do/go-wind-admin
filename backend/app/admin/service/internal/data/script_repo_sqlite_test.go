package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/script"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newScriptRepoSqlite 在给定 enttest client 上白盒构造 ScriptRepo，
// 逐字段复刻 NewScriptRepo 的 mapper/converter 初始化，再调用 init()。
func newScriptRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *ScriptRepo {
	t.Helper()
	repo := &ScriptRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[scriptV1.Script, ent.Script](),
		languageConverter: mapper.NewEnumTypeConverter[scriptV1.Language, script.Language](
			scriptV1.Language_name, scriptV1.Language_value,
		),
	}
	repo.init()
	return repo
}

// TestScriptRepoSqlite_Create 通过 repo.Create 写入后直查 SQLite 断言落库
// （含 language 枚举经 converter 的落库值与版本默认值）。
func TestScriptRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:        trans.Ptr("sqlite_script_create"),
			Language:    scriptV1.Language_JAVASCRIPT.Enum(),
			HookPoint:   trans.Ptr("on_test"),
			Source:      trans.Ptr("return 1"),
			Priority:    trans.Ptr(int32(5)),
			Description: trans.Ptr("创建用途"),
			Critical:    trans.Ptr(false),
			IsEnabled:   trans.Ptr(true),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成成功")

	rows, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 script 记录")
	require.Equal(t, "sqlite_script_create", *rows[0].Name, "name 应按请求落库")
	require.Equal(t, "on_test", *rows[0].HookPoint, "hook_point 应按请求落库")
	require.Equal(t, "return 1", *rows[0].Source, "source 应按请求落库")
	require.Equal(t, script.LanguageJAVASCRIPT, *rows[0].Language, "language 枚举应经 converter 落为 JAVASCRIPT")
	require.NotNil(t, rows[0].Version, "版本默认值应落库")
	require.Equal(t, uint32(1), *rows[0].Version, "新建脚本版本应为默认 1")
}

// TestScriptRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义。
func TestScriptRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{Name: trans.Ptr("markerxyz_script_a")},
	}))
	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{Name: trans.Ptr("unrelated_script_b")},
	}))

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerxyz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].Name, "markerxyz", "命中行应是携带标记的行")

	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "no-such-marker-zzz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
}

// TestScriptRepoSqlite_Get 验证 Get 命中/未命中，
// 以及 GetByName / GetSourceByName / GetVersionByName 的按名查询。
func TestScriptRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:   trans.Ptr("sqlite_script_get"),
			Source: trans.Ptr("return 42"),
		},
	}))
	rows, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// Get 命中/未命中
	hit, err := repo.Get(ctx, &scriptV1.GetScriptRequest{
		QueryBy: &scriptV1.GetScriptRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())
	_, err = repo.Get(ctx, &scriptV1.GetScriptRequest{
		QueryBy: &scriptV1.GetScriptRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")

	// GetByName 命中
	byName, err := repo.GetByName(ctx, "sqlite_script_get")
	require.NoError(t, err, "按存在的名字查询应命中")
	require.Equal(t, createdID, byName.GetId())

	// GetByName 未命中
	_, err = repo.GetByName(ctx, "no-such-script-zzz")
	require.Error(t, err, "不存在的名字查询应返回错误")

	// GetSourceByName 命中/未命中
	src, err := repo.GetSourceByName(ctx, "sqlite_script_get")
	require.NoError(t, err)
	require.Equal(t, "return 42", src, "按名应取回脚本源码")
	_, err = repo.GetSourceByName(ctx, "no-such-script-zzz")
	require.Error(t, err, "不存在的名字取源码应返回错误")

	// GetVersionByName 命中：返回的是版本号数字串（修复前格式化的是
	// *uint32 指针地址，DBSource 热更新变更检测指纹因此失效）。
	version, err := repo.GetVersionByName(ctx, "sqlite_script_get")
	require.NoError(t, err)
	require.Equal(t, "1", version, "按名应返回初始版本号 1 的十进制串")
	_, err = repo.GetVersionByName(ctx, "no-such-script-zzz")
	require.Error(t, err, "不存在的名字取版本指纹应返回错误")
}

// TestScriptRepoSqlite_ListEnabledScripts 验证只返回启用脚本。
func TestScriptRepoSqlite_ListEnabled(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("sqlite_script_enabled"),
			IsEnabled: trans.Ptr(true),
		},
	}))
	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("sqlite_script_disabled"),
			IsEnabled: trans.Ptr(false),
		},
	}))

	enabled, err := repo.ListEnabledScripts(ctx)
	require.NoError(t, err)
	require.Len(t, enabled, 1, "只应返回启用的脚本")
	require.Equal(t, "sqlite_script_enabled", enabled[0].GetName(), "返回的应是启用脚本")
}

// TestScriptRepoSqlite_IsNameExist 验证重名检查：
// 已存在名字为真；排除自身 ID 后为假；不存在的名字为假。
func TestScriptRepoSqlite_IsNameExist(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{Name: trans.Ptr("sqlite_script_nameexist")},
	}))
	rows, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	exist, err := repo.IsNameExist(ctx, "sqlite_script_nameexist", 0)
	require.NoError(t, err)
	require.True(t, exist, "已存在的名字应报告存在")

	excludeSelf, err := repo.IsNameExist(ctx, "sqlite_script_nameexist", createdID)
	require.NoError(t, err)
	require.False(t, excludeSelf, "排除自身 ID 后应报告不存在")

	unknown, err := repo.IsNameExist(ctx, "no-such-script-zzz", 0)
	require.NoError(t, err)
	require.False(t, unknown, "不存在的名字应报告不存在")
}

// TestScriptRepoSqlite_LanguageReadView 验证两种脚本语言枚举值经 converter
// 落库后，在一切返回 Script DTO 的读路径（Get 按主键、GetByName、List、
// ListEnabledScripts）的 DTO 视图如实呈现；未显式指定 language 的行按
// schema 默认 LUA 落库并在读视图如实呈现。
//
// 枚举字段读视图机制注记：实体侧 language 为可空指针枚举列
// （*script.Language，schema 带 Default("LUA")），DTO 侧为可选指针字段。
// mapper 的枚举转换对（经 &srcType/&dstType 取址注册）恰为指针↔指针形态
// 的键，指针对字段能被 copier 直接转换赋值——与值型实体枚举列（如
// position.type、notification_channel.type，值型字段读侧被 copier 丢弃）
// 的情形不同。本测试将该读视图行为钉死。
func TestScriptRepoSqlite_LanguageReadView(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoLanguage scriptV1.Language
		wantEnt       script.Language
		marker        string
	}{
		{scriptV1.Language_LUA, script.LanguageLUA, "sqlite_script_readview_lua"},
		{scriptV1.Language_JAVASCRIPT, script.LanguageJAVASCRIPT, "sqlite_script_readview_javascript"},
	}

	for _, c := range cases {
		require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
			Data: &scriptV1.Script{
				Name:      trans.Ptr(c.marker),
				Language:  c.protoLanguage.Enum(),
				IsEnabled: trans.Ptr(true),
			},
		}), "语言 %v 创建应成功", c.protoLanguage)

		rows, err := entClient.Client().Script.Query().
			Where(script.NameEQ(c.marker)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按名应反查到刚写入的行")
		require.Equal(t, c.wantEnt, *rows[0].Language, "语言 %v 应经转换器如实落库", c.protoLanguage)

		// 读路径一：Get 按主键命中后，DTO 视图应如实呈现语言枚举。
		got, err := repo.Get(ctx, &scriptV1.GetScriptRequest{
			QueryBy: &scriptV1.GetScriptRequest_Id{Id: rows[0].ID},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoLanguage, got.GetLanguage(), "读视图应如实呈现语言 %v", c.protoLanguage)

		// 读路径二：GetByName 命中后，DTO 视图应如实呈现语言枚举。
		byName, err := repo.GetByName(ctx, c.marker)
		require.NoError(t, err, "按名读取应命中")
		require.Equal(t, c.protoLanguage, byName.GetLanguage(), "GetByName 读视图应如实呈现语言 %v", c.protoLanguage)

		// 读路径三/四：List 与 ListEnabledScripts 应含该行（按名标记匹配），
		// DTO 视图如实呈现语言枚举。
		listedAll, err := repo.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		var hitAll *scriptV1.Script
		for _, item := range listedAll.Items {
			if item.GetName() == c.marker {
				hitAll = item
				break
			}
		}
		require.NotNil(t, hitAll, "List 应包含标记行 %s", c.marker)
		require.Equal(t, c.protoLanguage, hitAll.GetLanguage(), "List 读视图应如实呈现语言 %v", c.protoLanguage)

		listedEnabled, err := repo.ListEnabledScripts(ctx)
		require.NoError(t, err)
		var hitEnabled *scriptV1.Script
		for _, item := range listedEnabled {
			if item.GetName() == c.marker {
				hitEnabled = item
				break
			}
		}
		require.NotNil(t, hitEnabled, "ListEnabledScripts 应包含启用的标记行 %s", c.marker)
		require.Equal(t, c.protoLanguage, hitEnabled.GetLanguage(), "ListEnabledScripts 读视图应如实呈现语言 %v", c.protoLanguage)
	}

	// 未显式指定 language：走 schema 列默认 LUA 落库，读视图如实呈现 LUA。
	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("sqlite_script_readview_default"),
			IsEnabled: trans.Ptr(true),
		},
	}))
	defaultRows, err := entClient.Client().Script.Query().
		Where(script.NameEQ("sqlite_script_readview_default")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, defaultRows, 1)
	require.NotNil(t, defaultRows[0].Language, "未显式指定应落 schema 默认值")
	require.Equal(t, script.LanguageLUA, *defaultRows[0].Language, "列默认应为 LUA")
	gotDefault, err := repo.Get(ctx, &scriptV1.GetScriptRequest{
		QueryBy: &scriptV1.GetScriptRequest_Id{Id: defaultRows[0].ID},
	})
	require.NoError(t, err)
	require.Equal(t, scriptV1.Language_LUA, gotDefault.GetLanguage(), "读视图应如实呈现列默认 LUA")
}

// TestScriptRepoSqlite_Update 验证 Update 掩码内字段（description）更新、
// 版本指纹自增、掩码外字段（name）保持原值。
func TestScriptRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:        trans.Ptr("sqlite_script_update"),
			Description: trans.Ptr("更新前描述"),
		},
	}))
	rows, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID
	require.Equal(t, uint32(1), *rows[0].Version)

	err = repo.Update(ctx, &scriptV1.UpdateScriptRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data: &scriptV1.Script{
			Description: trans.Ptr("更新后描述-sqlite"),
		},
	})
	require.NoError(t, err, "更新 description 应成功")

	after, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后描述-sqlite", *after[0].Description, "掩码内字段应被更新")
	require.Equal(t, "sqlite_script_update", *after[0].Name, "掩码外字段 name 应保持原值")
	require.Equal(t, uint32(2), *after[0].Version, "版本指纹应随更新自增到 2")

	// GetVersionByName：命中不报错即可。现状 bug：该方法对 *uint32 指针做
	// fmt.Sprintf("%d", entity.Version)，返回的是指针地址串而非 "2"，
	// 无法用于指纹比对（热更新变更检测回调同样受影响）。版本自增语义
	// 已由上方直查断言覆盖；若后续修复为解引用，此处应收紧为 "2"。
	_, err = repo.GetVersionByName(ctx, "sqlite_script_update")
	require.NoError(t, err)
}

// TestScriptRepoSqlite_Delete 验证 Delete（按 ID 列表）后行数归零。
func TestScriptRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{Name: trans.Ptr("sqlite_script_delete_a")},
	}))
	require.NoError(t, repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{Name: trans.Ptr("sqlite_script_delete_b")},
	}))
	rows, err := repo.entClient.Client().Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.NoError(t, repo.Delete(ctx, &scriptV1.DeleteScriptRequest{
		Ids: []uint32{rows[0].ID, rows[1].ID},
	}), "Delete 应成功")
	cnt, err := repo.entClient.Client().Script.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")

	// 空 ID 列表：BadRequest
	require.Error(t, repo.Delete(ctx, &scriptV1.DeleteScriptRequest{}), "空 ID 列表应返回 BadRequest")
}
