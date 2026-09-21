package chat

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEstimateBytesNotRunes(t *testing.T) {
	msg := Message{Role: "user", Content: "日本語"} // 9 UTF-8 bytes, 3 runes
	if utf8.RuneCountInString(msg.Content) == len(msg.Content) {
		t.Fatal("fixture is not multibyte")
	}
	got, err := Estimate([]Message{msg})
	if err != nil {
		t.Fatal(err)
	}
	want := int64(PerRequest + PerMessage + len(msg.Content))
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestEstimateEmojiAndAccents(t *testing.T) {
	msg := Message{Role: "user", Content: "é🙂"}
	got, err := Estimate([]Message{msg})
	if err != nil {
		t.Fatal(err)
	}
	want := int64(PerRequest + PerMessage + len("é🙂"))
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestFitsExactBoundary(t *testing.T) {
	// one user message of 4 bytes: units = 32 + 32+4 = 68; plus reserve 32 = 100
	user := Message{Role: "user", Content: "abcd"}
	ok, err := Fits([]Message{user}, 100, 32)
	if err != nil || !ok {
		t.Fatalf("should fit exactly: ok=%v err=%v est must be 68", ok, err)
	}
	ok, err = Fits([]Message{user}, 99, 32)
	if err != nil || ok {
		t.Fatalf("one unit over should not fit: ok=%v err=%v", ok, err)
	}
}

func TestStageTrimsOldestPairsAndPreservesProtected(t *testing.T) {
	c := New(200, 40, "sys") // system "sys" = 3 bytes
	// Fill with pairs that will need trimming.
	for i := 0; i < 4; i++ {
		if _, _, err := c.Stage(strings.Repeat("u", 10)); err != nil {
			t.Fatal(err)
		}
		if err := c.Commit(strings.Repeat("a", 10)); err != nil {
			t.Fatal(err)
		}
	}
	before := c.History()
	msgs, trimmed, err := c.Stage("current-user-message")
	if err != nil {
		t.Fatal(err)
	}
	if trimmed == 0 {
		t.Fatal("expected trimming")
	}
	if len(c.History()) != len(before) {
		t.Fatal("committed history mutated during Stage")
	}
	if msgs[0].Role != "system" || msgs[0].Content != "sys" {
		t.Fatalf("system not preserved: %+v", msgs[0])
	}
	last := msgs[len(msgs)-1]
	if last.Role != "user" || last.Content != "current-user-message" {
		t.Fatalf("current user not preserved: %+v", last)
	}
	ok, err := Fits(msgs, 200, 40)
	if err != nil || !ok {
		t.Fatalf("admitted request does not fit: ok=%v err=%v", ok, err)
	}
	if err := assertPairs(msgs[1 : len(msgs)-1]); err != nil {
		t.Fatal(err)
	}
}

func TestRejectOversizedProtectedWithoutMutation(t *testing.T) {
	c := New(250, 40, strings.Repeat("s", 20))
	if _, _, err := c.Stage("hi"); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("yo"); err != nil {
		t.Fatal(err)
	}
	hist := append([]Message(nil), c.History()...)
	huge := strings.Repeat("x", 200)
	_, _, err := c.Stage(huge)
	if err == nil {
		t.Fatal("expected local rejection")
	}
	if !strings.Contains(err.Error(), "does not fit") {
		t.Fatalf("error: %v", err)
	}
	if c.HasStaged() {
		t.Fatal("failed stage left staged state")
	}
	got := c.History()
	if len(got) != len(hist) || got[0] != hist[0] {
		t.Fatalf("history mutated: %+v vs %+v", got, hist)
	}
}

func TestCommitOnlyAfterSuccessAndAbortRollback(t *testing.T) {
	c := New(500, 50, "")
	if _, _, err := c.Stage("one"); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("r1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Stage("two"); err != nil {
		t.Fatal(err)
	}
	c.Abort()
	if c.HasStaged() {
		t.Fatal("abort left staged turn")
	}
	h := c.History()
	if len(h) != 2 || h[0].Content != "one" || h[1].Content != "r1" {
		t.Fatalf("rollback failed: %+v", h)
	}
	if _, _, err := c.Stage("three"); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("r3"); err != nil {
		t.Fatal(err)
	}
	h = c.History()
	if len(h) != 4 || h[2].Content != "three" {
		t.Fatalf("commit after abort: %+v", h)
	}
}

func TestClearKeepsSystem(t *testing.T) {
	c := New(500, 50, "keep me")
	if _, _, err := c.Stage("u"); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("a"); err != nil {
		t.Fatal(err)
	}
	c.Clear()
	if len(c.History()) != 0 {
		t.Fatal("clear left history")
	}
	msgs, _, err := c.Stage("again")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Content != "keep me" || msgs[1].Content != "again" {
		t.Fatalf("system not retained after clear: %+v", msgs)
	}
}

func TestRepeatedInputDeterministic(t *testing.T) {
	run := func() ([][]Message, []int) {
		c := New(180, 30, "s")
		var reqs [][]Message
		var trims []int
		for i := 0; i < 6; i++ {
			msgs, n, err := c.Stage("same")
			if err != nil {
				t.Fatal(err)
			}
			reqs = append(reqs, msgs)
			trims = append(trims, n)
			if err := c.Commit("ans"); err != nil {
				t.Fatal(err)
			}
		}
		return reqs, trims
	}
	a, at := run()
	b, bt := run()
	if len(a) != len(b) {
		t.Fatal("length mismatch")
	}
	for i := range a {
		if at[i] != bt[i] {
			t.Fatalf("trim %d: %d vs %d", i, at[i], bt[i])
		}
		if len(a[i]) != len(b[i]) {
			t.Fatalf("req %d len", i)
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				t.Fatalf("req %d msg %d: %+v vs %+v", i, j, a[i][j], b[i][j])
			}
		}
	}
}

func TestLargeReserveLeavesLittlePromptRoom(t *testing.T) {
	c := New(100, 90, "")
	_, _, err := c.Stage("hello world") // 11 bytes + 64 overhead + 90 reserve = 165 > 100
	if err == nil {
		t.Fatal("expected rejection with large reserve")
	}
	_, _, err = c.Stage("h") // 1+64+90 = 155 > 100
	if err == nil {
		t.Fatal("expected rejection")
	}
	msgs, _, err := c.Stage("") // 0+64+90 = 154 still > 100
	if err == nil {
		t.Fatalf("empty user still needs 64+90 units, got msgs=%v", msgs)
	}
}

func TestOverflowArithmetic(t *testing.T) {
	if _, err := addUnits(math.MaxInt64, 1); err == nil {
		t.Fatal("expected overflow")
	}
	if _, err := addUnits(-1, 1); err == nil {
		t.Fatal("expected overflow on negative")
	}
	got, err := addUnits(1, 2)
	if err != nil || got != 3 {
		t.Fatalf("addUnits: %d %v", got, err)
	}
	_, err = Estimate([]Message{{Content: strings.Repeat("a", 8)}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSONEscapeNotCharged(t *testing.T) {
	// Content quotes would expand in JSON; accounting uses raw bytes.
	msg := Message{Role: "user", Content: `"quoted"`}
	got, err := Estimate([]Message{msg})
	if err != nil {
		t.Fatal(err)
	}
	want := int64(PerRequest + PerMessage + len(`"quoted"`))
	if got != want {
		t.Fatalf("charged JSON escapes? got %d want %d", got, want)
	}
}

func TestPropertyAdmittedRequestsFitAndPairs(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 100; trial++ {
		budget := int64(rng.Intn(800) + 200)
		reserve := int64(rng.Intn(int(budget/4)) + 1)
		var sys string
		if rng.Intn(2) == 0 {
			sys = randString(rng, rng.Intn(20))
		}
		c := New(budget, reserve, sys)
		for step := 0; step < 12; step++ {
			user := randString(rng, rng.Intn(40))
			histBefore := append([]Message(nil), c.History()...)
			msgs, _, err := c.Stage(user)
			if err != nil {
				if c.HasStaged() {
					t.Fatal("failed stage left staged")
				}
				if len(c.History()) != len(histBefore) {
					t.Fatal("failed stage mutated history")
				}
				continue
			}
			ok, ferr := Fits(msgs, budget, reserve)
			if ferr != nil || !ok {
				t.Fatalf("admitted request does not fit: ok=%v err=%v est msgs=%v", ok, ferr, msgs)
			}
			if sys != "" {
				if msgs[0].Role != "system" || msgs[0].Content != sys {
					t.Fatalf("system not retained")
				}
			}
			last := msgs[len(msgs)-1]
			if last.Role != "user" || last.Content != user {
				t.Fatal("current user not retained")
			}
			body := msgs
			if sys != "" {
				body = msgs[1:]
			}
			if err := assertPairs(body[:len(body)-1]); err != nil {
				t.Fatal(err)
			}
			if err := c.Commit(randString(rng, rng.Intn(30))); err != nil {
				t.Fatal(err)
			}
			if err := assertPairs(c.History()); err != nil {
				t.Fatal(err)
			}
		}
		c.Clear()
		if len(c.History()) != 0 {
			t.Fatal("clear")
		}
	}
}

func randString(rng *rand.Rand, n int) string {
	if n == 0 {
		return ""
	}
	var b strings.Builder
	alphabet := []rune("abé日🙂 ")
	for i := 0; i < n; i++ {
		b.WriteRune(alphabet[rng.Intn(len(alphabet))])
	}
	return b.String()
}

func assertPairs(msgs []Message) error {
	if len(msgs)%2 != 0 {
		return fmt.Errorf("odd history length %d", len(msgs))
	}
	for i := 0; i < len(msgs); i += 2 {
		if msgs[i].Role != "user" || msgs[i+1].Role != "assistant" {
			return fmt.Errorf("history is not user/assistant pairs at %d: %s/%s", i, msgs[i].Role, msgs[i+1].Role)
		}
	}
	return nil
}
