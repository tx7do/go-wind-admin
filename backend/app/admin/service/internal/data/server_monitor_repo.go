package data

import (
	"context"
	"database/sql"
	"os"
	"runtime"
	"time"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	serverMonitorV1 "go-wind-admin/api/gen/go/server_monitor/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
)

// ServerMonitorRepo 服务运行时只读监控视图（Go 运行时 / 数据库 / 主机）。
// 与 RedisCacheMonitorRepo 同型的纯透传聚合，不持有独立存储。
type ServerMonitorRepo struct {
	log *bLogger.Helper

	db         *sql.DB
	driverName string

	// startTime 进程启动时间：运行时长基准
	startTime time.Time
}

func NewServerMonitorRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *ServerMonitorRepo {
	cfg := ctx.GetConfig()
	var driverName string
	if cfg != nil && cfg.Data != nil && cfg.Data.Database != nil {
		driverName = cfg.Data.Database.GetDriver()
	}

	return &ServerMonitorRepo{
		log:        ctx.NewLoggerHelper("server-monitor/repo"),
		db:         entClient.DB(),
		driverName: driverName,
		startTime:  time.Now(),
	}
}

// GetInfo 聚合 Go 运行时 / 数据库 / 主机信息。
// 任一子项失败不影响其它子项（分别兜底为空段并记录错误）。
func (r *ServerMonitorRepo) GetInfo(ctx context.Context) (*serverMonitorV1.ServerMonitorInfo, error) {
	info := &serverMonitorV1.ServerMonitorInfo{
		Go:          r.goRuntimeInfo(),
		Host:        r.hostInfo(),
		CollectedAt: timestamppb.Now(),
	}
	info.Database = r.databaseInfo(ctx)
	return info, nil
}

func (r *ServerMonitorRepo) goRuntimeInfo() *serverMonitorV1.GoRuntimeInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return &serverMonitorV1.GoRuntimeInfo{
		Version:       strPtr(runtime.Version()),
		NumGoroutine:  u32Ptr(uint32(runtime.NumGoroutine())),
		MemAllocBytes: u64Ptr(ms.Alloc),
		MemSysBytes:   u64Ptr(ms.Sys),
		GcCycles:      u32Ptr(uint32(ms.NumGC)),
		UptimeSeconds: u64Ptr(uint64(time.Since(r.startTime).Seconds())),
		StartedAt:     timestamppb.New(r.startTime),
	}
}

func (r *ServerMonitorRepo) databaseInfo(ctx context.Context) *serverMonitorV1.DatabaseInfo {
	info := &serverMonitorV1.DatabaseInfo{
		Driver: strPtr(r.driverName),
	}

	if r.db == nil {
		info.PingError = strPtr("database client is not configured")
		return info
	}

	stats := r.db.Stats()
	info.MaxOpenConnections = u32Ptr(uint32(stats.MaxOpenConnections))
	info.OpenConnections = u32Ptr(uint32(stats.OpenConnections))
	info.InUseConnections = u32Ptr(uint32(stats.InUse))
	info.IdleConnections = u32Ptr(uint32(stats.Idle))

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := r.db.PingContext(pingCtx); err != nil {
		info.PingOk = boolPtr(false)
		info.PingError = strPtr(err.Error())
		r.log.Errorf(ctx, "database ping failed: %v", err)
		return info
	}

	info.PingOk = boolPtr(true)
	return info
}

func (r *ServerMonitorRepo) hostInfo() *serverMonitorV1.HostInfo {
	hostname, _ := os.Hostname()
	return &serverMonitorV1.HostInfo{
		Os:       strPtr(runtime.GOOS),
		Arch:     strPtr(runtime.GOARCH),
		NumCpu:   u32Ptr(uint32(runtime.NumCPU())),
		Hostname: strPtr(hostname),
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func u32Ptr(v uint32) *uint32 { return &v }
func u64Ptr(v uint64) *uint64 { return &v }
