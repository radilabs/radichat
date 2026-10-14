package mockllm

import (
	"strings"
	"time"
)

// Segment is one write in a scenario's response wire.
//
// A Segment carries exact bytes, an optional delay before the write, and an
// optional close-after flag. The segments of a scenario are the deterministic
// unit: the same scenario always produces the same segments in the same order.
//
// Segments describe what the mock writes, not what a reader observes. The
// operating system and network stack may coalesce or split writes, so a test
// must assert the configured segments and, separately, that a client
// reassembles the concatenated bytes correctly. It must not assert that a
// reader sees one segment per TCP read.
type Segment struct {
	// Bytes is written verbatim.
	Bytes []byte

	// Delay is applied before the write. It is honored in ModeHTTP, where it
	// can model a slow or fragmented stream. The package's own tests never
	// use it, so they stay fast and deterministic.
	Delay time.Duration

	// CloseAfter makes the mock close the connection immediately after this
	// segment, without any trailing bytes. It models early termination.
	CloseAfter bool
}

// Text returns the segment bytes as a string, for readable diagnostics and
// documentation.
func (s Segment) Text() string { return string(s.Bytes) }

// WireBytes concatenates a scenario's segments into the exact byte stream the
// mock writes.
func WireBytes(segments []Segment) []byte {
	n := 0
	for _, seg := range segments {
		n += len(seg.Bytes)
	}
	out := make([]byte, 0, n)
	for _, seg := range segments {
		out = append(out, seg.Bytes...)
	}
	return out
}

// WireText is WireBytes as a string.
func WireText(segments []Segment) string { return string(WireBytes(segments)) }

// Fragment splits data at the given fixed byte offsets.
//
// Offsets must be strictly increasing and lie strictly inside data. An invalid
// offset makes Fragment return nil rather than silently produce an empty or
// overlapping segment, so a caller sees a wrong segment count instead of
// plausible-looking garbage.
//
// Fragment is deterministic: the same data and offsets always yield the same
// segments. It says nothing about how the bytes are later delivered or read.
func Fragment(data []byte, offsets ...int) []Segment {
	type span struct{ start, end int }
	spans := make([]span, 0, len(offsets)+1)
	prev := 0
	for _, off := range offsets {
		if off <= prev || off >= len(data) {
			return nil
		}
		spans = append(spans, span{prev, off})
		prev = off
	}
	spans = append(spans, span{prev, len(data)})
	out := make([]Segment, 0, len(spans))
	for _, sp := range spans {
		out = append(out, Segment{Bytes: data[sp.start:sp.end]})
	}
	return out
}

// FragmentAt returns data as a single segment, for scenarios that need no
// splitting.
func FragmentAt(data []byte) []Segment {
	return []Segment{{Bytes: data}}
}

// FragmentText is FragmentAt for text.
func FragmentText(s string) []Segment {
	return []Segment{{Bytes: []byte(s)}}
}

// midRuneOffset returns a byte offset that lies inside a multibyte rune of s,
// or -1 when s has no multibyte rune.
//
// It is used to build fragments that split inside a UTF-8 sequence, which a
// strict parser must reassemble rather than reject.
func midRuneOffset(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i]&0xC0 == 0x80 { // continuation byte
			return i
		}
	}
	return -1
}

// firstCRLFSplit returns an offset between a CR and the LF that follows it, or
// -1 when s has no CRLF.
func firstCRLFSplit(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '\r' && s[i+1] == '\n' {
			return i + 1
		}
	}
	return -1
}

// firstDataFieldSplit returns an offset inside the first "data:" field, after
// the field name and before its payload, or -1 when there is none.
func firstDataFieldSplit(s string) int {
	idx := indexOf(s, "data:")
	if idx < 0 {
		return -1
	}
	// Skip the field name and one separating space, then land inside the
	// payload.
	return idx + len("data:") + 1
}

// firstDoneSplit returns an offset inside the first [DONE] marker, or -1 when
// there is none.
func firstDoneSplit(s string) int {
	idx := indexOf(s, "[DONE]")
	if idx < 0 {
		return -1
	}
	// Land inside the marker, not at its boundary.
	return idx + 3
}

// indexOf is strings.Index.
func indexOf(s, sub string) int {
	return strings.Index(s, sub)
}
