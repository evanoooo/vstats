package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	InitialReconnectDelay = 5 * time.Second
	MaxReconnectDelay     = 60 * time.Second
	AuthTimeout           = 10 * time.Second
	PingInterval          = 30 * time.Second
)

type WebSocketClient struct {
	config    *AgentConfig
	collector *MetricsCollector
}

func NewWebSocketClient(config *AgentConfig) *WebSocketClient {
	return &WebSocketClient{
		config:    config,
		collector: NewMetricsCollector(),
	}
}

func (wsc *WebSocketClient) Run() {
	reconnectDelay := InitialReconnectDelay

	for {
		log.Printf("Connecting to %s...", wsc.config.WSUrl())

		if err := wsc.connectAndRun(); err != nil {
			log.Printf("Connection error: %v", err)
		} else {
			log.Println("Connection closed normally")
			reconnectDelay = InitialReconnectDelay
		}

		log.Printf("Reconnecting in %v...", reconnectDelay)
		time.Sleep(reconnectDelay)

		// Exponential backoff
		reconnectDelay *= 2
		if reconnectDelay > MaxReconnectDelay {
			reconnectDelay = MaxReconnectDelay
		}
	}
}

func (wsc *WebSocketClient) connectAndRun() error {
	wsURL := wsc.config.WSUrl()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	log.Println("Connected to WebSocket server")

	// Send authentication message
	authMsg := AuthMessage{
		Type:     "auth",
		ServerID: wsc.config.ServerID,
		Token:    wsc.config.AgentToken,
		Version:  AgentVersion,
	}

	authData, err := json.Marshal(authMsg)
	if err != nil {
		return fmt.Errorf("failed to serialize auth message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, authData); err != nil {
		return fmt.Errorf("failed to send auth message: %w", err)
	}

	log.Println("Sent authentication message")

	// Wait for auth response
	conn.SetReadDeadline(time.Now().Add(AuthTimeout))
	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to receive auth response: %w", err)
	}

	var response ServerResponse
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	if response.Status != "ok" {
		return fmt.Errorf("authentication failed: %s", response.Message)
	}

	// Update ping targets from server config if provided
	if len(response.PingTargets) > 0 {
		log.Printf("Received %d ping targets from server", len(response.PingTargets))
		wsc.collector.SetPingTargets(response.PingTargets)
	}

	log.Println("Authentication successful!")

	// Reset read deadline
	conn.SetReadDeadline(time.Time{})

	// Start metrics sending loop
	metricsTicker := time.NewTicker(time.Duration(wsc.config.IntervalSecs) * time.Second)
	defer metricsTicker.Stop()

	pingTicker := time.NewTicker(PingInterval)
	defer pingTicker.Stop()

	// Handle incoming messages
	done := make(chan error, 1)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}

			var response ServerResponse
			if err := json.Unmarshal(message, &response); err != nil {
				continue
			}

			switch response.Type {
			case "error":
				log.Printf("Server error: %s", response.Message)
			case "command":
				if response.Command == "update" {
					log.Println("Received update command from server")
					wsc.handleUpdateCommand(response.DownloadURL)
				}
			}
		}
	}()

	for {
		select {
		case <-metricsTicker.C:
			metrics := wsc.collector.Collect()
			msg := MetricsMessage{
				Type:    "metrics",
				Metrics: metrics,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to serialize metrics: %v", err)
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return fmt.Errorf("failed to send metrics: %w", err)
			}

		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("failed to send ping: %w", err)
			}

		case err := <-done:
			return err
		}
	}
}

func (wsc *WebSocketClient) handleUpdateCommand(downloadURL string) {
	log.Println("Starting self-update process...")
	// Update implementation would go here
	// For now, just log
	log.Printf("Update command received with URL: %s", downloadURL)
}

