package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type WSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (a *Agent) connectWithBackoff(ctx context.Context) {
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := a.connect(ctx)
		if err != nil {
			log.Printf("WebSocket connect failed: %v", err)
		} else {
			backoff = 1 * time.Second // reset on success
			a.runLoop(ctx, conn)
		}

		// Jitter: ±20% of backoff
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (a *Agent) connect(ctx context.Context) (*WSConn, error) {
	u := url.URL{
		Scheme: "wss",
		Host:   extractHost(a.config.WorkerURL),
		Path:   "/agent/connect",
	}

	opts := &websocket.DialOptions{
		CompressionMode: websocket.CompressionContextTakeover,
		HTTPHeader: map[string][]string{
			"User-Agent":    {"AetherRuin-Agent/1.0"},
			"Authorization": {"Bearer " + a.jwt},
		},
	}

	ws, resp, err := websocket.Dial(ctx, u.String(), opts)
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			identity, identityErr := FetchInstanceIdentity(ctx)
			if identityErr != nil {
				return nil, identityErr
			}

			jwt, registerErr := Register(ctx, a.config, identity)
			if registerErr != nil {
				return nil, registerErr
			}

			a.jwt = jwt
			a.config.AgentJWT = jwt
			if saveErr := SaveConfig(configPath, a.config); saveErr != nil {
				log.Printf("Warning: failed to persist refreshed agent JWT: %v", saveErr)
			}

			opts.HTTPHeader["Authorization"] = []string{"Bearer " + a.jwt}
			ws, _, err = websocket.Dial(ctx, u.String(), opts)
		}
	}
	if err != nil {
		return nil, err
	}

	conn := &WSConn{conn: ws}
	a.setConn(conn)

	// Send initial health check
	a.sendHealth()

	log.Println("WebSocket connected")
	return conn, nil
}

func (a *Agent) runLoop(ctx context.Context, conn *WSConn) {
	defer func() {
		conn.Close()
		a.setConn(nil)
		log.Println("WebSocket disconnected")
	}()

	// Ping every 25 seconds
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(ctx); err != nil {
					return
				}
			}
		}
	}()

	// Read messages
	for {
		if ctx.Err() != nil {
			return
		}

		_, msg, err := conn.Read(ctx)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		a.handleCommand(msg)
	}
}

func (c *WSConn) Send(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *WSConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.conn.Read(ctx)
}

func (c *WSConn) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

func (c *WSConn) Close() {
	c.conn.Close(websocket.StatusNormalClosure, "")
}

func extractHost(workerURL string) string {
	u, err := url.Parse(workerURL)
	if err != nil {
		return workerURL
	}
	return u.Host
}
