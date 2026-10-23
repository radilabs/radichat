package repl

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestJSONLWriterRecordShapes(t *testing.T) {
	cases := []struct {
		name string
		rec  Record
		want string
	}{
		{
			name: "turn ok",
			rec: Record{
				V: 1, Line: 3, Kind: "turn", Status: "ok",
				Input: strPtr("hello"), Output: strPtr("hi"),
				FinishReason: strPtr("stop"), TrimmedPairs: intPtr(0),
			},
			want: `{"v":1,"line":3,"kind":"turn","status":"ok","input":"hello","output":"hi","finish_reason":"stop","trimmed_pairs":0}`,
		},
		{
			name: "turn error sent",
			rec: Record{
				V: 1, Line: 4, Kind: "turn", Status: "error",
				Sent: boolPtr(true), Input: strPtr("again"), Error: "endpoint returned HTTP 500",
			},
			want: `{"v":1,"line":4,"kind":"turn","status":"error","sent":true,"input":"again","error":"endpoint returned HTTP 500"}`,
		},
		{
			name: "turn error unsent overlong",
			rec: Record{
				V: 1, Line: 5, Kind: "turn", Status: "error",
				Sent: boolPtr(false), Error: "line too long",
			},
			want: `{"v":1,"line":5,"kind":"turn","status":"error","sent":false,"error":"line too long"}`,
		},
		{
			name: "interrupted",
			rec: Record{
				V: 1, Line: 6, Kind: "turn", Status: "interrupted",
				Sent: boolPtr(true), Input: strPtr("continue"),
			},
			want: `{"v":1,"line":6,"kind":"turn","status":"interrupted","sent":true,"input":"continue"}`,
		},
		{
			name: "clear",
			rec:  Record{V: 1, Line: 7, Kind: "clear"},
			want: `{"v":1,"line":7,"kind":"clear"}`,
		},
		{
			name: "quit",
			rec:  Record{V: 1, Line: 8, Kind: "quit"},
			want: `{"v":1,"line":8,"kind":"quit"}`,
		},
		{
			name: "command error",
			rec: Record{
				V: 1, Line: 9, Kind: "command_error",
				Input: strPtr(" /unknown "), Error: "unknown command",
			},
			want: `{"v":1,"line":9,"kind":"command_error","input":" /unknown ","error":"unknown command"}`,
		},
		{
			name: "empty finish and output",
			rec: Record{
				V: 1, Line: 1, Kind: "turn", Status: "ok",
				Input: strPtr("q"), Output: strPtr(""),
				FinishReason: strPtr(""), TrimmedPairs: intPtr(0),
			},
			want: `{"v":1,"line":1,"kind":"turn","status":"ok","input":"q","output":"","finish_reason":"","trimmed_pairs":0}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewJSONLWriter(&buf)
			if err := w.WriteRecord(tc.rec); err != nil {
				t.Fatal(err)
			}
			got := strings.TrimSuffix(buf.String(), "\n")
			if got != tc.want {
				t.Fatalf("got %s\nwant %s", got, tc.want)
			}
			if strings.Count(buf.String(), "\n") != 1 || !strings.HasSuffix(buf.String(), "\n") {
				t.Fatalf("framing %q", buf.String())
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			for _, v := range raw {
				if string(v) == "null" {
					t.Fatalf("null field in %s", buf.String())
				}
			}
		})
	}
}

func TestJSONLWriterEscapesAndRoundTrips(t *testing.T) {
	in := "quote \" slash \\ control \n \t \x01 html </>& multibyte héllo 😀"
	rec := Record{
		V: 1, Line: 1, Kind: "turn", Status: "ok",
		Input: strPtr(in), Output: strPtr(in),
		FinishReason: strPtr("stop"), TrimmedPairs: intPtr(2),
	}
	var buf bytes.Buffer
	if err := NewJSONLWriter(&buf).WriteRecord(rec); err != nil {
		t.Fatal(err)
	}
	raw := buf.String()
	if strings.Contains(raw, `\u003c`) || strings.Contains(raw, `\u003e`) || strings.Contains(raw, `\u0026`) {
		t.Fatalf("html escaped: %s", raw)
	}
	if !strings.Contains(raw, "</>&") {
		t.Fatalf("html missing: %s", raw)
	}
	if !strings.Contains(raw, "héllo 😀") {
		t.Fatalf("utf-8 missing: %s", raw)
	}
	var got Record
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Input == nil || *got.Input != in || got.Output == nil || *got.Output != in {
		t.Fatalf("round trip %+v", got)
	}
	if got.TrimmedPairs == nil || *got.TrimmedPairs != 2 {
		t.Fatalf("trimmed %+v", got.TrimmedPairs)
	}
}

func TestJSONLWriterOneWrite(t *testing.T) {
	var cw countWriter
	w := NewJSONLWriter(&cw)
	rec := Record{V: 1, Line: 1, Kind: "quit"}
	if err := w.WriteRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteRecord(rec); err != nil {
		t.Fatal(err)
	}
	if cw.writes != 2 {
		t.Fatalf("writes %d", cw.writes)
	}
	if !bytes.HasSuffix(cw.buf.Bytes(), []byte("\n")) {
		t.Fatalf("buf %q", cw.buf.String())
	}
}

func TestJSONLWriterShortWriteAndError(t *testing.T) {
	w := NewJSONLWriter(shortWriter{})
	err := w.WriteRecord(Record{V: 1, Line: 1, Kind: "quit"})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short %v", err)
	}
	w = NewJSONLWriter(errWriter{errors.New("disk full")})
	err = w.WriteRecord(Record{V: 1, Line: 1, Kind: "quit"})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err %v", err)
	}
}

type countWriter struct {
	writes int
	buf    bytes.Buffer
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.writes++
	return c.buf.Write(p)
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return 1, nil
}

type errWriter struct{ err error }

func (e errWriter) Write(p []byte) (int, error) {
	return 0, e.err
}
