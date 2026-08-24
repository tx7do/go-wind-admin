package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-transport/transport/asynq"
	"github.com/tx7do/kratos-transport/transport/sse"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	//_ "github.com/tx7do/kratos-bootstrap/config/consul"
	//_ "github.com/tx7do/kratos-bootstrap/config/etcd"
	//_ "github.com/tx7do/kratos-bootstrap/config/kubernetes"
	//_ "github.com/tx7do/kratos-bootstrap/config/nacos"
	//_ "github.com/tx7do/kratos-bootstrap/config/polaris"

	//_ "github.com/tx7do/kratos-bootstrap/logger/aliyun"
	//_ "github.com/tx7do/kratos-bootstrap/logger/fluent"
	//_ "github.com/tx7do/kratos-bootstrap/logger/logrus"
	//_ "github.com/tx7do/kratos-bootstrap/logger/tencent"
	//_ "github.com/tx7do/kratos-bootstrap/logger/zap"
	//_ "github.com/tx7do/kratos-bootstrap/logger/zerolog"

	//_ "github.com/tx7do/kratos-bootstrap/registry/consul"
	//_ "github.com/tx7do/kratos-bootstrap/registry/etcd"
	//_ "github.com/tx7do/kratos-bootstrap/registry/eureka"
	//_ "github.com/tx7do/kratos-bootstrap/registry/kubernetes"
	//_ "github.com/tx7do/kratos-bootstrap/registry/nacos"
	//_ "github.com/tx7do/kratos-bootstrap/registry/polaris"
	//_ "github.com/tx7do/kratos-bootstrap/registry/servicecomb"
	//_ "github.com/tx7do/kratos-bootstrap/registry/zookeeper"

	//_ "github.com/tx7do/kratos-bootstrap/tracer"

	appCrypto "go-wind-admin/pkg/crypto"
	"go-wind-admin/pkg/serviceid"
)

var version = "1.0.0"

// go build -ldflags "-X main.version=x.y.z"

func newApp(
	ctx *bootstrap.Context,
	hs *http.Server,
	as *asynq.Server,
	ss *sse.Server,
) *kratos.App {
	// asynq / sse 在配置缺失时返回 nil（typed-nil）。
	// kratos 的 app.Run() 会对每个 server 调 Start()/Stop()，
	// 把 typed-nil 指针当作 transport.Server 解引用会 panic。
	// 因此这里跳过未配置的服务器，使"未配置 asynq/SSE"的部署能正常启动。
	servers := []transport.Server{hs}
	if as != nil {
		servers = append(servers, as)
	}
	if ss != nil {
		servers = append(servers, ss)
	}
	return bootstrap.NewApp(ctx, servers...)
}

func runApp() error {
	// 初始化全局加密器（供 task payload 等路径对敏感配置做 AES-GCM 加密）。
	// 通过环境变量 GOWIND_CRYPTO_KEY 提供密钥；未设置时加密功能禁用（no-op）。
	// 注意：EncryptPayload 只在真正加密时才标记 IsEncryptedKey，因此未配置密钥
	// 时 payload 以明文存储且不会被误判为已加密——调用方无需额外处理。
	initGlobalEncryptorFromEnv()

	ctx := bootstrap.NewContext(
		context.Background(),
		&conf.AppInfo{
			Project: serviceid.ProjectName,
			AppId:   serviceid.AdminService,
			Version: version,
		},
	)
	return bootstrap.RunApp(ctx, initApp)
}

// initGlobalEncryptorFromEnv 从环境变量读取加密密钥并初始化全局加密器。
func initGlobalEncryptorFromEnv() {
	key := os.Getenv("GOWIND_CRYPTO_KEY")
	if key == "" {
		bLogger.GetLogger().Warn(context.Background(), "GOWIND_CRYPTO_KEY not set, global encryption disabled (payloads stored in plaintext)")
		return
	}
	if err := appCrypto.InitGlobalEncryptor(key, true); err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("init global encryptor failed, encryption disabled: %v", err))
	}
}

func main() {
	if err := runApp(); err != nil {
		panic(err)
	}
}
