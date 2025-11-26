package osti

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestOstiDial(t *testing.T) {

	conn, err := net.DialTimeout("tcp", "osti.gov:443", 5*time.Second)
	if err != nil {
		log.Fatal("DialTimeout error: ", err)
	}
	conn.Close()
}

func TestOstiGet(t *testing.T) {

	resp, err := http.Get("https://www.osti.gov")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
}
