package repl

import (
	"bytes"
	"encoding/json"
	"io"
)

// Record is one JSONL v1 result. Absent fields stay nil or empty so they
// are omitted. Empty output and finish reason are pointers so a successful
// empty value is still written.
type Record struct {
	V            int     `json:"v"`
	Line         int     `json:"line"`
	Kind         string  `json:"kind"`
	Status       string  `json:"status,omitempty"`
	Sent         *bool   `json:"sent,omitempty"`
	Input        *string `json:"input,omitempty"`
	Output       *string `json:"output,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
	TrimmedPairs *int    `json:"trimmed_pairs,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// RecordWriter accepts one completed record.
type RecordWriter interface {
	WriteRecord(Record) error
}

// JSONLWriter encodes each record as one JSON line and writes it once.
type JSONLWriter struct {
	w io.Writer
}

func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{w: w}
}

func (j *JSONLWriter) WriteRecord(rec Record) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(rec); err != nil {
		return err
	}
	n, err := j.w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if n != buf.Len() {
		return io.ErrShortWrite
	}
	return nil
}

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func intPtr(n int) *int { return &n }
