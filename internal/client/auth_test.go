package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/chat"
)

const authSentinel = "phase2-test-sentinel-aa11"

func testAuthClient(url, token string) *Client {
	cfg := testCfg(url)
	cfg.BearerTokenEnv = "RADICHAT_PHASE2_TEST_TOKEN"
	cfg.BearerToken = token
	return New(cfg, Options{
		DialTimeout:           2 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		IdleReadTimeout:       200 * time.Millisecond,
		MaxEventBytes:         64 << 10,
		MaxErrorBodyBytes:     1024,
		MaxStreamBytes:        1 << 20,
	})
}

func TestStreamSendsBearerOnlyWhenConfigured(t *testing.T) {
	var gotAuth string
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("hi", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := testAuthClient(srv.URL+"/v1/chat/completions", authSentinel)
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "hello"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hi" {
		t.Fatalf("text %q", text)
	}
	if gotAuth != "Bearer "+authSentinel {
		t.Fatal("Authorization header missing or not the expected Bearer value")
	}
	if gotCookie != "" {
		t.Fatal("Cookie header present")
	}
}

func TestStreamUnauthenticatedOmitsAuthorization(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("hi", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/v1/chat/completions")
	if _, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "hello"}}, 8192, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatal("zero-auth sent an Authorization header")
	}
}

func TestProtectedAndUnprotectedStreamsMatch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("hello", ""))
		io.WriteString(w, sseChunk(" world", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	open := httptest.NewServer(handler)
	defer open.Close()
	protected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+authSentinel {
			http.Error(w, "missing auth "+authSentinel, http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer protected.Close()

	openText, err := testClient(open.URL+"/").Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	authText, err := testAuthClient(protected.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if openText != "hello world" || authText != openText {
		t.Fatalf("open %q auth %q", openText, authText)
	}
}

func TestAuthRedirectLocationOmitsEncodedSentinel(t *testing.T) {
	encoded := "t=" + strings.ReplaceAll(authSentinel, "-", "%2D")
	loc := "/leak?" + encoded + "&frag=" + authSentinel[:8] + authSentinel[8:]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected redirect refusal")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) || strings.Contains(msg, encoded) || strings.Contains(msg, "leak") {
		t.Fatal("redirect diagnostic disclosed Location or sentinel")
	}
	if !strings.Contains(msg, "redirect") {
		t.Fatal("expected redirect classification")
	}
	if strings.Contains(msg, "http://") || strings.Contains(msg, "https://") {
		t.Fatal("authenticated redirect diagnostic included a URL")
	}
}

func TestAuthTransportErrorIsGeneric(t *testing.T) {
	cfg := testCfg("http://127.0.0.1:1/v1/chat/completions")
	cfg.BearerTokenEnv = "RADICHAT_PHASE2_TEST_TOKEN"
	cfg.BearerToken = authSentinel
	c := New(cfg, Options{
		DialTimeout:           time.Second,
		TLSHandshakeTimeout:   time.Second,
		ResponseHeaderTimeout: time.Second,
		IdleReadTimeout:       time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("dial blocked %s raw=%s", authSentinel, "leak")
		},
	})
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) || strings.Contains(msg, "dial blocked") || strings.Contains(msg, "raw=leak") {
		t.Fatal("transport error included underlying text")
	}
	if !strings.Contains(msg, "cannot contact") {
		t.Fatal("expected classified contact failure")
	}
}

func TestUnauthenticatedTransportErrorByteForByte(t *testing.T) {
	dialErr := fmt.Errorf("dial blocked by test")
	cfg := testCfg("http://127.0.0.1:1/v1/chat/completions")
	c := New(cfg, Options{
		DialTimeout:           time.Second,
		TLSHandshakeTimeout:   time.Second,
		ResponseHeaderTimeout: time.Second,
		IdleReadTimeout:       time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, dialErr
		},
	})
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	wrapped := &url.Error{Op: "Post", URL: cfg.Endpoint, Err: dialErr}
	want := fmt.Sprintf("cannot contact %s: %v; check that the server is running and the endpoint URL is correct", cfg.Endpoint, wrapped)
	if err.Error() != want {
		t.Fatal("unauthenticated transport diagnostic changed")
	}
}

