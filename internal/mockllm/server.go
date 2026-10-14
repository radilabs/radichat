package mockllm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// scenarioPathPrefix is the URL path prefix under which scenarios are selected.
const scenarioPathPrefix = "/v1/chat/completions/"

// LoopbackHost is the only host the mock binds.
const LoopbackHost = "127.0.0.1"

// DefaultAddr binds an ephemeral port on the IPv4 loopback interface.
const DefaultAddr = LoopbackHost + ":0"

// errLoopbackRequired is returned when a caller asks the mock to bind a
// non-loopback address.
var errLoopbackRequired = errors.New("mockllm: refusing to bind a non-loopback address; use a loopback host such as 127.0.0.1, [::1], or localhost")

// Options configures a Server.
type Options struct {
	// Addr is the listen address. It must be a loopback address such as
	// "127.0.0.1:0" or "[::1]:0". Empty means DefaultAddr.
	Addr string

	// ExpectedBearer is the token that authenticated scenarios expect. When
	// empty, an authenticated scenario accepts any nonempty bearer token.
	ExpectedBearer string
}

// Server is a mock OpenAI-compatible endpoint.
//
// A Server is safe for concurrent use. Create one with New, start it with
// Start or StartContext, and stop it with Close or Shutdown.
type Server struct {
	opts Options
	ln   net.Listener
	http *http.Server

	mu       sync.Mutex
	recs     []Record
	closed   bool
	started  bool
	serveErr error
	doneChan chan struct{}
}

// New creates a Server without starting it.
func New(opts Options) (*Server, error) {
	if opts.Addr == "" {
		opts.Addr = DefaultAddr
	}
	host, _, err := net.SplitHostPort(opts.Addr)
	if err != nil {
		return nil, fmt.Errorf("mockllm: invalid listen address %q: %w", opts.Addr, err)
	}
	if !isLoopback(host) {
		return nil, errLoopbackRequired
	}
	return &Server{opts: opts, doneChan: make(chan struct{})}, nil
}

// Start creates and starts a Server on a loopback address.
func Start(opts Options) (*Server, error) {
	s, err := New(opts)
	if err != nil {
		return nil, err
	}
	if err := s.Start(); err != nil {
		return nil, err
	}
	return s, nil
}

// StartContext creates and starts a Server that stops when ctx is done.
func StartContext(ctx context.Context, opts Options) (*Server, error) {
	s, err := New(opts)
	if err != nil {
		return nil, err
	}
	if err := s.StartContext(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// Start binds the loopback listener and serves in a background goroutine.
//
// Start returns once the listener is bound, so the caller can read URL
// immediately.
func (s *Server) Start() error {
	return s.StartContext(context.Background())
}

// StartContext binds the listener, serves in a background goroutine, and stops
// the server when ctx is done.
func (s *Server) StartContext(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("mockllm: server already started")
	}
	s.started = true
	addr := s.opts.Addr
	s.mu.Unlock()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("mockllm: cannot listen on %s: %w", addr, err)
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = ln.Close()
		return nil
	}
	s.ln = ln
	s.mu.Unlock()

	s.http = &http.Server{
		Handler:           http.HandlerFunc(s.handle),
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		err := s.http.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		s.mu.Lock()
		s.serveErr = err
		s.mu.Unlock()
		close(s.doneChan)
	}()

	if ctx != nil && ctx.Done() != nil {
		go func() {
			<-ctx.Done()
			_ = s.Close()
		}()
	}
	return nil
}

// URL returns the base URL of the running server, for example
// "http://127.0.0.1:54321". Callers append a scenario path.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return ""
	}
	return "http://" + s.ln.Addr().String()
}

// Addr returns the listen address of the running server.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// Close stops the server immediately. In-flight requests are cancelled rather
// than drained.
//
// Close is safe to call more than once and safe to call concurrently with
// Start.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	h := s.http
	s.mu.Unlock()
	if h == nil {
		return nil
	}
	return h.Close()
}

// Shutdown gracefully stops the server, waiting for in-flight requests to
// finish.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	h := s.http
	s.mu.Unlock()
	if h == nil {
		return nil
	}
	err := h.Shutdown(ctx)
	// Wait for the serve goroutine to observe the shutdown so Done is closed
	// deterministically rather than racing the caller.
	select {
	case <-s.doneChan:
	case <-ctx.Done():
	}
	return err
}

// Done returns a channel closed when the server has stopped serving.
func (s *Server) Done() <-chan struct{} { return s.doneChan }

// ServeErr returns the error the server goroutine observed, if any.
func (s *Server) ServeErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serveErr
}

