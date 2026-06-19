package service

import (
	"context"
	"testing"
)

// panicGenContentRepo 的 Create 会 panic，用于验证采集器 fail-open：即便脱敏后写库 panic，
// Collect 也必须 recover、正常返回，绝不向上抛、绝不影响主调用/计费（判断基准3）。
type panicGenContentRepo struct{}

func (panicGenContentRepo) Create(context.Context, *GenerationContent) error {
	panic("boom in repo.Create")
}

func TestCollectorRecoversRepoPanic(t *testing.T) {
	collector := NewGenerationContentCollector(panicGenContentRepo{}, enabledContentCaptureCfg())
	// 若 panic 未被 recover，测试会直接 crash；能执行到下一行即证明已兜底。
	collector.Collect(context.Background(), GenerationContentCaptureArgs{
		RequestID:  "panic-1",
		PromptBody: []byte(`{"messages":[{"role":"user","content":"hi"}]}`),
		Result:     &ForwardResult{ResponseSample: []byte("partial")},
	})
}
