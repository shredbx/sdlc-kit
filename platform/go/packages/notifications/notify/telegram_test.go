package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTelegramAdapter_BuildsSendMessage spins a stub Bot API, points the adapter
// at it, and asserts the request shape: POST, path carries the bot token +
// /sendMessage, and the JSON body carries chat_id + the message text.
func TestTelegramAdapter_BuildsSendMessage(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   telegramSendRequest
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tn := NewTelegramNotifier("BOT123:secret", "chat-99", srv.Client())
	tn.baseURL = srv.URL

	msg := Message{
		Title:  "New lead",
		Body:   "Someone wants to sell",
		Fields: map[string]string{"Email": "a@b.com"},
	}
	if err := tn.Notify(context.Background(), msg); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotPath, "/botBOT123:secret/sendMessage") {
		t.Errorf("path = %q, want it to contain /bot<token>/sendMessage", gotPath)
	}
	if gotBody.ChatID != "chat-99" {
		t.Errorf("chat_id = %q, want chat-99", gotBody.ChatID)
	}
	if gotBody.ParseMode != "HTML" {
		t.Errorf("parse_mode = %q, want HTML", gotBody.ParseMode)
	}
	if !strings.Contains(gotBody.Text, "New lead") || !strings.Contains(gotBody.Text, "Someone wants to sell") {
		t.Errorf("text = %q, want it to carry the title + body", gotBody.Text)
	}
	if !strings.Contains(gotBody.Text, "a@b.com") {
		t.Errorf("text = %q, want it to carry the field value", gotBody.Text)
	}
}

// TestTelegramAdapter_Non2xxIsError asserts a non-2xx Bot API response surfaces
// as an error (so a MultiNotifier can log + continue).
func TestTelegramAdapter_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"bad chat_id"}`))
	}))
	defer srv.Close()

	tn := NewTelegramNotifier("tok", "chat", srv.Client())
	tn.baseURL = srv.URL

	err := tn.Notify(context.Background(), Message{Title: "x"})
	if err == nil {
		t.Fatal("expected an error on non-2xx response, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %v, want it to mention the 400 status", err)
	}
}

// TestNoopNotifier_AlwaysOK is the null-channel contract.
func TestNoopNotifier_AlwaysOK(t *testing.T) {
	if err := (NoopNotifier{}).Notify(context.Background(), Message{Title: "x"}); err != nil {
		t.Errorf("NoopNotifier.Notify = %v, want nil", err)
	}
}

// stubNotifier records calls and returns a fixed error.
type stubNotifier struct {
	called int
	err    error
}

func (s *stubNotifier) Notify(_ context.Context, _ Message) error {
	s.called++
	return s.err
}

// TestMultiNotifier_BestEffort asserts fan-out reaches EVERY target even when an
// earlier one fails, and returns the first error.
func TestMultiNotifier_BestEffort(t *testing.T) {
	failErr := errTest("boom")
	a := &stubNotifier{err: failErr}
	b := &stubNotifier{}
	m := NewMultiNotifier(nil, a, nil, b) // nil entry must be skipped, not panic

	err := m.Notify(context.Background(), Message{Title: "x"})
	if err == nil {
		t.Fatal("expected first error to propagate, got nil")
	}
	if a.called != 1 || b.called != 1 {
		t.Errorf("both targets must be called once: a=%d b=%d", a.called, b.called)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
