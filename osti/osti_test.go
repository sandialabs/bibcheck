package osti

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sandialabs/bibcheck/internal/testutil"
)

func TestOstiDial(t *testing.T) {
	testutil.SkipIfTCPUnavailable(t, "www.osti.gov:443")
}

func TestOstiGet(t *testing.T) {
	testutil.SkipIfTCPUnavailable(t, "www.osti.gov:443")

	resp, err := http.Get("https://www.osti.gov")
	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "INTERNAL_ERROR") {
			t.Skipf("OSTI returned a transient internal error: %v", err)
		}
		t.Fatalf("http.Get error: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}
}
