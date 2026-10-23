package repl

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
)

// inputLine is one physical input line. Number is 1-based and counts blanks.
// Text has the same CR removal the REPL applied before file input existed.
type inputLine struct {
	Number int
	Text   string
}

// inputReader owns bounded line reading and physical line numbers.
// It works for any io.Reader, including pipes and FIFOs.
type inputReader struct {
	br   *bufio.Reader
	max  int64
	line int
}

func newInputReader(r io.Reader, max int64) *inputReader {
	return &inputReader{br: bufio.NewReader(r), max: max}
}

func (r *inputReader) ReadLine(ctx context.Context) (inputLine, error) {
	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		text, err := readBoundedLine(r.br, r.max)
		ch <- result{text, err}
	}()
	select {
	case <-ctx.Done():
		return inputLine{}, errInterrupted
	case res := <-ch:
		if errors.Is(res.err, io.EOF) && res.text == "" {
			return inputLine{}, io.EOF
		}
		if res.err != nil && !errors.Is(res.err, io.EOF) && !errors.Is(res.err, errLineTooLong) {
			return inputLine{}, res.err
		}
		r.line++
		if errors.Is(res.err, errLineTooLong) {
			return inputLine{Number: r.line}, errLineTooLong
		}
		return inputLine{Number: r.line, Text: strings.TrimRight(res.text, "\r")}, res.err
	}
}

func readBoundedLine(br *bufio.Reader, max int64) (string, error) {
	var b []byte
	for {
		c, err := br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(b) == 0 {
					return "", io.EOF
				}
				return string(b), io.EOF
			}
			return "", err
		}
		if c == '\n' {
			if len(b) > 0 && b[len(b)-1] == '\r' {
				b = b[:len(b)-1]
			}
			return string(b), nil
		}
		if int64(len(b)) >= max {
			for {
				c2, err2 := br.ReadByte()
				if err2 != nil || c2 == '\n' {
					break
				}
			}
			return "", errLineTooLong
		}
		b = append(b, c)
	}
}
