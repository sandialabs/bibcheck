package osti

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOstiDial(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "www.osti.gov:443", 5*time.Second)
	if err != nil {
		log.Fatal("DialTimeout error: ", err)
	}
	conn.Close()
}

func TestOstiGet(t *testing.T) {

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
