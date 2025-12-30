package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// ============================================================================
// Notifier Interface
// ============================================================================

// Notifier interface for all notification channels
type Notifier interface {
	Send(title, message string) error
	Type() string
	Validate() error
}

// CreateNotifier creates a notifier from channel configuration
func CreateNotifier(channel NotificationChannel) (Notifier, error) {
	switch channel.Type {
	case "email":
		return NewEmailNotifier(channel.Config)
	case "telegram":
		return NewTelegramNotifier(channel.Config)
	case "discord":
		return NewDiscordNotifier(channel.Config)
	case "webhook":
		return NewWebhookNotifier(channel.Config)
	case "bark":
		return NewBarkNotifier(channel.Config)
	case "serverchan":
		return NewServerChanNotifier(channel.Config)
	default:
		return nil, fmt.Errorf("unknown notification channel type: %s", channel.Type)
	}
}

// ============================================================================
// Email Notifier
// ============================================================================

type EmailNotifier struct {
	SMTPHost     string
	SMTPPort     string
	Username     string
	Password     string
	From         string
	To           []string
	UseTLS       bool
	SkipVerify   bool
}

func NewEmailNotifier(config map[string]string) (*EmailNotifier, error) {
	n := &EmailNotifier{
		SMTPHost:   config["smtp_host"],
		SMTPPort:   config["smtp_port"],
		Username:   config["username"],
		Password:   config["password"],
		From:       config["from"],
		UseTLS:     config["use_tls"] == "true",
		SkipVerify: config["skip_verify"] == "true",
	}
	if to := config["to"]; to != "" {
		n.To = strings.Split(to, ",")
		for i := range n.To {
			n.To[i] = strings.TrimSpace(n.To[i])
		}
	}
	return n, nil
}

func (e *EmailNotifier) Type() string { return "email" }

func (e *EmailNotifier) Validate() error {
	if e.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if e.SMTPPort == "" {
		e.SMTPPort = "587"
	}
	if e.From == "" {
		return fmt.Errorf("from address is required")
	}
	if len(e.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	return nil
}

func (e *EmailNotifier) Send(title, message string) error {
	if err := e.Validate(); err != nil {
		return err
	}

	addr := e.SMTPHost + ":" + e.SMTPPort
	
	// Build email
	header := make(map[string]string)
	header["From"] = e.From
	header["To"] = strings.Join(e.To, ",")
	header["Subject"] = title
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=UTF-8"

	body := ""
	for k, v := range header {
		body += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	body += "\r\n" + message

	var auth smtp.Auth
	if e.Username != "" && e.Password != "" {
		auth = smtp.PlainAuth("", e.Username, e.Password, e.SMTPHost)
	}

	if e.UseTLS {
		// TLS connection
		tlsConfig := &tls.Config{
			InsecureSkipVerify: e.SkipVerify,
			ServerName:         e.SMTPHost,
		}
		
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS dial failed: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.SMTPHost)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %v", err)
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %v", err)
			}
		}

		if err := client.Mail(e.From); err != nil {
			return fmt.Errorf("MAIL FROM failed: %v", err)
		}
		for _, rcpt := range e.To {
			if err := client.Rcpt(rcpt); err != nil {
				return fmt.Errorf("RCPT TO failed: %v", err)
			}
		}
		
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("DATA failed: %v", err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return fmt.Errorf("write failed: %v", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("close failed: %v", err)
		}
		return client.Quit()
	}

	// Standard connection
	return smtp.SendMail(addr, auth, e.From, e.To, []byte(body))
}

// ============================================================================
// Telegram Notifier
// ============================================================================

type TelegramNotifier struct {
	BotToken        string
	ChatID          string
	MessageThreadID string
	Silent          bool
}

func NewTelegramNotifier(config map[string]string) (*TelegramNotifier, error) {
	return &TelegramNotifier{
		BotToken:        config["bot_token"],
		ChatID:          config["chat_id"],
		MessageThreadID: config["message_thread_id"],
		Silent:          config["silent"] == "true",
	}, nil
}

func (t *TelegramNotifier) Type() string { return "telegram" }

func (t *TelegramNotifier) Validate() error {
	if t.BotToken == "" {
		return fmt.Errorf("bot_token is required")
	}
	if t.ChatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	return nil
}

