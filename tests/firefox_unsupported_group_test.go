package tests

import (
	"crypto/tls"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/stretchr/testify/assert"
)

// TestFirefoxHelloRetryRequest checks that Firefox_110 completes a handshake with a server that only accepts P-384,
// a group Firefox_110 supports but sends no key share for, so the server has to ask for it with a HelloRetryRequest.
func TestFirefoxHelloRetryRequest(t *testing.T) {
	server := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {}))
	server.TLS = &tls.Config{
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{tls.CurveP384},
	}
	server.StartTLS()
	defer server.Close()

	client, err := tls_client.NewHttpClient(nil,
		tls_client.WithClientProfile(profiles.Firefox_110),
		tls_client.WithInsecureSkipVerify(),
	)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if !assert.NoError(t, err) {
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
