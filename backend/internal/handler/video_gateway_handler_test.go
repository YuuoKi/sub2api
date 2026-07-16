package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVideoGatewayHandlerRequiresEmployeeAPIKeyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		method, path string
		handler      gin.HandlerFunc
	}{
		{http.MethodGet, "/v1/video/providers", (&VideoGatewayHandler{}).Providers},
		{http.MethodPost, "/v1/video/tasks", (&VideoGatewayHandler{}).Create},
		{http.MethodGet, "/v1/video/tasks/1", (&VideoGatewayHandler{}).Get},
	} {
		r := gin.New()
		r.Handle(tc.method, tc.path, tc.handler)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, http.StatusForbidden, w.Code, tc.path)
	}
}
