package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestCreateTaskRejectsNonAdminProviderAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &VideoHandler{video: nil}
	r := gin.New()
	r.POST("/video/tasks", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 9})
		c.Set(string(middleware.ContextKeyUserRole), "user")
		h.CreateTask(c)
	})
	body := `{"provider_account_id":99,"task_type":"text_to_video","prompt":"hello","execution_mode":"mock"}`
	req := httptest.NewRequest(http.MethodPost, "/video/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusCreated {
		t.Fatalf("expected non-admin bare provider_account_id rejection, got %d", rec.Code)
	}
	payload := rec.Body.String()
	if !bytes.Contains(rec.Body.Bytes(), []byte("PROVIDER_ACCOUNT_ID_FORBIDDEN")) &&
		!bytes.Contains(rec.Body.Bytes(), []byte("不能指定")) {
		t.Fatalf("unexpected body: %s", payload)
	}
}

func TestVideoCreateRequestSerializesExecutionMode(t *testing.T) {
	reqBody := videoTaskCreateRequest{ExecutionMode: service.ExecutionModeMock, TaskType: "text_to_video", Prompt: "x"}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"execution_mode":"mock"`)) {
		t.Fatalf("execution_mode must serialize on create payload: %s", raw)
	}
}
