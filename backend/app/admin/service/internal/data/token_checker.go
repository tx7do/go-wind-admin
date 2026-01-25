package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type TokenChecker struct {
	log *log.Helper

	authenticator *Authenticator
	clientType    authenticationV1.ClientType
}

func NewTokenChecker(
	ctx *bootstrap.Context,
	authenticator *Authenticator,
	clientType authenticationV1.ClientType,
) auth.AccessTokenChecker {
	return &TokenChecker{
		log:           log.NewHelper(log.With(ctx.GetLogger(), "module", "token-checker/auth/middleware")),
		authenticator: authenticator,
		clientType:    clientType,
	}
}

// IsValidAccessToken checks if the access token is valid for the given user ID.
func (tc *TokenChecker) IsValidAccessToken(ctx context.Context, accessToken string, skipRedis bool) (bool, *authenticationV1.UserTokenPayload) {
	resp, err := tc.authenticator.Authenticate(ctx, &authenticationV1.ValidateTokenRequest{
		Token:         accessToken,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
		ClientType:    tc.clientType,
		SkipRedis:     trans.Ptr(skipRedis),
	})
	if err != nil {
		return false, nil
	}

	if !resp.IsValid {
		return false, nil
	}

	return true, resp.Payload
}

// IsBlockedAccessToken checks if the access token is blocked for the given user ID.
func (tc *TokenChecker) IsBlockedAccessToken(ctx context.Context, accessToken string) bool {
	resp, err := tc.authenticator.Authenticate(ctx, &authenticationV1.ValidateTokenRequest{
		Token:         accessToken,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
		ClientType:    tc.clientType,
		SkipRedis:     trans.Ptr(true),
	})
	if err != nil {
		return true
	}
	return !resp.IsValid
}
