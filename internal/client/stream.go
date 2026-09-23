package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type streamChunk struct {
	Error   json.RawMessage `json:"error"`
	Choices []streamChoice  `json:"choices"`
}

type streamChoice struct {
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type streamDelta struct {
	Role         string          `json:"role"`
	Content      json.RawMessage `json:"content"`
	ToolCalls    json.RawMessage `json:"tool_calls"`
	FunctionCall json.RawMessage `json:"function_call"`
}

func decodeStream(streamCtx, parent context.Context, r io.Reader, opt Options, outputLimit int64, onDelta func(string) error) (string, error) {
	br := bufio.NewReaderSize(r, 32*1024)
	var assembled strings.Builder
	gotDone := false
	var finish string
	sawChoice := false
	totalRead := int64(0)

	for {
		if err := parent.Err(); err != nil {
			return assembled.String(), err
		}
		if err := streamCtx.Err(); err != nil {
			return assembled.String(), fmt.Errorf("idle timeout waiting for stream data; the connection was closed")
		}
		event, n, err := readSSEEvent(br, opt.MaxEventBytes)
		totalRead += int64(n)
		if totalRead > opt.MaxStreamBytes {
			return assembled.String(), fmt.Errorf("stream exceeded the %d-byte local read limit; the turn was discarded", opt.MaxStreamBytes)
		}
		if err != nil {
			if parent.Err() != nil {
				return assembled.String(), parent.Err()
			}
			if streamCtx.Err() != nil {
				return assembled.String(), fmt.Errorf("idle timeout waiting for stream data; the connection was closed")
			}
			if errors.Is(err, io.EOF) {
				if gotDone {
					break
				}
				return assembled.String(), fmt.Errorf("stream ended before [DONE]; the endpoint closed the connection early")
			}
			return assembled.String(), err
		}
		if event == nil {
			continue
		}
		data := strings.TrimSpace(event.data)
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			gotDone = true
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return assembled.String(), fmt.Errorf("malformed stream JSON; the endpoint sent an event RadiChat cannot parse")
		}
		if err := classifyStreamError(chunk.Error, opt.omitErrorSnippets); err != nil {
			return assembled.String(), err
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if len(chunk.Choices) != 1 {
			return assembled.String(), fmt.Errorf("stream included %d choices; RadiChat supports exactly one choice", len(chunk.Choices))
		}
		sawChoice = true
		ch := chunk.Choices[0]
		if len(ch.Delta.ToolCalls) > 0 && string(ch.Delta.ToolCalls) != "null" && string(ch.Delta.ToolCalls) != "[]" {
			return assembled.String(), fmt.Errorf("endpoint requested tool calls, which Phase 1 does not support")
		}
		if len(ch.Delta.FunctionCall) > 0 && string(ch.Delta.FunctionCall) != "null" {
			return assembled.String(), fmt.Errorf("endpoint requested a function call, which Phase 1 does not support")
		}
		text, err := deltaText(ch.Delta.Content)
		if err != nil {
			return assembled.String(), err
		}
		if text != "" {
			if outputLimit > 0 && int64(assembled.Len())+int64(len(text)) > outputLimit {
				return assembled.String(), fmt.Errorf("assistant output exceeded the local %d-byte bound derived from context_budget; the turn was discarded", outputLimit)
			}
			assembled.WriteString(text)
			if onDelta != nil {
				if err := onDelta(text); err != nil {
					return assembled.String(), err
				}
			}
		}
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			finish = *ch.FinishReason
		}
	}

	if !gotDone {
		return assembled.String(), fmt.Errorf("stream ended before [DONE]; the endpoint closed the connection early")
	}
	if !sawChoice && assembled.Len() == 0 {
		return "", fmt.Errorf("stream completed without any choices; check that the model name is valid")
	}
	switch finish {
	case "", "stop", "length":
		return assembled.String(), nil
	case "tool_calls", "function_call":
		return assembled.String(), fmt.Errorf("endpoint finished with %s, which Phase 1 does not support", finish)
	case "content_filter":
		return assembled.String(), fmt.Errorf("endpoint refused the completion (content_filter)")
	default:
		return assembled.String(), fmt.Errorf("endpoint finished with unsupported reason %q", finish)
	}
}

