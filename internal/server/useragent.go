package server

import (
	"net/url"
	"strings"

	"github.com/sandialabs/bibcheck/version"
)

// These upstreams don't like it when we forward the browser user agent
var userAgentExcludedHosts = map[string]struct{}{
	"www.intel.com": {},
}

func defaultUserAgent() string {
	return "bibcheck / " + version.String() + " github.com/sandialabs/bibcheck"
}

func upstreamUserAgent(target *url.URL, browserUserAgent string) string {
	if _, excluded := userAgentExcludedHosts[strings.ToLower(target.Hostname())]; excluded || browserUserAgent == "" {
		return defaultUserAgent()
	}
	return browserUserAgent
}
