package data

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	configV1 "go-wind-admin/api/gen/go/config/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newConfigRepoSqlite 用 enttest helper 白盒构造一个可直接做 CRUD 的 ConfigRepo
// （同 position_repo_sqlite_test.go 的套路）。
func newConfigRepoSqlite(t *testing.T) *ConfigRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	repo := &ConfigRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[configV1.Config, ent.SysConfig](),
		valueTypeConverter: mapper.NewEnumTypeConverter[configV1.Config_ConfigValueType, sysconfig.ValueType](
			configV1.Config_ConfigValueType_name, configV1.Config_ConfigValueType_value,
		),
		cache: make(map[string]sysConfigCacheEntry),
	}
	repo.init()
	return repo
}

func newConfigRepoCtx() context.Context {
	return enttest.NewSystemViewerCtx(context.Background())
}

// TestConfigRepoSqlite_AccessorTypedReads 端到端验证参数读取器：三种类型按声明类型解析，
// 未声明类型（proto 零值守卫路径）落库为默认 STRING。
func TestConfigRepoSqlite_AccessorTypedReads(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	err := repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Name:      trans.Ptr("是否开启验证码"),
			Key:       trans.Ptr("sys.login.captchaEnabled"),
			Value:     trans.Ptr("true"),
			ValueType: configV1.Config_BOOL.Enum(),
		},
	})
	require.NoError(t, err, "创建 BOOL 参数应成功")
	require.True(t, repo.GetConfigBool(ctx, "sys.login.captchaEnabled", false), "GetConfigBool 应读到 true")

	err = repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Name:      trans.Ptr("口令最小长度"),
			Key:       trans.Ptr("sys.password.minLen"),
			Value:     trans.Ptr("12"),
			ValueType: configV1.Config_INT.Enum(),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 12, repo.GetConfigInt(ctx, "sys.password.minLen", 8), "GetConfigInt 应读到 12")

	// 未声明 value_type：Create 走零值守卫跳过设置，落库为 schema 默认 STRING
	err = repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Key:   trans.Ptr("sys.demo.plainString"),
			Value: trans.Ptr("hello"),
		},
	})
	require.NoError(t, err, "不声明 value_type 的创建不应触发枚举校验失败")
	require.Equal(t, "hello", repo.GetConfigString(ctx, "sys.demo.plainString", ""), "默认 STRING 类型应按字符串读出")
	require.Equal(t, 8, repo.GetConfigInt(ctx, "sys.demo.plainString", 8), "STRING 参数按 int 读应回退默认值")
}

