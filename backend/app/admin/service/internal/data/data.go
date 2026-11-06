package data

import (
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/captcha"
	"github.com/tx7do/go-utils/password"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	redisClient "github.com/tx7do/kratos-bootstrap/cache/redis"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/serviceid"
)

func NewClientType() authenticationV1.ClientType {
	return authenticationV1.ClientType_admin
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(ctx *bootstrap.Context) (*redis.Client, func(), error) {
	cfg := ctx.GetConfig()
	if cfg == nil {
		return nil, func() {}, nil
	}

	l := ctx.NewLoggerHelper("redis/data/admin-service")

	cli := redisClient.NewClient(cfg.Data, klog.NewHelper(bLogger.AsKratosLogger(l)))

	return cli, func() {
		if err := cli.Close(); err != nil {
			l.Error(ctx.Context(), err.Error())
		}
	}, nil
}

func NewMinIoClient(ctx *bootstrap.Context) *oss.MinIOClient {
	return oss.NewMinIoClient(ctx.GetConfig(), ctx.GetLogger())
}

func NewPasswordCrypto() password.Crypto {
	// P3 加固：cost 10 → 12（2026 年离线 GPU 撞库背景下 10 偏弱）。
	// bcrypt 哈希自带 cost 字段，Verify 兼容旧 cost=10 存量哈希（校验按哈希
	// 内嵌 cost 执行）；新口令/改密一律落 12。
	return password.NewBCryptCrypto(12)
}

func NewCaptcha(rdb *redis.Client) *captcha.Captcha {
	captchaInstance := captcha.NewCaptcha(rdb,
		captcha.WithDriverType(captcha.DriverString),
		captcha.WithExpire(10*time.Minute),
		captcha.WithKeyPrefix(serviceid.ProjectName+":captcha"),
		captcha.WithStringCount(6),
		captcha.WithStringSource("ABCDEFGHJKLMNPQRSTUVWXYZ23456789"),
	)
	return captchaInstance
}
