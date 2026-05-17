package data

import (
	"go-wind-admin/pkg/serviceid"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/captcha"
	"github.com/tx7do/go-utils/password"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	redisClient "github.com/tx7do/kratos-bootstrap/cache/redis"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/oss"
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

	cli := redisClient.NewClient(cfg.Data, l)

	return cli, func() {
		if err := cli.Close(); err != nil {
			l.Error(err)
		}
	}, nil
}

func NewMinIoClient(ctx *bootstrap.Context) *oss.MinIOClient {
	return oss.NewMinIoClient(ctx.GetConfig(), ctx.GetLogger())
}

func NewPasswordCrypto() password.Crypto {
	crypto, err := password.CreateCrypto("bcrypt")
	if err != nil {
		panic(err)
	}
	return crypto
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
