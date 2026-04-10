package version

import (
	"os"
	"strings"
	"testing"
)

func TestVersionMatchesTag(t *testing.T) {
	tag := os.Getenv("EXPECTED_VERSION_TAG")
	if tag == "" {
		t.Skip("EXPECTED_VERSION_TAG is not set")
	}

	expectedVersion := strings.TrimPrefix(tag, "v")
	if expectedVersion != String() {
		t.Fatalf("version.String() = %q, want %q from tag %q", String(), expectedVersion, tag)
	}
}
