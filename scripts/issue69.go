// Command repro_issue69 makes a curl-like request using only Go's standard
// library and prints the request and response headers.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

const defaultURL = "https://www.intel.com/content/www/us/en/docs/mpi-library/developer-guide-linux/2021-16/notified-one-sided-communications.html"

func main() {
	target := defaultURL
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	userAgent := "curl/8.14.1"
	if len(os.Args) > 2 {
		userAgent = os.Args[2]
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", userAgent)

	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("> %s\n", dump)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("< %s %s\n", resp.Proto, resp.Status)
	if err := resp.Header.Write(os.Stdout); err != nil {
		log.Fatal(err)
	}

	const previewBytes = 512
	preview, err := io.ReadAll(io.LimitReader(resp.Body, previewBytes))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%s\n", preview)
}
