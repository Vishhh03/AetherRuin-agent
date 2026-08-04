package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	Version    = "1.0.0"
	configPath = "/etc/aetherruin-agent/config.json"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("AetherRuin Host Agent v%s starting", Version)

	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := validateWorkerURL(config.WorkerURL); err != nil {
		log.Fatalf("Unsafe worker URL in config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	identity, err := FetchInstanceIdentity(ctx)
	if err != nil {
		log.Printf("Warning: could not fetch instance identity: %v", err)
	}

	jwt := config.AgentJWT
	if jwt == "" {
		jwt, err = Register(ctx, config, identity)
		if err != nil {
			log.Fatalf("Registration failed: %v", err)
		}
		config.AgentJWT = jwt
		if err := SaveConfig(configPath, config); err != nil {
			log.Printf("Warning: failed to persist agent JWT: %v", err)
		}
	}
	log.Println("Registered successfully")

	engine := NewEngine(config)

	agent := &Agent{
		config: config,
		jwt:    jwt,
		engine: engine,
	}

	logFilter := NewLogFilter()

	// Wire engine callbacks to agent
	engine.SetCallbacks(
		func(lines []string) {
			filtered := logFilter.Filter(lines)
			if len(filtered) == 0 {
				return
			}
			msg := map[string]any{
				"type":      "log",
				"lines":     filtered,
				"timestamp": 0, // Will be set by DO
			}
			agent.send(msg)
		},
		func() {
			log.Println("Server is ready, sending running status")
			agent.sendStatus("running")
		},
		func() {
			agent.sendStatus("stopped")
		},
	)

	// Start stats loop
	go agent.StartStatsLoop(ctx)

	// Start self-update loop (hourly check)
	go agent.StartUpdateLoop(ctx)

	// Start WebSocket with reconnect
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		agent.connectWithBackoff(ctx)
	}()

	sig := <-sigCh
	log.Printf("Received signal %v, shutting down", sig)
	cancel()

	if engine.IsRunning() {
		engine.Stop(30)
	}

	wg.Wait()
	log.Println("Agent stopped")
}

type Agent struct {
	config *Config
	jwt    string
	engine GameEngine
	conn   *WSConn
	mu     sync.Mutex
}

type CommandMessage struct {
	Type               string  `json:"type"`
	GracePeriodSeconds int     `json:"gracePeriodSeconds,omitempty"`
	Command            string  `json:"command,omitempty"`
	CommandID          string  `json:"commandId,omitempty"`
	ServerProperties   *string `json:"serverProperties,omitempty"`
}

func (a *Agent) handleCommand(msg []byte) {
	var cmd CommandMessage
	if err := json.Unmarshal(msg, &cmd); err != nil {
		log.Printf("Failed to parse command: %v", err)
		return
	}

	switch cmd.Type {
	case "start-server":
		if cmd.ServerProperties != nil {
			err := os.WriteFile("server.properties", []byte(*cmd.ServerProperties), 0644)
			if err != nil {
				log.Printf("Failed to write server.properties: %v", err)
			} else if err := chownToGameUser("server.properties"); err != nil {
				log.Printf("Warning: failed to chown server.properties to %s: %v", gameSystemUser, err)
			}
		}
		err := a.engine.Start()
		if err != nil {
			log.Printf("Failed to start server engine: %v", err)
			return
		}
		// Don't send "running" here — wait for onReady callback
		a.sendStatus("starting")

	case "stop-server":
		grace := cmd.GracePeriodSeconds
		if grace == 0 {
			grace = 60
		}
		a.sendStatus("stopping")
		a.engine.Stop(grace)
		a.sendStatus("stopped")

	case "restart-server":
		a.engine.Stop(30)
		if cmd.ServerProperties != nil {
			err := os.WriteFile("server.properties", []byte(*cmd.ServerProperties), 0644)
			if err != nil {
				log.Printf("Failed to write server.properties: %v", err)
			} else if err := chownToGameUser("server.properties"); err != nil {
				log.Printf("Warning: failed to chown server.properties to %s: %v", gameSystemUser, err)
			}
		}
		a.engine.Start()
		a.sendStatus("starting")

	case "console-command":
		if isBlockedConsoleCommand(cmd.Command) {
			a.sendResult(cmd.CommandID, false, "command is blocked")
			return
		}
		_, err := a.engine.SendCommand(cmd.Command)
		ok := err == nil
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		a.sendResult(cmd.CommandID, ok, errMsg)

	case "ping":
		a.sendPong()
	}
}

func (a *Agent) sendStatus(status string) {
	ip, _ := GetPublicIP()
	msg := map[string]any{
		"type":     "status",
		"status":   status,
		"publicIp": ip,
	}
	a.send(msg)
}

func (a *Agent) sendResult(commandID string, ok bool, errMsg string) {
	msg := map[string]any{
		"type":      "result",
		"commandId": commandID,
		"ok":        ok,
		"error":     errMsg,
	}
	a.send(msg)
}

func (a *Agent) sendPong() {
	a.send(map[string]string{"type": "pong"})
}

func (a *Agent) sendHealth() {
	msg := map[string]any{
		"type":          "health",
		"version":       Version,
		"uptimeSeconds": GetUptimeSeconds(),
	}
	a.send(msg)
}

func (a *Agent) send(msg any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		a.conn.Send(msg)
	}
}

func (a *Agent) setConn(conn *WSConn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conn = conn
}