// TestConfigRepoSqlite_ValueTypeReadView 验证三种参数值类型枚举经 converter
// 落库后，在读路径（Get 按主键 / List）的 DTO 视图如实呈现；未显式指定
// value_type 的行经写路径零值守卫走 schema 列默认 STRING 落库，读视图如实
// 呈现 STRING。同时联动断言读取器（带缓存）不受影响。
//
// 枚举字段读视图机制注记：实体侧 value_type 为可空指针枚举列
// （*sysconfig.ValueType，schema 默认 STRING），DTO 侧为可选指针字段。
// mapper 的枚举转换对（经 &srcType/&dstType 取址注册）恰为指针↔指针形态
// 的键，指针对字段能被 copier 直接转换赋值——与值型实体枚举列（如
// position.type、notification_channel.type）读侧被丢弃的情形不同。
// 本测试将该读视图行为钉死；缓存读取器（GetConfigXxx）走独立实体查询，
// 不经 DTO，语义不受读视图行为影响，此处一并钉死。
func TestConfigRepoSqlite_ValueTypeReadView(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	cases := []struct {
		// input 为创建请求显式指定的值类型（nil 表示不显式指定、走列默认）
		input *configV1.Config_ConfigValueType
		// expectedRead 为落库后读视图（Get/List）应呈现的值类型——
		// 显式指定时即所指定值；未指定时为 schema 列默认 STRING 落库后的如实值
		expectedRead configV1.Config_ConfigValueType
		wantEnt      sysconfig.ValueType
		key          string
		value        string
	}{
		{configV1.Config_STRING.Enum(), configV1.Config_STRING, sysconfig.ValueTypeString, "sys.readview.string", "hello"},
		{configV1.Config_BOOL.Enum(), configV1.Config_BOOL, sysconfig.ValueTypeBool, "sys.readview.bool", "true"},
		{configV1.Config_INT.Enum(), configV1.Config_INT, sysconfig.ValueTypeInt, "sys.readview.int", "42"},
		// 未显式指定：写路径零值守卫跳过 Set，ent 落 schema 列默认 STRING，
		// 读视图如实呈现落库值 STRING（而非输入缺省的 INVALID）
		{nil, configV1.Config_STRING, sysconfig.ValueTypeString, "sys.readview.default", "x"},
	}

	createdIDs := make(map[string]uint32, len(cases))
	for _, c := range cases {
		data := &configV1.Config{
			Key:   trans.Ptr(c.key),
			Value: trans.Ptr(c.value),
		}
		if c.input != nil {
			data.ValueType = c.input
		}
		require.NoError(t, repo.Create(ctx, &configV1.CreateConfigRequest{Data: data}),
			"键 %s 创建应成功", c.key)

		rows, err := repo.entClient.Client().SysConfig.Query().
			Where(sysconfig.KeyEQ(c.key)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按键应反查到刚写入的行")
		require.NotNil(t, rows[0].ValueType, "键 %s 的 value_type 应落库", c.key)
		require.Equal(t, c.wantEnt, *rows[0].ValueType, "键 %s 的 value_type 应如实落库（含列默认）", c.key)
		createdIDs[c.key] = rows[0].ID
	}

	// 读路径一：Get 按主键命中后，DTO 视图应如实呈现落库的值类型枚举。
	for _, c := range cases {
		got, err := repo.Get(ctx, &configV1.GetConfigRequest{
			QueryBy: &configV1.GetConfigRequest_Id{Id: createdIDs[c.key]},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.expectedRead, got.GetValueType(), "键 %s 读视图应如实呈现落库的值类型", c.key)
	}

	// 读路径二：List 的 DTO 视图应如实呈现各行落库的值类型（按 key 匹配）。
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	view := map[string]configV1.Config_ConfigValueType{}
	for _, item := range listed.Items {
		view[item.GetKey()] = item.GetValueType()
	}
	for _, c := range cases {
		got, ok := view[c.key]
		require.True(t, ok, "List 应包含键 %s", c.key)
		require.Equal(t, c.expectedRead, got, "键 %s 的 List 读视图应如实呈现落库的值类型", c.key)
	}

	// 联动断言：读取器（缓存路径）仍按落库声明的类型解析——BOOL 行读回 true。
	//（读取器走独立实体查询，不经 DTO；此处确认读视图行为与缓存语义互不影响。）
	require.True(t, repo.GetConfigBool(ctx, "sys.readview.bool", false), "BOOL 参数经读取器应读到 true（缓存语义不受影响）")
}

// TestConfigRepoSqlite_AccessorCacheInvalidation 验证写路径同步失效：
// Update/Delete 后读取器必须立即看到新值/回退默认值，不允许读到陈旧缓存。
func TestConfigRepoSqlite_AccessorCacheInvalidation(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	err := repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Key:       trans.Ptr("sys.cache.probe"),
			Value:     trans.Ptr("false"),
			ValueType: configV1.Config_BOOL.Enum(),
		},
	})
	require.NoError(t, err)
	require.False(t, repo.GetConfigBool(ctx, "sys.cache.probe", true), "首次读取应落缓存并读到 false")

	// Update（掩码只带 value，模拟前端部分字段编辑）后缓存必须失效
	err = repo.Update(ctx, &configV1.UpdateConfigRequest{
		Id: 1,
		Data: &configV1.Config{
			Value: trans.Ptr("true"),
		},
		UpdateMask: &field_mask.FieldMask{Paths: []string{"value"}},
	})
	require.NoError(t, err)
	require.True(t, repo.GetConfigBool(ctx, "sys.cache.probe", false), "Update 后应读到新值 true（缓存已失效）")

	// Delete 后读取应回退默认值
	err = repo.Delete(ctx, &configV1.DeleteConfigRequest{QueryBy: &configV1.DeleteConfigRequest_Id{Id: 1}})
	require.NoError(t, err)
	require.True(t, repo.GetConfigBool(ctx, "sys.cache.probe", true), "Delete 后应回退默认值 true")
}

// TestConfigRepoSqlite_AccessorMissingKey 验证负缓存与缺省回退。
func TestConfigRepoSqlite_AccessorMissingKey(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	require.Equal(t, "def", repo.GetConfigString(ctx, "sys.not.existing", "def"))
	require.Equal(t, "def", repo.GetConfigString(ctx, "sys.not.existing", "def"), "负缓存命中后第二次读取仍回默认")
	require.Equal(t, 7, repo.GetConfigInt(ctx, "sys.not.existing", 7))
}

// TestConfigRepoSqlite_BuiltInDeleteGuard 验证内置参数禁删、非内置可删。
func TestConfigRepoSqlite_BuiltInDeleteGuard(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	err := repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Key:       trans.Ptr("sys.guard.builtIn"),
			Value:     trans.Ptr("1"),
			ValueType: configV1.Config_INT.Enum(),
			IsBuiltIn: trans.Ptr(true),
		},
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Key:       trans.Ptr("sys.guard.normal"),
			Value:     trans.Ptr("2"),
			ValueType: configV1.Config_INT.Enum(),
		},
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, &configV1.DeleteConfigRequest{QueryBy: &configV1.DeleteConfigRequest_Id{Id: 1}})
	require.Error(t, err, "内置参数删除应被拒绝")
	require.True(t, strings.Contains(err.Error(), "cannot be deleted"), "拒绝原因应说明内置参数禁删")

	err = repo.Delete(ctx, &configV1.DeleteConfigRequest{QueryBy: &configV1.DeleteConfigRequest_Id{Id: 2}})
	require.NoError(t, err, "非内置参数应可删除")

	// 幂等删除：不存在目标视为已删除
	err = repo.Delete(ctx, &configV1.DeleteConfigRequest{QueryBy: &configV1.DeleteConfigRequest_Id{Id: 999}})
	require.NoError(t, err)
}

// TestConfigRepoSqlite_DuplicateKeyRejected 验证唯一键约束映射为 400。
func TestConfigRepoSqlite_DuplicateKeyRejected(t *testing.T) {
	repo := newConfigRepoSqlite(t)
	ctx := newConfigRepoCtx()

	err := repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{Key: trans.Ptr("sys.dup.key"), Value: trans.Ptr("a")},
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{Key: trans.Ptr("sys.dup.key"), Value: trans.Ptr("b")},
	})
	require.Error(t, err, "重复键创建应被拒绝")
	require.True(t, strings.Contains(err.Error(), "already exists"), "错误应说明键已存在")
}
