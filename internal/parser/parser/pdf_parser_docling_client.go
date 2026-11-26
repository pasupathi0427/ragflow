package parser

import (
	"net/http"
	"net/url"
	"strings"

	"ragflow/internal/common"
)

// doclingHTTPClient picks the HTTP client for a Docling Serve request.
//
// DOCLING_SERVER_URL is set by whoever runs the server, so a private address
// there (Docling on another machine in the same network) is trusted: requests
// to that scheme and host use the scheme-safe client, as self-hosted model
// drivers do. Any other URL, including one taken from a dataset's parser
// settings, keeps the strict SSRF client, so a dataset cannot point the parser
// at another internal address.
func doclingHTTPClient(serverURL string) *http.Client {
	configured := strings.TrimSpace(common.GetEnv(common.EnvDoclingServerURL))
	if configured != "" && sameDoclingOrigin(serverURL, configured) {
		return common.GetSchemeSafeHTTPClient()
	}
	return common.GetSSRFHTTPClient()
}

// sameDoclingOrigin reports whether two URLs share scheme, host and port.
func sameDoclingOrigin(a, b string) bool {
	ua, errA := url.Parse(strings.TrimSpace(a))
	ub, errB := url.Parse(strings.TrimSpace(b))
	if errA != nil || errB != nil || ua.Host == "" || ub.Host == "" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) &&
		strings.EqualFold(ua.Hostname(), ub.Hostname()) &&
		ua.Port() == ub.Port()
}
