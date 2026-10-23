package client

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/chat"
)

func TestFinishReasonStopLengthAndEmpty(t *testing.T) {
	cases := []struct {
		name   string
		finish string
		want   string
	}{
		{name: "stop", finish: "stop", want: "stop"},
		{name: "length", finish: "length", want: "length"},
		{name: "empty", finish: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, sseChunk("hi", tc.finish))
				if tc.finish == "" {
					io.WriteString(w, "data: [DONE]\n\n")
				} else {
					io.WriteString(w, "data: [DONE]\n\n")
				}
			}))
			t.Cleanup(srv.Close)
			res, err := testClient(srv.URL).StreamResult(context.Background(), []chat.Message{{Role: "user", Content: "q"}}, 8192, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Text != "hi" {
				t.Fatalf("text %q", res.Text)
			}
			if res.FinishReason != tc.want {
				t.Fatalf("finish %q", res.FinishReason)
			}
			if !res.RequestSent {
				t.Fatal("expected request sent after a successful write")
			}
		})
	}
}

func TestFinishReasonUnsupportedRemainsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("", "content_filter"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	res, err := testClient(srv.URL).StreamResult(context.Background(), []chat.Message{{Role: "user", Content: "q"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "content_filter") {
		t.Fatalf("err %v result %+v", err, res)
	}
	if !res.RequestSent {
		t.Fatal("request was written before the finish error")
	}
}

func TestFinishReasonRequestSentFalseBeforeWrite(t *testing.T) {
	c := New(testCfg("http://127.0.0.1:9/v1/chat/completions"), Options{
		DialTimeout:           50 * time.Millisecond,
		TLSHandshakeTimeout:   50 * time.Millisecond,
		ResponseHeaderTimeout: 50 * time.Millisecond,
		IdleReadTimeout:       50 * time.Millisecond,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("dial refused before write")
		},
	})
	res, err := c.StreamResult(context.Background(), []chat.Message{{Role: "user", Content: "q"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected dial error")
	}
	if res.RequestSent {
		t.Fatal("request marked sent before a successful write")
	}
	if res.Text != "" || res.FinishReason != "" {
		t.Fatalf("result %+v", res)
	}
}

func TestStreamCompatibilityPreservesText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("pong", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	c := testClient(srv.URL)
	msgs := []chat.Message{{Role: "user", Content: "q"}}
	text, err := c.Stream(context.Background(), msgs, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "pong" {
		t.Fatalf("stream text %q", text)
	}
	res, err := c.StreamResult(context.Background(), msgs, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != text || res.FinishReason != "stop" || !res.RequestSent {
		t.Fatalf("result %+v", res)
	}
}
