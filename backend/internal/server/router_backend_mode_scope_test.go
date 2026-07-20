//go:build unit

package server

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackendModeProductGuardScope(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	source, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "router.go"))
	require.NoError(t, err)
	routerSource := string(source)

	require.Contains(t, routerSource, `v1 := r.Group("/api/v1")`)
	require.Contains(t, routerSource, `v1.Use(middleware2.BackendModeProductSurfaceGuard(settingService))`)
	require.Contains(t, routerSource, `routes.RegisterGatewayRoutes(r,`)
	require.NotContains(t, routerSource, `routes.RegisterGatewayRoutes(v1,`)
}