// Handler returns the server's HTTP handler, for tests that prefer to drive the
// mock through httptest instead of a real socket.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// handle dispatches one request.
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	auth := classifyAuth(r.Header.Get(AuthorizationHeader), s.opts.ExpectedBearer)
	name := scenarioNameFromPath(r.URL.Path)
	sc, known := Lookup(name)
	if !known {
		s.record(r, auth, name, "", sc, false)
		http.Error(w, "mockllm: unknown scenario "+name+"; list scenarios with radichat-mock list", http.StatusNotFound)
		return
	}

	body, err := readRequestBody(r)
	if err != nil {
		s.record(r, auth, name, "", sc, false)
		http.Error(w, "mockllm: cannot read request body", http.StatusBadRequest)
		return
	}
	s.record(r, auth, name, body, sc, true)

	if sc.Auth.Required && !auth.Matched {
		writeStatus(w, sc, http.StatusUnauthorized)
		_, _ = io.WriteString(w, "mockllm: missing or mismatched bearer credential\n")
		return
	}

	if sc.Mode == ModeRaw {
		s.serveRaw(w, r, sc)
		return
	}
	s.serveHTTP(w, r, sc)
}

// serveHTTP writes a scenario through net/http, flushing per segment.
func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request, sc Scenario) {
	if sc.Redirect != "" {
		w.Header().Set("Location", sc.Redirect)
		w.WriteHeader(sc.redirectStatus())
		return
	}
	for k, v := range sc.Headers {
		w.Header().Set(k, v)
	}
	if code := sc.status(); code != http.StatusOK {
		// A non-200 status must be written before any body, so a scenario
		// that declares only a body still produces the intended status.
		w.WriteHeader(code)
	}
	fl, _ := w.(http.Flusher)
	for _, seg := range sc.Wire {
		if seg.Delay > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(seg.Delay):
			}
		}
		if _, err := w.Write(seg.Bytes); err != nil {
			return
		}
		if fl != nil {
			fl.Flush()
		}
		if seg.CloseAfter {
			return
		}
	}
}

// serveRaw hijacks the connection and writes the configured bytes verbatim.
//
// Raw mode is required when a test must control the exact response bytes, for
// example to reproduce framing that net/http would reframe or reject.
func (s *Server) serveRaw(w http.ResponseWriter, r *http.Request, sc Scenario) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "mockllm: connection cannot be hijacked for raw mode", http.StatusInternalServerError)
		return
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()

	if sc.Redirect != "" {
		head := fmt.Sprintf("HTTP/1.1 %d %s\r\nLocation: %s\r\nContent-Length: 0\r\n\r\n",
			sc.redirectStatus(), http.StatusText(sc.redirectStatus()), sc.Redirect)
		_, _ = brw.WriteString(head)
		_ = brw.Flush()
		return
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", sc.status(), http.StatusText(sc.status()))
	_, _ = brw.WriteString(statusLine)
	for k, v := range sc.Headers {
		_, _ = brw.WriteString(k + ": " + v + "\r\n")
	}
	_, _ = brw.WriteString("Connection: close\r\n\r\n")
	_ = brw.Flush()

	for _, seg := range sc.Wire {
		if seg.Delay > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(seg.Delay):
			}
		}
		if _, err := brw.Write(seg.Bytes); err != nil {
			return
		}
		_ = brw.Flush()
		if seg.CloseAfter {
			return
		}
	}
}

// record appends one request observation.
//
// The bearer token value is never stored. Only the classified AuthOutcome
// reaches the Record. The request's query string is deliberately not captured,
// because it can carry a token such as ?access_token=...; only the URL path is
// recorded.
func (s *Server) record(r *http.Request, auth AuthOutcome, name, body string, sc Scenario, known bool) {
	rec := Record{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: recordedHeaders(r.Header),
		Body:    decodeBody([]byte(body)),
		BodyRaw: body,
		Auth:    auth,
	}
	if known {
		rec.Scenario = name
		rec.ExpectCheck = checkExpect(sc, &rec)
	}
	s.mu.Lock()
	s.recs = append(s.recs, rec)
	s.mu.Unlock()
}

// Records returns a copy of the recorded request observations.
func (s *Server) Records() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.recs))
	copy(out, s.recs)
	return out
}

// ResetRecords clears recorded observations. It does not stop the server.
func (s *Server) ResetRecords() {
	s.mu.Lock()
	s.recs = nil
	s.mu.Unlock()
}

// Last returns the most recent record, or false when none exist.
func (s *Server) Last() (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.recs) == 0 {
		return Record{}, false
	}
	return s.recs[len(s.recs)-1], true
}

// scenarioNameFromPath extracts the scenario name from a URL path.
func scenarioNameFromPath(path string) string {
	if !strings.HasPrefix(path, scenarioPathPrefix) {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(path, scenarioPathPrefix), "/")
}

// isLoopback reports whether host is a loopback address. A host name such as
// "localhost" is accepted because it resolves to loopback.
func isLoopback(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1", "0:0:0:0:0:0:0:1":
		return true
	}
	if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// writeStatus writes the scenario's headers and a status code.
func writeStatus(w http.ResponseWriter, sc Scenario, code int) {
	for k, v := range sc.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(code)
}
