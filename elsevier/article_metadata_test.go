package elsevier

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sandialabs/bibcheck/internal/testutil"
)

func TestProxyReachable(t *testing.T) {

	// Create a dummy request to extract proxy settings
	// The URL matters because proxy selection can be URL-dependent
	targetURL := "https://api.elsevier.com"
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Extract proxy URL from environment (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
	proxyURL, err := http.ProxyFromEnvironment(req)
	if err != nil {
		t.Fatalf("failed to get proxy from environment: %v", err)
	}

	if proxyURL == nil {
		t.Skip("no proxy configured in environment (set HTTP_PROXY or HTTPS_PROXY)")
	}

	// Attempt to dial the proxy
	proxyHost := proxyURL.Host
	if proxyURL.Port() == "" {
		// Add default port based on scheme
		switch proxyURL.Scheme {
		case "http":
			proxyHost = net.JoinHostPort(proxyURL.Hostname(), "80")
		case "https":
			proxyHost = net.JoinHostPort(proxyURL.Hostname(), "443")
		case "socks5":
			proxyHost = net.JoinHostPort(proxyURL.Hostname(), "1080")
		}
	}

	testutil.SkipIfTCPUnavailable(t, proxyHost)
	fmt.Println("Successfully connected to proxy")
}

func TestElsevierReachable(t *testing.T) {
	testutil.SkipIfTCPUnavailable(t, "api.elsevier.com:443")

	resp, err := http.Get("https://api.elsevier.com")
	if err != nil {
		t.Fatalf("http.Get error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}
}

func TestArticleMetadata(t *testing.T) {

	apiKey, ok := os.LookupEnv("ELSEVIER_API_KEY")

	if !ok {
		t.Skipf("ELSEVIER_API_KEY not provided")
	}
	testutil.SkipIfTCPUnavailable(t, "api.elsevier.com:443")

	client := NewClient(apiKey, WithTimeout(10*time.Second))

	_, err := client.ArticleMetadata(&Query{
		Authors: []string{"IDO, NOTEXIST"},
	}, nil)
	if err != nil {
		t.Errorf("elsevier client error: %v", err)
	}
}
