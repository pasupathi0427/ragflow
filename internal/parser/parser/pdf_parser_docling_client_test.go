package parser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"ragflow/internal/common"
)

// doclingUnavailableServer answers every convert request with 503 and counts
// the requests that reached it.
func doclingUnavailableServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func TestParsePDFWithDocling_TrustsEnvConfiguredPrivateHost(t *testing.T) {
	t.Setenv(common.EnvAllowAnyHost, "")
	server, hits := doclingUnavailableServer(t)
	t.Setenv(common.EnvDoclingServerURL, server.URL)

	res := parsePDFWithDocling(t.Context(), "sample.pdf", []byte("%PDF-1.4"), NewPDFParser())
	if res.Err == nil {
		t.Fatal("error = nil, want the 503 responses reported")
	}
	if strings.Contains(res.Err.Error(), "non-public address") {
		t.Fatalf("error = %v, want the env-configured loopback host allowed", res.Err)
	}
	if hits.Load() == 0 {
		t.Fatal("no request reached the env-configured Docling server")
	}
}

func TestParsePDFWithDocling_ParserURLPrivateHostStaysBlocked(t *testing.T) {
	t.Setenv(common.EnvAllowAnyHost, "")
	server, hits := doclingUnavailableServer(t)
	t.Setenv(common.EnvDoclingServerURL, "http://10.9.9.9:5001")

	p := NewPDFParser()
	p.DoclingServerURL = server.URL
	res := parsePDFWithDocling(t.Context(), "sample.pdf", []byte("%PDF-1.4"), p)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "non-public address") {
		t.Fatalf("error = %v, want the non-public address rejection", res.Err)
	}
	if hits.Load() != 0 {
		t.Fatalf("server requests = %d, want 0 for a private host not in DOCLING_SERVER_URL", hits.Load())
	}
}

func TestParsePDFWithDocling_ParserURLWithoutEnvStaysBlocked(t *testing.T) {
	t.Setenv(common.EnvAllowAnyHost, "")
	t.Setenv(common.EnvDoclingServerURL, "")
	server, hits := doclingUnavailableServer(t)

	p := NewPDFParser()
	p.DoclingServerURL = server.URL
	res := parsePDFWithDocling(t.Context(), "sample.pdf", []byte("%PDF-1.4"), p)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "non-public address") {
		t.Fatalf("error = %v, want the non-public address rejection", res.Err)
	}
	if hits.Load() != 0 {
		t.Fatalf("server requests = %d, want 0", hits.Load())
	}
}

func TestSameDoclingOrigin(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"http://10.0.147.88:5001", "http://10.0.147.88:5001/", true},
		{"HTTP://Docling.Local:5001", "http://docling.local:5001", true},
		{"http://10.0.147.88:5001", "http://10.0.147.88:5002", false},
		{"http://10.0.147.88:5001", "https://10.0.147.88:5001", false},
		{"http://10.0.147.88:5001", "http://10.0.147.89:5001", false},
		{"http://10.0.147.88:5001", "", false},
		{"not a url", "http://10.0.147.88:5001", false},
	}
	for _, tc := range cases {
		if got := sameDoclingOrigin(tc.a, tc.b); got != tc.want {
			t.Errorf("sameDoclingOrigin(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