func TestAuthStreamEchoesTokenSplitDeltas(t *testing.T) {
	mid := len(authSentinel) / 2
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("pre", ""))
		io.WriteString(w, sseChunk(authSentinel[:mid], ""))
		io.WriteString(w, sseChunk(authSentinel[mid:], ""))
		io.WriteString(w, sseChunk("post", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var got strings.Builder
	text, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, func(s string) error {
		got.WriteString(s)
		return nil
	})
	if err != nil {
		t.Fatal("stream failed")
	}
	if strings.Contains(got.String(), authSentinel) || strings.Contains(text, authSentinel) {
		t.Fatal("streamed output disclosed sentinel")
	}
	if !strings.Contains(got.String(), "pre") || !strings.Contains(got.String(), "post") {
		t.Fatal("streamed output lost surrounding text")
	}
	if !strings.Contains(text, "pre") || !strings.Contains(text, "post") {
		t.Fatal("assembled text lost surrounding content")
	}
}

func TestAuthStreamTrailingTokenPrefixOmitsFromOutput(t *testing.T) {
	prefix := authSentinel[:len(authSentinel)-1]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("pre", ""))
		io.WriteString(w, sseChunk(prefix, "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var got strings.Builder
	text, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, func(s string) error {
		got.WriteString(s)
		return nil
	})
	if err != nil {
		t.Fatal("stream failed")
	}
	if strings.Contains(got.String(), prefix) || strings.Contains(text, prefix) {
		t.Fatal("successful stream disclosed trailing token prefix")
	}
	if !strings.Contains(got.String(), "pre") {
		t.Fatal("streamed output lost surrounding text")
	}
	if !strings.Contains(text, "pre") {
		t.Fatal("assembled text lost surrounding content")
	}
}

func TestAuthStreamKeepsNonSecretTrailingOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("hello", ""))
		io.WriteString(w, sseChunk("tail-ok", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var got strings.Builder
	text, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, func(s string) error {
		got.WriteString(s)
		return nil
	})
	if err != nil {
		t.Fatal("stream failed")
	}
	if got.String() != "hellotail-ok" {
		t.Fatal("authenticated stream dropped non-secret trailing output")
	}
	if text != "hellotail-ok" {
		t.Fatal("assembled text dropped non-secret trailing output")
	}
}

func TestUnauthenticatedStreamPrintsTrailingPrefix(t *testing.T) {
	prefix := authSentinel[:len(authSentinel)-1]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("pre", ""))
		io.WriteString(w, sseChunk(prefix, "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	text, err := testClient(srv.URL+"/").Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal("stream failed")
	}
	if text != "pre"+prefix {
		t.Fatal("zero-auth stream filtering changed trailing assistant text")
	}
}

func TestUnauthenticatedStreamPrintsLiteralSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk(authSentinel, "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	text, err := testClient(srv.URL+"/").Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal("stream failed")
	}
	if text != authSentinel {
		t.Fatal("zero-auth stream filtering changed assistant text")
	}
}

func TestAuthHTTPFailureEchoesOmitToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "unauthorized "+r.Header.Get("Authorization")+" token="+authSentinel+"\n")
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected HTTP failure")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) {
		t.Fatal("HTTP error disclosed sentinel")
	}
	if strings.Contains(msg, "Bearer") || strings.Contains(msg, "server said") {
		t.Fatal("HTTP error included untrusted response snippet")
	}
	if !strings.Contains(msg, "401") {
		t.Fatal("expected HTTP 401 in diagnostic")
	}
	if !strings.Contains(msg, "omitted") {
		t.Fatal("expected omitted-body classification")
	}
	if !strings.Contains(msg, "fix the endpoint or model configuration") {
		t.Fatal("expected corrective hint")
	}
}

