package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
)

// defaultTelegramBaseURL is the public Bot API host. The path is built per-call
// as <baseURL>/bot<token>/sendMessage. It is a field on TelegramNotifier (not a
// const) so tests can repoint it at an httptest.Server.
const defaultTelegramBaseURL = "https://api.telegram.org"

// TelegramNotifier delivers a Message as an HTML-formatted Telegram message via
// the Bot API sendMessage method. It is provider-agnostic only in shape — the
// transport is Telegram-specific.
type TelegramNotifier struct {
	token   string
	chatID  string
	client  *http.Client
	baseURL string
}

// NewTelegramNotifier builds a Telegram adapter for the given bot token + chat.
// A nil client falls back to http.DefaultClient. baseURL defaults to the public
// API host; override the BaseURL field in tests to target a stub server.
func NewTelegramNotifier(token, chatID string, client *http.Client) *TelegramNotifier {
	if client == nil {
		client = http.DefaultClient
	}
	return &TelegramNotifier{
		token:   token,
		chatID:  chatID,
		client:  client,
		baseURL: defaultTelegramBaseURL,
	}
}

// telegramSendRequest is the JSON body for sendMessage.
type telegramSendRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// Notify formats the Message as HTML and POSTs it to sendMessage. A non-2xx
// response is an error (the body is included for diagnostics).
func (t *TelegramNotifier) Notify(ctx context.Context, msg Message) error {
	body, err := json.Marshal(telegramSendRequest{
		ChatID:    t.chatID,
		Text:      formatTelegramHTML(msg),
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal body: %w", err)
	}

	url := strings.TrimRight(t.baseURL, "/") + "/bot" + t.token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

// formatTelegramHTML renders the Message as Telegram-flavoured HTML: a bold
// title, the body, then one line per field ("<b>key</b>: value"). Every dynamic
// value is HTML-escaped so user-supplied content cannot inject markup.
func formatTelegramHTML(msg Message) string {
	var b strings.Builder
	if msg.Title != "" {
		b.WriteString("<b>")
		b.WriteString(html.EscapeString(msg.Title))
		b.WriteString("</b>")
	}
	if msg.Body != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(html.EscapeString(msg.Body))
	}
	for k, v := range msg.Fields {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("<b>")
		b.WriteString(html.EscapeString(k))
		b.WriteString("</b>: ")
		b.WriteString(html.EscapeString(v))
	}
	return b.String()
}
