package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type fakeGenContentRepo struct {
	rows []*GenerationContent
	err  error
}

func (f *fakeGenContentRepo) Create(_ context.Context, c *GenerationContent) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, c)
	return nil
}

// 只读看板方法在采集器写入测试中用不到，提供满足接口的空实现即可。
func (f *fakeGenContentRepo) GetCaptureStats(context.Context) (*GenerationContentStats, error) {
	return &GenerationContentStats{}, nil
}

func (f *fakeGenContentRepo) GetRecent(context.Context, int) ([]GenerationContentSample, error) {
	return nil, nil
}

func enabledContentCaptureCfg() *config.Config {
	return &config.Config{Gateway: config.GatewayConfig{ContentCapture: config.ContentCaptureConfig{Enabled: true}}}
}

func TestCollectorStoresRedactedPrompt(t *testing.T) {
	repo := &fakeGenContentRepo{}
	cfg := enabledContentCaptureCfg()
	collector := NewGenerationContentCollector(repo, cfg)
	gid := int64(7)
	collector.Collect(context.Background(), GenerationContentCaptureArgs{
		RequestID:  "req-1",
		UserID:     11,
		APIKeyID:   22,
		GroupID:    &gid,
		AccountID:  33,
		Model:      "claude-opus-4-8",
		PromptBody: []byte(`{"messages":[{"role":"user","content":"hi a@b.com"}]}`),
		Result:     &ForwardResult{},
	})
	if len(repo.rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(repo.rows))
	}
	row := repo.rows[0]
	if row.RequestID != "req-1" || row.UserID == nil || *row.UserID != 11 || row.APIKeyID == nil || *row.APIKeyID != 22 || row.GroupID == nil || *row.GroupID != 7 || row.AccountID == nil || *row.AccountID != 33 {
		t.Fatalf("attribution not stored correctly: %+v", row)
	}
	if strings.Contains(row.PromptRedacted, "a@b.com") {
		t.Errorf("prompt not redacted: %s", row.PromptRedacted)
	}
	if row.RedactionVersion != generationRedactionVersion {
		t.Errorf("redaction_version = %d, want %d", row.RedactionVersion, generationRedactionVersion)
	}
	if row.PromptBytes == 0 {
		t.Errorf("expected PromptBytes > 0")
	}
}

func TestCollectorFailOpenOnRepoError(t *testing.T) {
	repo := &fakeGenContentRepo{err: errors.New("db down")}
	collector := NewGenerationContentCollector(repo, enabledContentCaptureCfg())
	// must not panic / must return normally despite repo error
	collector.Collect(context.Background(), GenerationContentCaptureArgs{RequestID: "req-2", PromptBody: []byte(`{}`)})
}

func TestCollectorNilSafe(t *testing.T) {
	var c *GenerationContentCollector
	c.Collect(context.Background(), GenerationContentCaptureArgs{RequestID: "x"}) // nil collector
	c2 := NewGenerationContentCollector(nil, enabledContentCaptureCfg())          // nil repo
	c2.Collect(context.Background(), GenerationContentCaptureArgs{RequestID: "y"})
}

func TestGatewayCollectGenerationContentGating(t *testing.T) {
	repo := &fakeGenContentRepo{}

	// disabled config → no-op even with collector wired
	disabled := &GatewayService{cfg: &config.Config{}, generationCollector: NewGenerationContentCollector(repo, &config.Config{})}
	disabled.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{RequestID: "d", PromptBody: []byte(`{}`)})
	if len(repo.rows) != 0 {
		t.Fatalf("disabled capture must not store, got %d rows", len(repo.rows))
	}

	// nil collector → no-op, no panic
	noCollector := &GatewayService{cfg: enabledContentCaptureCfg()}
	noCollector.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{RequestID: "n", PromptBody: []byte(`{}`)})

	// enabled + wired → stores
	cfg := enabledContentCaptureCfg()
	on := &GatewayService{cfg: cfg, generationCollector: NewGenerationContentCollector(repo, cfg)}
	on.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{RequestID: "e", PromptBody: []byte(`{}`)})
	if len(repo.rows) != 1 {
		t.Fatalf("enabled capture must store 1 row, got %d", len(repo.rows))
	}
}