func (t *TelegramNotifier) Send(title, message string) error {
	if err := t.Validate(); err != nil {
		return err
	}

	text := fmt.Sprintf("*%s*\n\n%s", escapeMarkdownV2(title), escapeMarkdownV2(message))
	
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	
	payload := map[string]interface{}{
		"chat_id":    t.ChatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}
	if t.MessageThreadID != "" {
		payload["message_thread_id"] = t.MessageThreadID
	}
	if t.Silent {
		payload["disable_notification"] = true
	}

	return postJSON(apiURL, payload)
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2
func escapeMarkdownV2(s string) string {
	chars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	result := s
	for _, c := range chars {
		result = strings.ReplaceAll(result, c, "\\"+c)
	}
	return result
}

// ============================================================================
// Discord Notifier
// ============================================================================

type DiscordNotifier struct {
	WebhookURL string
	Username   string
	AvatarURL  string
}

func NewDiscordNotifier(config map[string]string) (*DiscordNotifier, error) {
	return &DiscordNotifier{
		WebhookURL: config["webhook_url"],
		Username:   config["username"],
		AvatarURL:  config["avatar_url"],
	}, nil
}

func (d *DiscordNotifier) Type() string { return "discord" }

func (d *DiscordNotifier) Validate() error {
	if d.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}

func (d *DiscordNotifier) Send(title, message string) error {
	if err := d.Validate(); err != nil {
		return err
	}

	embed := map[string]interface{}{
		"title":       title,
		"description": message,
		"color":       16711680, // Red
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}
	if d.Username != "" {
		payload["username"] = d.Username
	}
	if d.AvatarURL != "" {
		payload["avatar_url"] = d.AvatarURL
	}

	return postJSON(d.WebhookURL, payload)
}

// ============================================================================
// Webhook Notifier (Generic)
// ============================================================================

type WebhookNotifier struct {
	URL         string
	Method      string
	Headers     map[string]string
	BodyFormat  string // json, form
}

func NewWebhookNotifier(config map[string]string) (*WebhookNotifier, error) {
	n := &WebhookNotifier{
		URL:        config["url"],
		Method:     strings.ToUpper(config["method"]),
		BodyFormat: config["body_format"],
		Headers:    make(map[string]string),
	}
	if n.Method == "" {
		n.Method = "POST"
	}
	if n.BodyFormat == "" {
		n.BodyFormat = "json"
	}
	
	// Parse headers from "header_*" config keys
	for k, v := range config {
		if strings.HasPrefix(k, "header_") {
			headerName := strings.TrimPrefix(k, "header_")
			n.Headers[headerName] = v
		}
	}
	
	return n, nil
}

func (w *WebhookNotifier) Type() string { return "webhook" }

func (w *WebhookNotifier) Validate() error {
	if w.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (w *WebhookNotifier) Send(title, message string) error {
	if err := w.Validate(); err != nil {
		return err
	}

	var body io.Reader
	contentType := "application/json"

	payload := map[string]string{
		"title":   title,
		"message": message,
		"time":    time.Now().Format(time.RFC3339),
	}

	if w.BodyFormat == "form" {
		form := url.Values{}
		for k, v := range payload {
			form.Add(k, v)
		}
		body = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	} else {
		jsonData, _ := json.Marshal(payload)
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(w.Method, w.URL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ============================================================================
// Bark Notifier (iOS Push)
// ============================================================================

type BarkNotifier struct {
	ServerURL string
	DeviceKey string
	Sound     string
	Group     string
	Icon      string
}

func NewBarkNotifier(config map[string]string) (*BarkNotifier, error) {
	serverURL := config["server_url"]
	if serverURL == "" {
		serverURL = "https://api.day.app"
	}
	return &BarkNotifier{
		ServerURL: strings.TrimSuffix(serverURL, "/"),
		DeviceKey: config["device_key"],
		Sound:     config["sound"],
		Group:     config["group"],
		Icon:      config["icon"],
	}, nil
}

func (b *BarkNotifier) Type() string { return "bark" }

func (b *BarkNotifier) Validate() error {
	if b.DeviceKey == "" {
		return fmt.Errorf("device_key is required")
	}
	return nil
}

func (b *BarkNotifier) Send(title, message string) error {
	if err := b.Validate(); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"title": title,
		"body":  message,
	}
	if b.Sound != "" {
		payload["sound"] = b.Sound
	}
	if b.Group != "" {
		payload["group"] = b.Group
	}
	if b.Icon != "" {
		payload["icon"] = b.Icon
	}

	apiURL := fmt.Sprintf("%s/%s", b.ServerURL, b.DeviceKey)
	return postJSON(apiURL, payload)
}

// ============================================================================
// ServerChan Notifier (WeChat)
// ============================================================================

type ServerChanNotifier struct {
	SendKey string
	Channel string // 9: 微信测试号, 1: 企业微信, 0: 方糖
}

func NewServerChanNotifier(config map[string]string) (*ServerChanNotifier, error) {
	return &ServerChanNotifier{
		SendKey: config["send_key"],
		Channel: config["channel"],
	}, nil
}

func (s *ServerChanNotifier) Type() string { return "serverchan" }

func (s *ServerChanNotifier) Validate() error {
	if s.SendKey == "" {
		return fmt.Errorf("send_key is required")
	}
	return nil
}

func (s *ServerChanNotifier) Send(title, message string) error {
	if err := s.Validate(); err != nil {
		return err
	}

	// ServerChan Turbo API
	apiURL := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", s.SendKey)
	
	payload := map[string]interface{}{
		"title": title,
		"desp":  message,
	}
	if s.Channel != "" {
		payload["channel"] = s.Channel
	}

	return postJSON(apiURL, payload)
}

// ============================================================================
// Helper Functions
// ============================================================================

// postJSON sends a JSON POST request
func postJSON(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}


