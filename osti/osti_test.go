package osti

import (
	"io"
	"log"
	"net"
	"net/http"
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
		t.Fatalf("http.Get error: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}
}
