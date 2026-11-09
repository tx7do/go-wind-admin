package scripting

import (
	"context"
	"testing"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/pkg/scripting/api"
)

// 复现 TestRun 沙箱链路：NewEngine → SetAICompleter → Execute 引用 ai 模块的脚本。
type stubCompleter struct{}

func (stubCompleter) ChatForScript(_ context.Context, _ uint64, _, content string) (string, error) {
	return "echo:" + content, nil
}

func TestSandboxAIModule(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EngineType = "javascript"
	cfg.ScriptDir = ""
	logger := bLogger.GetLogger()
	sandbox := NewEngine(cfg, logger)
	sandbox.SetAICompleter(stubCompleter{})
	defer sandbox.Close()

	script := &Script{
		Name:    "ai_smoke",
		Source:  `const reply = ai.chat('hi'); __set_ctx('reply', reply); __set_ctx('typeofAI', typeof ai);`,
		Enabled: true,
	}
	execCtx := NewContext("test")
	if err := sandbox.Execute(context.Background(), script, execCtx); err != nil {
		t.Fatalf("execute: %v", err)
	}
	t.Logf("typeofAI=%v typeofLog=%v", execCtx.Get("typeofAI"), execCtx.Get("typeofLog"))
	if got := execCtx.Get("reply"); got != "echo:hi" {
		t.Fatalf("reply = %v, want echo:hi", got)
	}
	_ = time.Second
	_ = api.ModuleAI
}
