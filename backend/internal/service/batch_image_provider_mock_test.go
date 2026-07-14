package service

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestMockBatchImageProviderDeterministicPNGJSONL(t *testing.T) {
	p := NewMockBatchImageProvider()
	job := &BatchImageJob{BatchID: "imgbatch_mock_1", Provider: BatchImageProviderMock}
	got, err := p.Submit(context.Background(), job, localMockBatchImageAccount(), BatchImageInput{
		BatchID: job.BatchID,
		Items: []BatchImageInputItem{
			{CustomID: "cover_001", Prompt: "a"},
			{CustomID: "cover_002", Prompt: "b"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ProviderJobName == "" {
		t.Fatal("missing provider job name")
	}
	if p.CreateCount() != 1 {
		t.Fatalf("create count=%d, want 1", p.CreateCount())
	}
	rc, contentType, err := p.OpenResult(context.Background(), job, localMockBatchImageAccount())
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contentType, "json") {
		t.Fatalf("content type=%q", contentType)
	}
	body := string(raw)
	if !strings.Contains(body, "cover_001") || !strings.Contains(body, "image/png") {
		t.Fatalf("unexpected jsonl: %s", body)
	}
	// Second submit on same provider increments create count (one per task).
	job2 := &BatchImageJob{BatchID: "imgbatch_mock_2", Provider: BatchImageProviderMock}
	if _, err := p.Submit(context.Background(), job2, localMockBatchImageAccount(), BatchImageInput{BatchID: job2.BatchID, Items: []BatchImageInputItem{{CustomID: "x", Prompt: "y"}}}); err != nil {
		t.Fatal(err)
	}
	if p.CreateCount() != 2 {
		t.Fatalf("create count=%d, want 2", p.CreateCount())
	}
}