type sseEvent struct {
	data string
}

func readSSEEvent(br *bufio.Reader, maxBytes int) (*sseEvent, int, error) {
	var dataLines []string
	size := 0
	gotField := false
	for {
		line, n, err := readSSELine(br, maxBytes-size)
		size += n
		if err != nil {
			if errors.Is(err, io.EOF) && !gotField && size == 0 {
				return nil, size, io.EOF
			}
			if errors.Is(err, io.EOF) {
				return nil, size, fmt.Errorf("stream ended before [DONE]; the endpoint closed the connection early")
			}
			return nil, size, err
		}
		if line == "" {
			if !gotField {
				return nil, size, nil
			}
			return &sseEvent{data: strings.Join(dataLines, "\n")}, size, nil
		}
		gotField = true
		if strings.HasPrefix(line, ":") {
			continue
		}
		name, value := splitSSEField(line)
		switch name {
		case "data":
			dataLines = append(dataLines, value)
		case "event", "id", "retry":
			// ignored; RadiChat only consumes data payloads
		}
	}
}

func readSSELine(br *bufio.Reader, remaining int) (string, int, error) {
	if remaining <= 0 {
		return "", 0, fmt.Errorf("SSE event exceeded the %d-byte local limit", remaining)
	}
	var buf []byte
	nread := 0
	for {
		b, err := br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(buf) == 0 && nread == 0 {
				return "", nread, io.EOF
			}
			if errors.Is(err, io.EOF) {
				return "", nread, io.EOF
			}
			if isTimeout(err) {
				return "", nread, fmt.Errorf("idle timeout waiting for stream data; the connection was closed")
			}
			return "", nread, err
		}
		nread++
		if nread > remaining {
			return "", nread, fmt.Errorf("SSE event exceeded the local size limit")
		}
		if b == '\n' {
			if len(buf) > 0 && buf[len(buf)-1] == '\r' {
				buf = buf[:len(buf)-1]
			}
			return string(buf), nread, nil
		}
		if b == '\r' {
			next, err := br.ReadByte()
			if err == nil {
				nread++
				if next != '\n' {
					if err := br.UnreadByte(); err != nil {
						buf = append(buf, b)
						continue
					}
					nread--
				}
			}
			return string(buf), nread, nil
		}
		buf = append(buf, b)
	}
}

func splitSSEField(line string) (name, value string) {
	name, value, ok := strings.Cut(line, ":")
	if !ok {
		return line, ""
	}
	if strings.HasPrefix(value, " ") {
		value = value[1:]
	}
	return name, value
}

func classifyStreamError(raw json.RawMessage, omitSnippets bool) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] != '{' {
		if omitSnippets {
			return fmt.Errorf("malformed stream error payload; expected a JSON object or null; server response omitted")
		}
		return fmt.Errorf("malformed stream error payload; expected a JSON object or null, %s", snippetHint(string(trimmed)))
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		if omitSnippets {
			return fmt.Errorf("malformed stream error payload; expected a JSON object or null; server response omitted")
		}
		return fmt.Errorf("malformed stream error payload; expected a JSON object or null, %s", snippetHint(string(trimmed)))
	}
	if omitSnippets {
		return fmt.Errorf("endpoint error object in stream; %s", omittedBodyHint)
	}
	return fmt.Errorf("endpoint error object in stream; %s", snippetHint(string(trimmed)))
}

func deltaText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("stream delta content was not a JSON string; the endpoint is not using the supported protocol subset")
	}
	return s, nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "i/o timeout")
}
