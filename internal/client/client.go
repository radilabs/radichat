package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/radilabs/radichat/internal/chat"
	"github.com/radilabs/radichat/internal/config"
)

const (
	defaultDialTimeout     = 10 * time.Second
	defaultTLSHandshake    = 10 * time.Second
	defaultHeaderTimeout   = 15 * time.Second
	defaultIdleReadTimeout = 60 * time.Second
	defaultMaxEventBytes   = 1 << 20
	defaultMaxErrorBody    = 8 << 10
	defaultMaxStreamBytes  = 8 << 20
)

// Options controls transport and stream bounds. Zero values receive defaults.
type Options struct {
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleReadTimeout       time.Duration
	MaxEventBytes         int
	MaxErrorBodyBytes     int
	MaxStreamBytes        int64
	DialContext           func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (o Options) withDefaults() Options {
	if o.DialTimeout == 0 {
		o.DialTimeout = defaultDialTimeout
	}
	if o.TLSHandshakeTimeout == 0 {
		o.TLSHandshakeTimeout = defaultTLSHandshake
	}
	if o.ResponseHeaderTimeout == 0 {
		o.ResponseHeaderTimeout = defaultHeaderTimeout
	}
	if o.IdleReadTimeout == 0 {
		o.IdleReadTimeout = defaultIdleReadTimeout
	}
	if o.MaxEventBytes == 0 {
		o.MaxEventBytes = defaultMaxEventBytes
	}
	if o.MaxErrorBodyBytes == 0 {
		o.MaxErrorBodyBytes = defaultMaxErrorBody
	}
	if o.MaxStreamBytes == 0 {
		o.MaxStreamBytes = defaultMaxStreamBytes
	}
	return o
}

type Client struct {
	cfg  config.Config
	opt  Options
	http *http.Client
}

func New(cfg config.Config, opt Options) *Client {
	opt = opt.withDefaults()
	dial := opt.DialContext
	if dial == nil {
		d := &net.Dialer{Timeout: opt.DialTimeout, KeepAlive: 0}
		dial = d.DialContext
	}
	transport := &http.Transport{
		Proxy:                  nil,
		DialContext:            dial,
		TLSHandshakeTimeout:    opt.TLSHandshakeTimeout,
		ResponseHeaderTimeout:  opt.ResponseHeaderTimeout,
		DisableKeepAlives:      true,
		DisableCompression:     true,
		ForceAttemptHTTP2:      false,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		MaxResponseHeaderBytes: 1 << 20,
	}
	return &Client{
		cfg: cfg,
		opt: opt,
		http: &http.Client{
			Transport: transport,
			Timeout:   0,
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				return fmt.Errorf("redirect to %s refused; set endpoint to the final chat-completions URL", req.URL.Redacted())
			},
		},
	}
}

func withIdleCancel(parent context.Context, idle time.Duration) (ctx context.Context, reset, stop func()) {
	ctx, cancel := context.WithCancel(parent)
	timer := time.AfterFunc(idle, cancel)
	reset = func() {
		if timer.Stop() {
			timer.Reset(idle)
		}
	}
	stop = func() {
		timer.Stop()
		cancel()
	}
	return ctx, reset, stop
}

type wireMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type requestBody struct {
	Model     string        `json:"model"`
	Messages  []wireMessage `json:"messages"`
	Stream    bool          `json:"stream"`
	MaxTokens int64         `json:"max_tokens"`
}

func (c *Client) Stream(ctx context.Context, messages []chat.Message, outputLimit int64, onDelta func(string) error) (string, error) {
	body := requestBody{
		Model:     c.cfg.Model,
		Stream:    true,
		MaxTokens: c.cfg.GenerationReserve,
		Messages:  make([]wireMessage, len(messages)),
	}
	for i, m := range messages {
		body.Messages[i] = wireMessage{Role: m.Role, Content: m.Content}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("cannot encode chat request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("cannot build request to %s: %v", c.cfg.Endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	var gotConn net.Conn
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			gotConn = info.Conn
		},
	}))

	resp, err := c.http.Do(req)
	if err != nil {
		return "", c.transportError(ctx, err)
	}
	defer resp.Body.Close()
	closeReq := func() {
		_ = resp.Body.Close()
		if gotConn != nil {
			_ = gotConn.Close()
		}
	}
	streamCtx, resetIdle, stopIdle := withIdleCancel(ctx, c.opt.IdleReadTimeout)
	defer stopIdle()
	stopWatch := context.AfterFunc(streamCtx, closeReq)
	defer stopWatch()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet := readBounded(resp.Body, c.opt.MaxErrorBodyBytes)
		return "", fmt.Errorf("endpoint returned HTTP %d; %s", resp.StatusCode, snippetHint(snippet))
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "text/event-stream") {
		snippet := readBounded(resp.Body, c.opt.MaxErrorBodyBytes)
		return "", fmt.Errorf("endpoint Content-Type %q is not text/event-stream; %s", resp.Header.Get("Content-Type"), snippetHint(snippet))
	}

	limited := idleResetReader{
		r:     io.LimitReader(resp.Body, c.opt.MaxStreamBytes+1),
		reset: resetIdle,
	}
	return decodeStream(streamCtx, ctx, limited, c.opt, outputLimit, onDelta)
}

// idleResetReader treats any successful underlying Read that returns bytes as
// stream progress, so the idle watchdog is not tied to complete SSE events.
type idleResetReader struct {
	r     io.Reader
	reset func()
}

func (r idleResetReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 && r.reset != nil {
		r.reset()
	}
	return n, err
}

func (c *Client) transportError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return fmt.Errorf("timed out contacting %s; check that the server is running and responding", c.cfg.Endpoint)
	}
	return fmt.Errorf("cannot contact %s: %v; check that the server is running and the endpoint URL is correct", c.cfg.Endpoint, err)
}

func readBounded(r io.Reader, n int) string {
	b, _ := io.ReadAll(io.LimitReader(r, int64(n)+1))
	if len(b) > n {
		b = b[:n]
	}
	s := strings.TrimSpace(string(b))
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "empty error body"
	}
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func snippetHint(snippet string) string {
	return fmt.Sprintf("server said %q; fix the endpoint or model configuration", snippet)
}
