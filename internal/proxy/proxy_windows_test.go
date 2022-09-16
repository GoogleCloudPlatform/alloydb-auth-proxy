package proxy_test

import (
	"testing"
)

func verifySocketPermissions(t *testing.T, addr string) {
	// On Linux and Darwin, we check that the socket named by addr exists with
	// os.Stat. That operation is not supported on Windows.
	// See https://github.com/microsoft/Windows-Containers/issues/97#issuecomment-887713195
}
