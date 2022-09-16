package proxy_test

import (
	"os"
	"testing"
)

func verifySocketPermissions(t *testing.T, addr string) {
	fi, err := os.Stat(addr)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", addr, err)
	}
	if fm := fi.Mode(); fm != 0777|os.ModeSocket {
		t.Fatalf("file mode: want = %v, got = %v", 0777|os.ModeSocket, fm)
	}
}
