package testutil

import (
	"net"
	"testing"
	"time"
)

// SkipIfTCPUnavailable skips the current test when address cannot be reached.
// It is intended for tests that intentionally exercise external services so
// they can be run in restricted no-network environments without failing the
// whole test suite.
func SkipIfTCPUnavailable(t *testing.T, address string) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Skipf("external network unavailable or %s unreachable: %v", address, err)
	}
	_ = conn.Close()
}
