package data

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestServerMonitorRepoSqlite_GetInfoWithDatabase 验证 GetInfo 的聚合：
// Go 运行时段、主机段与数据库段（SQLite 内存库：driver 标识与 ping 成功）
// 均被填充，collected_at 非空。
func TestServerMonitorRepoSqlite_GetInfoWithDatabase(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := &ServerMonitorRepo{
		log:        bLogger.NewHelper(bLogger.NopLogger()),
		db:         entClient.DB(),
		driverName: "sqlite",
		startTime:  time.Now().Add(-time.Minute),
	}
	ctx := enttest.NewSystemViewerCtx(context.Background())

	info, err := repo.GetInfo(ctx)
	require.NoError(t, err, "GetInfo 应成功")
	require.NotNil(t, info, "GetInfo 应返回非空 info")
	require.NotNil(t, info.GetCollectedAt(), "collected_at 应被填充")

	// Go 运行时段：与 runtime 实际值一致
	goInfo := info.GetGo()
	require.NotNil(t, goInfo, "go 段应被填充")
	require.Equal(t, runtime.Version(), goInfo.GetVersion(), "go 版本应等于 runtime.Version()")
	// goroutine 数为运行期瞬态值（两次调用间可能增减），只断言被填充为正数。
	require.Greater(t, goInfo.GetNumGoroutine(), uint32(0), "goroutine 数应被填充为正数")
	require.NotNil(t, goInfo.GetMemAllocBytes(), "内存分配字段应被填充")
	require.NotNil(t, goInfo.GetMemSysBytes(), "系统内存字段应被填充")
	require.NotNil(t, goInfo.GetGcCycles(), "GC 周期字段应被填充")
	require.NotNil(t, goInfo.GetStartedAt(), "started_at 应被填充")
	require.GreaterOrEqual(t, goInfo.GetUptimeSeconds(), uint64(0), "uptime 应非负")

	// 主机段：与运行环境一致（hostname 受环境影响，不 assert）
	hostInfo := info.GetHost()
	require.NotNil(t, hostInfo, "host 段应被填充")
	require.Equal(t, runtime.GOOS, hostInfo.GetOs(), "os 应等于 runtime.GOOS")
	require.Equal(t, runtime.GOARCH, hostInfo.GetArch(), "arch 应等于 runtime.GOARCH")
	require.Equal(t, uint32(runtime.NumCPU()), hostInfo.GetNumCpu(), "cpu 数应等于 runtime.NumCPU()")

	// 数据库段：SQLite 内存库 ping 成功
	dbInfo := info.GetDatabase()
	require.NotNil(t, dbInfo, "database 段应被填充")
	require.Equal(t, "sqlite", dbInfo.GetDriver(), "driver 应为构造传入的驱动名")
	require.True(t, dbInfo.GetPingOk(), "SQLite 内存库 ping 应成功")
	require.Empty(t, dbInfo.GetPingError(), "ping 成功时不应有错误串")
}

// TestServerMonitorRepoSqlite_GetInfoWithoutDatabase 验证 db 未配置时
// 数据库段返回 "database client is not configured"、ping 不成功、driver 为空，
// 其余段不受影响。
func TestServerMonitorRepoSqlite_GetInfoWithoutDatabase(t *testing.T) {
	repo := &ServerMonitorRepo{
		log:        bLogger.NewHelper(bLogger.NopLogger()),
		db:         nil,
		driverName: "",
		startTime:  time.Now(),
	}
	ctx := enttest.NewSystemViewerCtx(context.Background())

	info, err := repo.GetInfo(ctx)
	require.NoError(t, err, "GetInfo 应成功")
	require.NotNil(t, info)

	dbInfo := info.GetDatabase()
	require.NotNil(t, dbInfo, "database 段应被填充")
	require.Equal(t, "database client is not configured", dbInfo.GetPingError(),
		"db 为 nil 时应报告 client 未配置")
	require.False(t, dbInfo.GetPingOk(), "db 为 nil 时 ping 不应成功")
	require.Empty(t, dbInfo.GetDriver(), "db 为 nil 时 driver 应为空")
	require.Zero(t, dbInfo.GetMaxOpenConnections(), "db 为 nil 时连接池统计应为零值")
	require.Zero(t, dbInfo.GetOpenConnections(), "db 为 nil 时连接池统计应为零值")
	require.Zero(t, dbInfo.GetInUseConnections(), "db 为 nil 时连接池统计应为零值")
	require.Zero(t, dbInfo.GetIdleConnections(), "db 为 nil 时连接池统计应为零值")

	// 其它段不受影响
	require.NotNil(t, info.GetGo(), "go 段仍应被填充")
	require.NotNil(t, info.GetHost(), "host 段仍应被填充")
	require.NotNil(t, info.GetCollectedAt(), "collected_at 仍应被填充")
}