func TestAuthWrongContentTypeEchoesOmitToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"error":"`+r.Header.Get("Authorization")+`","token":"`+authSentinel+`"}`)
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected content-type failure")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) {
		t.Fatal("content-type error disclosed sentinel")
	}
	if strings.Contains(msg, "Bearer") || strings.Contains(msg, "server said") {
		t.Fatal("content-type error included untrusted response snippet")
	}
	if !strings.Contains(msg, "text/event-stream") {
		t.Fatal("expected content-type diagnostic")
	}
	if !strings.Contains(msg, "omitted") {
		t.Fatal("expected omitted-body classification")
	}
}

func TestAuthHTTPFailureOmitsReencodedToken(t *testing.T) {
	frag := authSentinel[:8] + " " + authSentinel[8:]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "split="+frag+"\n")
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected HTTP failure")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) || strings.Contains(msg, authSentinel[:8]) {
		t.Fatal("HTTP error disclosed fragmented sentinel")
	}
	if !strings.Contains(msg, "403") || !strings.Contains(msg, "omitted") {
		t.Fatal("expected useful status classification")
	}
}

func TestAuthTransportErrorOmitsToken(t *testing.T) {
	cfg := testCfg("http://127.0.0.1:1/v1/chat/completions")
	cfg.BearerTokenEnv = "RADICHAT_PHASE2_TEST_TOKEN"
	cfg.BearerToken = authSentinel
	c := New(cfg, Options{
		DialTimeout:           time.Second,
		TLSHandshakeTimeout:   time.Second,
		ResponseHeaderTimeout: time.Second,
		IdleReadTimeout:       time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("dial blocked %s", authSentinel)
		},
	})
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	msg := err.Error()
	if strings.Contains(msg, authSentinel) || strings.Contains(msg, "dial blocked") {
		t.Fatal("transport error disclosed sentinel or underlying error")
	}
}

func TestUnauthenticatedErrorSnippetUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	msgs := []chat.Message{{Role: "user", Content: "x"}}
	_, openErr := testClient(srv.URL+"/").Stream(context.Background(), msgs, 8192, nil)
	_, authErr := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), msgs, 8192, nil)
	if openErr == nil || authErr == nil {
		t.Fatal("expected errors")
	}
	want := `endpoint returned HTTP 502; server said "nope"; fix the endpoint or model configuration`
	if openErr.Error() != want {
		t.Fatal("unauthenticated diagnostic changed from Phase 1")
	}
	if authErr.Error() == openErr.Error() {
		t.Fatal("authenticated mode still printed the untrusted body snippet")
	}
	if strings.Contains(authErr.Error(), "nope") || strings.Contains(authErr.Error(), "server said") {
		t.Fatal("authenticated diagnostic included server body")
	}
	if !strings.Contains(authErr.Error(), "502") || !strings.Contains(authErr.Error(), "omitted") {
		t.Fatal("authenticated diagnostic lost status or hint")
	}
	if strings.Contains(authErr.Error(), authSentinel) {
		t.Fatal("authenticated diagnostic disclosed sentinel")
	}
}

func TestAuthStreamErrorRedactsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"error":{"message":"bad `+authSentinel+`"}}`+"\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "error object") {
		t.Fatal("expected stream error object")
	}
	if strings.Contains(err.Error(), authSentinel) {
		t.Fatal("stream error disclosed sentinel")
	}
	if strings.Contains(err.Error(), "server said") {
		t.Fatal("stream error included untrusted payload snippet")
	}
	if !strings.Contains(err.Error(), "omitted") {
		t.Fatal("expected omitted-body classification")
	}
}

func TestTLSSettingsUnchanged(t *testing.T) {
	c := testClient("https://127.0.0.1:1/")
	tr, ok := c.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport %T", c.http.Transport)
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("missing TLS config")
	}
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify enabled")
	}
	if tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion %d", tr.TLSClientConfig.MinVersion)
	}
	if tr.TLSClientConfig.RootCAs != nil {
		t.Fatal("custom CA pool set")
	}
}

func TestHTTPSUnknownCARejected(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request reached TLS server with unknown CA")
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected TLS verification failure")
	}
	if strings.Contains(err.Error(), authSentinel) {
		t.Fatal("TLS error disclosed sentinel")
	}
}

func TestAuthNoRetryOnFailure(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		http.Error(w, "fail", 500)
	}))
	defer srv.Close()
	_, err := testAuthClient(srv.URL+"/", authSentinel).Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("attempts %d", n.Load())
	}
	if strings.Contains(err.Error(), authSentinel) {
		t.Fatal("error disclosed sentinel")
	}
}
