package chat

import (
	"fmt"
	"math"
)

const (
	PerMessage = 32
	PerRequest = 32
)

// Message is one chat message using OpenAI role names.
type Message struct {
	Role    string
	Content string
}

type stagedTurn struct {
	history []Message
	user    Message
	trimmed int
}

// Conversation is in-memory chat state. It is not persisted.
type Conversation struct {
	budget  int64
	reserve int64
	system  *Message
	history []Message
	staged  *stagedTurn
}

func New(budget, reserve int64, systemPrompt string) *Conversation {
	c := &Conversation{budget: budget, reserve: reserve}
	if systemPrompt != "" {
		c.system = &Message{Role: "system", Content: systemPrompt}
	}
	return c
}

// Stage prepares a request for userContent. Committed history is unchanged until Commit.
func (c *Conversation) Stage(userContent string) (messages []Message, trimmedPairs int, err error) {
	if c.staged != nil {
		return nil, 0, fmt.Errorf("a turn is already staged; commit or abort it first")
	}

	user := Message{Role: "user", Content: userContent}
	protected := c.prefix(nil, user)
	ok, err := Fits(protected, c.budget, c.reserve)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, fmt.Errorf("message does not fit in context_budget with the system prompt and generation_reserve; shorten the input, reduce generation_reserve, or increase context_budget")
	}

	history := clone(c.history)
	trimmed := 0
	for {
		candidate := c.prefix(history, user)
		ok, err := Fits(candidate, c.budget, c.reserve)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			c.staged = &stagedTurn{history: history, user: user, trimmed: trimmed}
			return candidate, trimmed, nil
		}
		if len(history) < 2 {
			return nil, 0, fmt.Errorf("message does not fit in context_budget with the system prompt and generation_reserve; shorten the input, reduce generation_reserve, or increase context_budget")
		}
		history = clone(history[2:])
		trimmed++
	}
}

// Commit records the staged user message and the complete assistant response.
func (c *Conversation) Commit(assistantContent string) error {
	if c.staged == nil {
		return fmt.Errorf("no staged turn to commit")
	}
	c.history = append(clone(c.staged.history), c.staged.user, Message{Role: "assistant", Content: assistantContent})
	c.staged = nil
	return nil
}

// Abort discards a staged turn and leaves committed history unchanged.
func (c *Conversation) Abort() {
	c.staged = nil
}

// Clear removes committed and staged turns. The configured system prompt is kept.
func (c *Conversation) Clear() {
	c.history = nil
	c.staged = nil
}

func (c *Conversation) History() []Message {
	return clone(c.history)
}

func (c *Conversation) HasStaged() bool {
	return c.staged != nil
}

func (c *Conversation) StreamLimit() int64 {
	return c.budget
}

func (c *Conversation) Reserve() int64 {
	return c.reserve
}

func (c *Conversation) prefix(history []Message, user Message) []Message {
	n := len(history) + 1
	if c.system != nil {
		n++
	}
	out := make([]Message, 0, n)
	if c.system != nil {
		out = append(out, *c.system)
	}
	out = append(out, history...)
	out = append(out, user)
	return out
}

func clone(in []Message) []Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]Message, len(in))
	copy(out, in)
	return out
}

// Estimate returns application budget units for messages.
func Estimate(messages []Message) (int64, error) {
	total := int64(PerRequest)
	for _, m := range messages {
		n, err := addUnits(int64(len(m.Content)), PerMessage)
		if err != nil {
			return 0, err
		}
		total, err = addUnits(total, n)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

// Fits reports whether estimated prompt units plus reserve are within budget.
func Fits(messages []Message, budget, reserve int64) (bool, error) {
	if budget <= 0 || reserve <= 0 {
		return false, fmt.Errorf("context_budget and generation_reserve must be positive")
	}
	est, err := Estimate(messages)
	if err != nil {
		return false, err
	}
	sum, err := addUnits(est, reserve)
	if err != nil {
		return false, err
	}
	return sum <= budget, nil
}

func addUnits(a, b int64) (int64, error) {
	if a < 0 || b < 0 {
		return 0, fmt.Errorf("context accounting overflow")
	}
	if a > math.MaxInt64-b {
		return 0, fmt.Errorf("context accounting overflow")
	}
	return a + b, nil
}
