package repl

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestInputReaderLinesAndNumbers(t *testing.T) {
	r := newInputReader(strings.NewReader("one\n\n  \nthree\n"), 100)
	got := readAll(t, r)
	want := []inputLine{
		{Number: 1, Text: "one"},
		{Number: 2, Text: ""},
		{Number: 3, Text: "  "},
		{Number: 4, Text: "three"},
	}
	if len(got) != len(want) {
		t.Fatalf("lines %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestInputReaderEOFWithAndWithoutFinalLF(t *testing.T) {
	t.Run("with lf", func(t *testing.T) {
		r := newInputReader(strings.NewReader("one\n"), 100)
		line, err := r.ReadLine(context.Background())
		if err != nil || line != (inputLine{Number: 1, Text: "one"}) {
			t.Fatalf("first %+v %v", line, err)
		}
		_, err = r.ReadLine(context.Background())
		if !errors.Is(err, io.EOF) {
			t.Fatalf("eof %v", err)
		}
	})
	t.Run("without lf", func(t *testing.T) {
		r := newInputReader(strings.NewReader("one\ntwo"), 100)
		line, err := r.ReadLine(context.Background())
		if err != nil || line.Text != "one" || line.Number != 1 {
			t.Fatalf("first %+v %v", line, err)
		}
		line, err = r.ReadLine(context.Background())
		if !errors.Is(err, io.EOF) || line.Text != "two" || line.Number != 2 {
			t.Fatalf("last %+v %v", line, err)
		}
	})
}

func TestInputReaderCRLFMatchesREPL(t *testing.T) {
	r := newInputReader(strings.NewReader("a\r\nb\r\r\nc\r"), 100)
	got := readAll(t, r)
	want := []inputLine{
		{Number: 1, Text: "a"},
		{Number: 2, Text: "b"},
		{Number: 3, Text: "c"},
	}
	if len(got) != len(want) {
		t.Fatalf("lines %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestInputReaderExactLimitAndOverlongDrain(t *testing.T) {
	exact := strings.Repeat("a", 4)
	r := newInputReader(strings.NewReader(exact+"\nnext\n"), 4)
	line, err := r.ReadLine(context.Background())
	if err != nil || line.Text != exact || line.Number != 1 {
		t.Fatalf("exact %+v %v", line, err)
	}
	line, err = r.ReadLine(context.Background())
	if err != nil || line.Text != "next" || line.Number != 2 {
		t.Fatalf("next %+v %v", line, err)
	}

	over := strings.Repeat("b", 5)
	r = newInputReader(strings.NewReader(over+"\nkept\n"), 4)
	line, err = r.ReadLine(context.Background())
	if !errors.Is(err, errLineTooLong) || line.Number != 1 || line.Text != "" {
		t.Fatalf("overlong %+v %v", line, err)
	}
	line, err = r.ReadLine(context.Background())
	if err != nil || line.Text != "kept" || line.Number != 2 {
		t.Fatalf("after drain %+v %v", line, err)
	}
}

func TestInputReaderCancellation(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		r := newInputReader(pr, 100)
		_, err := r.ReadLine(ctx)
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, errInterrupted) && !errors.Is(err, context.Canceled) {
			t.Fatalf("err %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadLine did not return on cancel")
	}
}

func TestInputReaderPipeDoesNotRequireSeek(t *testing.T) {
	pr, pw := io.Pipe()
	r := newInputReader(pr, 100)
	go func() {
		_, _ = io.WriteString(pw, "piped\n")
		_ = pw.Close()
	}()
	line, err := r.ReadLine(context.Background())
	if err != nil || line.Text != "piped" || line.Number != 1 {
		t.Fatalf("pipe %+v %v", line, err)
	}
	_, err = r.ReadLine(context.Background())
	if !errors.Is(err, io.EOF) {
		t.Fatalf("eof %v", err)
	}
}

func readAll(t *testing.T, r *inputReader) []inputLine {
	t.Helper()
	var lines []inputLine
	for {
		line, err := r.ReadLine(context.Background())
		if errors.Is(err, io.EOF) {
			if line.Text != "" || line.Number != 0 {
				lines = append(lines, line)
			}
			return lines
		}
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
}
