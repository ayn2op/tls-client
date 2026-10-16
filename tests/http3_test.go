package tests

import (
	"io"
	"net"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/httptest"
	"github.com/bogdanfinn/quic-go-utls/http3"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"
)

// TestHTTP3 checks that protocol racing uses HTTP/3 when the server offers both HTTP/2 and HTTP/3 and HTTP/3 is quick,
// as it always is on loopback, where QUIC finishes well within the head start HTTP/3 gets.
func TestHTTP3(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	})

	h2Server := httptest.NewUnstartedServer(handler)
	h2Server.EnableHTTP2 = true
	h2Server.StartTLS()
	defer h2Server.Close()

	// HTTP/3 listens on the UDP port with the number of the HTTP/2 TCP port.
	udpConn, err := net.ListenPacket("udp", h2Server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	h3Server := &http3.Server{
		Handler:   handler,
		TLSConfig: http3.ConfigureTLSConfig(&tls.Config{Certificates: h2Server.TLS.Certificates}),
	}
	go func() { _ = h3Server.Serve(udpConn) }()
	defer h3Server.Close()

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_144),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithProtocolRacing(),
		tls_client.WithInsecureSkipVerify(),
	}

	client, err := tls_client.NewHttpClient(nil, options...)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, h2Server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header = defaultHeader

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if proto := string(body); proto != "HTTP/3.0" {
		t.Fatalf("server saw %q, want HTTP/3.0", proto)
	}
}

func TestDisableHTTP3(t *testing.T) {
	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_144),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithDisableHttp3(),
		tls_client.WithDebug(),
	}

	client, err := tls_client.NewHttpClient(nil, options...)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://http3.is/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header = defaultHeader

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(body), "HTTP/3 (h3-29 or h3-27) was not used to request this page") {
		t.Fatal("Response did contain HTTP3 result")
	}
}
