package notify

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"
)

// SocketServer handles communication with TUI clients.
type SocketServer struct {
	path     string
	listener net.Listener
	clients  map[net.Conn]bool
	mu       sync.RWMutex
	done     chan struct{}
}

// NewSocketServer creates a new Unix socket server.
func NewSocketServer(path string) *SocketServer {
	return &SocketServer{
		path:    path,
		clients: make(map[net.Conn]bool),
		done:    make(chan struct{}),
	}
}

// Start begins listening for connections.
func (s *SocketServer) Start() error {
	// Remove existing socket file
	os.Remove(s.path)

	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.listener = listener

	// Set permissions so user can connect
	os.Chmod(s.path, 0700)

	go s.acceptLoop()
	return nil
}

// Stop shuts down the server.
func (s *SocketServer) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[net.Conn]bool)
	s.mu.Unlock()

	os.Remove(s.path)
}

// Broadcast sends a message to all connected clients.
func (s *SocketServer) Broadcast(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	data = append(data, '\n')

	s.mu.RLock()
	defer s.mu.RUnlock()

	for conn := range s.clients {
		conn.Write(data)
	}
}

// ClientCount returns the number of connected clients.
func (s *SocketServer) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

func (s *SocketServer) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Printf("[socket] Accept error: %v", err)
				continue
			}
		}

		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()

		log.Printf("[socket] Client connected (%d total)", s.ClientCount())

		go s.handleClient(conn)
	}
}

func (s *SocketServer) handleClient(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
		log.Printf("[socket] Client disconnected (%d total)", s.ClientCount())
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Handle incoming messages from TUI (e.g., acknowledge)
		line := scanner.Text()
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Could handle ACK messages here if needed
		if msgType, ok := msg["type"].(string); ok {
			log.Printf("[socket] Received: %s", msgType)
		}
	}
}

// SocketClient connects to the insight server.
type SocketClient struct {
	conn      net.Conn
	connected bool
	onMessage func(map[string]any)
	mu        sync.Mutex
}

// NewSocketClient creates a new socket client.
func NewSocketClient() *SocketClient {
	return &SocketClient{}
}

// Connect connects to the socket server.
func (c *SocketClient) Connect(path string) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	c.conn = conn
	c.connected = true

	go c.readLoop()
	return nil
}

// Close closes the connection.
func (c *SocketClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.connected = false
	}
}

// IsConnected returns whether the client is connected.
func (c *SocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Send sends a message to the server.
func (c *SocketClient) Send(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.conn.Write(data)
	return err
}

// OnMessage sets the callback for incoming messages.
func (c *SocketClient) OnMessage(callback func(map[string]any)) {
	c.onMessage = callback
}

func (c *SocketClient) readLoop() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if c.onMessage != nil {
			c.onMessage(msg)
		}
	}

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
}
