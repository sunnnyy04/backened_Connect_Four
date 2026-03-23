package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message from/to the client
type Message struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// GameState represents the current state of a game
type GameState struct {
	Board  [][]int `json:"board"`
	Turn   int     `json:"turn"`
	Winner int     `json:"winner"`
}

// Player represents a player in a game
type Player struct {
	Username string
	Conn     *websocket.Conn
	PlayerID int
	GameID   string
	IsBot    bool
	mu       sync.Mutex
}

// Game represents a Connect 4 game session
type Game struct {
	ID       string
	Player1  *Player
	Player2  *Player
	Board    [][]int
	Turn     int
	Winner   int
	GameOver bool
	mu       sync.RWMutex
}

// LeaderboardEntry represents a player's leaderboard stats
type LeaderboardEntry struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
}

// GlobalState holds all game sessions and player stats
type GlobalState struct {
	Games   map[string]*Game
	Players map[string]*LeaderboardEntry
	Waiting *Player
	mu      sync.RWMutex
}

const (
	ROWS = 6
	COLS = 7
)

// NewGame creates a new game instance
func NewGame(id string, p1, p2 *Player) *Game {
	board := make([][]int, ROWS)
	for i := range board {
		board[i] = make([]int, COLS)
	}

	return &Game{
		ID:      id,
		Player1: p1,
		Player2: p2,
		Board:   board,
		Turn:    1,
		Winner:  0,
	}
}

// NewPlayer creates a new player instance
func NewPlayer(username string, conn *websocket.Conn, playerID int) *Player {
	return &Player{
		Username: username,
		Conn:     conn,
		PlayerID: playerID,
		IsBot:    false,
	}
}

// NewBotPlayer creates a new bot player instance
func NewBotPlayer(playerID int) *Player {
	return &Player{
		Username: "Bot",
		Conn:     nil,
		PlayerID: playerID,
		IsBot:    true,
	}
}

// SendUpdate sends a game state update to a player
func (p *Player) SendUpdate(state *GameState) error {
	// Bots don't receive messages (no connection)
	if p.IsBot {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Conn == nil {
		return nil
	}

	msg := map[string]interface{}{
		"type":   "update",
		"board":  state.Board,
		"turn":   state.Turn,
		"winner": state.Winner,
	}

	return p.Conn.WriteJSON(msg)
}

// SendMessage sends a JSON message to a player
func (p *Player) SendMessage(msg map[string]interface{}) error {
	// Bots don't receive messages (no connection)
	if p.IsBot || p.Conn == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Conn.WriteJSON(msg)
}

// ReadMessage reads a JSON message from a player
func (p *Player) ReadMessage() (*Message, error) {
	var msg Message
	err := p.Conn.ReadJSON(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetState returns the current game state
func (g *Game) GetState() *GameState {
	g.mu.RLock()
	defer g.mu.RUnlock()

	board := make([][]int, ROWS)
	for i := range board {
		board[i] = make([]int, COLS)
		copy(board[i], g.Board[i])
	}

	return &GameState{
		Board:  board,
		Turn:   g.Turn,
		Winner: g.Winner,
	}
}

// SafeClose safely closes a player connection
func (p *Player) SafeClose() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Conn != nil {
		p.Conn.Close()
	}
}
