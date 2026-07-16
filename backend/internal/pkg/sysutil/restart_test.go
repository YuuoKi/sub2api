package sysutil

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestartServiceRejectsUnsupportedRuntime(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("restart is supported on Linux and would terminate the unit-test process")
	}

	require.Error(t, RestartService())
}
