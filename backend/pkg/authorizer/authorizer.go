package authorizer

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	"github.com/tx7do/kratos-authz/engine/casbin"
	"github.com/tx7do/kratos-authz/engine/noop"
	"github.com/tx7do/kratos-authz/engine/opa"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// Authorizer 权限管理器
type Authorizer struct {
	log *log.Helper

	engine   authzEngine.Engine
	provider Provider
}

func NewAuthorizer(
	ctx *bootstrap.Context,
	provider Provider,
) *Authorizer {
	a := &Authorizer{
		log:      ctx.NewLoggerHelper("authorizer"),
		provider: provider,
	}

	if ctx == nil {
		a.log.Warn("bootstrap context is nil")
		return a
	}

	if ctx.GetConfig() == nil || ctx.GetConfig().Authz == nil {
		a.log.Warn("authorization config is nil")
		return a
	}

	a.init(ctx.Context(), ctx.GetConfig().Authz)

	return a
}

func (a *Authorizer) init(ctx context.Context, cfg *conf.Authorization) {
	a.engine = a.newEngine(ctx, cfg)

	//if err := a.ResetPolicies(ctx); err != nil {
	//	a.log.Errorf("reset policies error: %v", err)
	//}
}

func (a *Authorizer) Engine() authzEngine.Engine {
	return a.engine
}

// ResetPolicies 重置策略
func (a *Authorizer) ResetPolicies(ctx context.Context) error {
	//a.log.Info("*******************reset policies")

	result, err := a.provider.ProvidePolicies(ctx)
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
func (a *Authorizer) generateCasbinPolicies(data PermissionDataMap) (authzEngine.PolicyMap, error) {
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
func (a *Authorizer) generateOpaPolicies(data PermissionDataMap) (authzEngine.PolicyMap, error) {
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

// newEngine 创建权限引擎
func (a *Authorizer) newEngine(ctx context.Context, cfg *conf.Authorization) authzEngine.Engine {
	if cfg == nil {
		return nil
	}

	switch cfg.GetType() {
	default:
		fallthrough
	case "noop":
		return a.newEngineNoop(ctx)

	case "casbin":
		return a.newEngineCasbin(ctx)

	case "opa":
		return a.newEngineOPA(ctx)

	case "zanzibar":
		return a.newEngineZanzibar(ctx)
	}
}

// newEngineZanzibar 创建 Zanzibar 引擎（未实现）
func (a *Authorizer) newEngineZanzibar(_ context.Context) authzEngine.Engine {
	return nil
}

// newEngineNoop 创建 Noop 引擎
func (a *Authorizer) newEngineNoop(ctx context.Context) authzEngine.Engine {
	state, err := noop.NewEngine(ctx)
	if err != nil {
		a.log.Errorf("new noop engine error: %v", err)
		return nil
	}
	return state
}

// newEngineCasbin 创建 Casbin 引擎
func (a *Authorizer) newEngineCasbin(ctx context.Context) authzEngine.Engine {
	state, err := casbin.NewEngine(ctx)
	if err != nil {
		a.log.Errorf("init casbin engine error: %v", err)
		return nil
	}
	return state
}

// newEngineOPA 创建 OPA 引擎
func (a *Authorizer) newEngineOPA(ctx context.Context) authzEngine.Engine {
	modelName := "rbac.rego"
	models := a.provider.ProvideModels("opa")
	var model []byte
	var ok bool
	if model, ok = models[modelName]; ok {
		a.log.Infof("load custom OPA model: %s", modelName)
	} else {
		a.log.Errorf("OPA model not found: %s", modelName)
		return nil
	}

	state, err := opa.NewEngine(ctx,
		opa.WithModulesFromString(map[string]string{
			modelName: string(model),
		}),
	)
	if err != nil {
		a.log.Errorf("init opa engine error: %v", err)
		return nil
	}

	if err = state.InitModulesFromString(map[string]string{
		modelName: string(model),
	}); err != nil {
		a.log.Errorf("init opa modules error: %v", err)
	}

	return state
}
