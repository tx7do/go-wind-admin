package auth

import (
	"context"
	"reflect"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/go-utils/trans"
	authzEngine "github.com/tx7do/kratos-authz/engine"
	authz "github.com/tx7do/kratos-authz/middleware"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

func processAuthz(
	ctx context.Context,
	tr transport.Transporter,
	tokenPayload *authenticationV1.UserTokenPayload,
) (context.Context, error) {
	path := authzEngine.Resource(tr.Operation())
	action := defaultAction

	var htr *http.Transport
	var ok bool
	if htr, ok = tr.(*http.Transport); ok {
		path = authzEngine.Resource(htr.PathTemplate())
		action = authzEngine.Action(htr.Request().Method)
	}

	//log.Infof("Coming API Request: PATH[%s] ACTION[%s] USER ROLES[%v] USER ID[%d]",
	//	path, action, tokenPayload.GetRoles(), tokenPayload.UserId,
	//)

	authzClaims := authzEngine.AuthClaims{
		Subjects: trans.Ptr(tokenPayload.GetRoles()),
		Action:   trans.Ptr(action),
		Resource: trans.Ptr(path),
	}

	ctx = authz.NewContext(ctx, &authzClaims)

	return ctx, nil
}

func setRequestOperationId(req interface{}, payload *authenticationV1.UserTokenPayload) error {
	if req == nil {
		return ErrInvalidRequest
	}

	v := reflect.ValueOf(req).Elem()
	field := v.FieldByName("OperatorId")
	if field.IsValid() && field.Kind() == reflect.Pointer {
		field.Set(reflect.ValueOf(&payload.UserId))
	}

	return nil
}

func setRequestTenantId(req interface{}, payload *authenticationV1.UserTokenPayload) error {
	if req == nil {
		return ErrInvalidRequest
	}

	v := reflect.ValueOf(req).Elem()
	// 此前误用小写 "tenantId"：proto 生成的导出字段为 "TenantId"，小写名匹配不到
	// 会导致 FieldByName 返回 invalid Value、注入被静默跳过。修正为正确的导出字段名。
	field := v.FieldByName("TenantId")
	if field.IsValid() && field.Kind() == reflect.Pointer && field.CanSet() {
		// payload.TenantId 本身就是 *uint64（proto optional 字段），
		// 再取地址会得到 **uint64，reflect.Set 直接 panic。
		field.Set(reflect.ValueOf(payload.TenantId))
	}

	return nil
}
