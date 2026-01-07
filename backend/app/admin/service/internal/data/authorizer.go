package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	"github.com/tx7do/kratos-authz/engine/casbin"
	"github.com/tx7do/kratos-authz/engine/noop"
	"github.com/tx7do/kratos-authz/engine/opa"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/cmd/server/assets"
)

// Authorizer 权限管理器
type Authorizer struct {
	log *log.Helper

	engine   authzEngine.Engine
	provider *AuthorizerProvider
}

func NewAuthorizer(
	ctx *bootstrap.Context,
	provider *AuthorizerProvider,
) *Authorizer {
	a := &Authorizer{
		log:      ctx.NewLoggerHelper("authorizer/data/admin-service"),
		provider: provider,
	}

	a.init(ctx.GetConfig())

	return a
}

func (a *Authorizer) init(cfg *conf.Bootstrap) {
	a.engine = a.newEngine(cfg)

	//if err := a.ResetPolicies(context.Background()); err != nil {
	//	a.log.Errorf("reset policies error: %v", err)
	//}
}

func (a *Authorizer) newEngine(cfg *conf.Bootstrap) authzEngine.Engine {
	if cfg.Authz == nil {
		return nil
	}

	ctx := context.Background()

	switch cfg.GetAuthz().GetType() {
	default:
		fallthrough
	case "noop":
		state, err := noop.NewEngine(ctx)
		if err != nil {
			a.log.Errorf("new noop engine error: %v", err)
			return nil
		}
		return state

	case "casbin":
		state, err := casbin.NewEngine(ctx, casbin.WithStringModel(string(assets.OpaRbacRego)))
		if err != nil {
			a.log.Errorf("init casbin engine error: %v", err)
			return nil
		}
		return state

	case "opa":
		state, err := opa.NewEngine(ctx,
			opa.WithModulesFromString(map[string]string{
				"rbac.rego": string(assets.OpaRbacRego),
			}),
		)
		if err != nil {
			a.log.Errorf("init opa engine error: %v", err)
			return nil
		}

		if err = state.InitModulesFromString(map[string]string{
			"rbac.rego": string(assets.OpaRbacRego),
		}); err != nil {
			a.log.Errorf("init opa modules error: %v", err)
		}

		return state

		//case "zanzibar":
		//	state, err := zanzibar.NewEngine(ctx)
		//	if err != nil {
		//		return nil
		//	}
		//	return state
	}
}

func (a *Authorizer) Engine() authzEngine.Engine {
	return a.engine
}

// ResetPolicies 重置策略
func (a *Authorizer) ResetPolicies(ctx context.Context) error {
	//a.log.Info("*******************reset policies")

	result, err := a.provider.Provide(ctx)
	if err != nil {
		a.log.Errorf("provide authorizer data error: %v", err)
		return err
	}

	//a.log.Debugf("roles [%d] apis [%d]", len(roles.Items), len(apis.Items))
	//a.log.Debugf("Generating policies for engine: %s", a.engine.Name())

	var policies authzEngine.PolicyMap

	switch a.engine.Name() {
	case "casbin":
		if policies, err = a.generateCasbinPolicies(result); err != nil {
			a.log.Errorf("generate casbin policies error: %v", err)
			return err
		}

	case "opa":
		if policies, err = a.generateOpaPolicies(result); err != nil {
			a.log.Errorf("generate OPA policies error: %v", err)
			return err
		}

	case "noop":
		return nil

	default:
		err = fmt.Errorf("unknown engine name: %s", a.engine.Name())
		a.log.Warnf(err.Error())
		return err
	}

	//a.log.Debugf("***************** policy rules len: %v", len(policies))

	if err = a.engine.SetPolicies(ctx, policies, nil); err != nil {
		a.log.Errorf("set policies error: %v", err)
		return err
	}

	a.log.Infof("reloaded policy rules [%d] successfully for engine: %s", len(policies), a.engine.Name())

	return nil
}

// generateCasbinPolicies 生成 Casbin 策略
func (a *Authorizer) generateCasbinPolicies(data AuthorizerDataMap) (authzEngine.PolicyMap, error) {
	var rules []casbin.PolicyRule

	for roleCode, aRules := range data {
		for _, api := range aRules {
			rules = append(rules, casbin.PolicyRule{
				PType: "p",
				V0:    roleCode,
				V1:    api.Path,
				V2:    api.Method,
				V3:    api.Domain,
			})
		}
	}

	policies := authzEngine.PolicyMap{
		"policies": rules,
		"projects": authzEngine.MakeProjects(),
	}

	return policies, nil
}

// generateOpaPolicies 生成 OPA 策略
func (a *Authorizer) generateOpaPolicies(data AuthorizerDataMap) (authzEngine.PolicyMap, error) {
	type OpaPolicyPath struct {
		Pattern string `json:"pattern"`
		Method  string `json:"method"`
	}

	policies := make(authzEngine.PolicyMap, len(data))

	for roleCode, aRule := range data {
		paths := make([]OpaPolicyPath, 0, len(aRule))

		for _, api := range aRule {
			paths = append(paths, OpaPolicyPath{
				Pattern: api.Path,
				Method:  api.Method,
			})

			//a.log.Debugf("OPA Policy - Role: [%s], Path: [%s], Method: [%s]", roleCode, api.Path, api.Method)
		}

		policies[roleCode] = paths
	}

	return policies, nil
}
