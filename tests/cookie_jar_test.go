package tests

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/bogdanfinn/tls-client/profiles"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/stretchr/testify/assert"
)

func TestClient_SkipExistingCookiesOnClientSetCookies(t *testing.T) {
	jarOptions := []tls_client.CookieJarOption{
		tls_client.WithSkipExisting(),
	}

	jar := tls_client.NewCookieJar(jarOptions...)

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}
	u := &url.URL{
		Scheme: "http",
		Host:   "testhost.de",
		Path:   "/test",
	}

	assert.Equal(t, 0, len(client.GetCookies(u)))

	cookie := &http.Cookie{
		Name:   "test1",
		Value:  "test1",
		MaxAge: 1,
	}

	client.SetCookies(u, []*http.Cookie{cookie})

	assert.Equal(t, 1, len(client.GetCookies(u)))

	cookie2 := &http.Cookie{
		Name:   "test2",
		Value:  "test2",
		MaxAge: 1,
	}
	client.SetCookies(u, []*http.Cookie{cookie2})

	assert.Equal(t, 2, len(client.GetCookies(u)))

	cookie3 := &http.Cookie{
		Name:   "test1",
		Value:  "test1",
		MaxAge: 1,
	}
	client.SetCookies(u, []*http.Cookie{cookie3})

	assert.Equal(t, 2, len(client.GetCookies(u)))
}

func TestClient_SkipExistingCookiesOnSetCookiesResponse(t *testing.T) {
	// The server sets the same cookies on every response, with the number of the response as their value.
	cookieNames := []string{"a", "b", "c"}
	responses := 0
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		responses++
		for _, name := range cookieNames {
			stdhttp.SetCookie(w, &stdhttp.Cookie{Name: name, Value: strconv.Itoa(responses), Path: "/", MaxAge: 60})
		}
	}))
	defer server.Close()

	jarOptions := []tls_client.CookieJarOption{
		tls_client.WithSkipExisting(),
	}

	jar := tls_client.NewCookieJar(jarOptions...)

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		t.Fatal(err)
	}

	get := func() {
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if _, err := io.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
	}

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	get()
	cookiesAfterFirstRequest := client.GetCookies(u)
	assert.Equal(t, len(cookieNames), len(cookiesAfterFirstRequest))

	// The jar keeps the cookies from the first response, whether they are set again by the client or by the server.
	assertFirstValues := func() {
		t.Helper()
		cookies := client.GetCookies(u)
		assert.Equal(t, len(cookieNames), len(cookies))
		for _, cookie := range cookies {
			assert.Equal(t, "1", cookie.Value, cookie.Name)
		}
	}
	assertFirstValues()

	cookie3 := &http.Cookie{
		Name:   cookiesAfterFirstRequest[0].Name,
		Value:  "changed",
		Domain: cookiesAfterFirstRequest[0].Domain,
		MaxAge: cookiesAfterFirstRequest[0].MaxAge,
	}
	client.SetCookies(u, []*http.Cookie{cookie3})
	assertFirstValues()

	get()
	assertFirstValues()
}

func TestClient_ExcludeExpiredCookiesFromRequest(t *testing.T) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}
	u := &url.URL{
		Scheme: "http",
		Host:   "testhost.de",
		Path:   "/test",
	}

	assert.Equal(t, 0, len(client.GetCookies(u)))

	cookieAlive := &http.Cookie{
		Name:   "test1",
		Value:  "test1",
		MaxAge: 1,
	}

	cookieExpired := &http.Cookie{
		Name:   "test2",
		Value:  "test2",
		MaxAge: -1,
	}

	client.SetCookies(u, []*http.Cookie{cookieAlive, cookieExpired})

	assert.Equal(t, 1, len(client.GetCookies(u)))

	cookieExpireExisting := &http.Cookie{
		Name:   "test1",
		Value:  "test1",
		MaxAge: -1,
	}
	client.SetCookies(u, []*http.Cookie{cookieExpireExisting})

	assert.Equal(t, 0, len(client.GetCookies(u)))
}
