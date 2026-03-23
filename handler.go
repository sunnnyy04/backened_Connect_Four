package main

import (
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// HandleWebSocket handles a new WebSocket connection
func (g *GlobalState) HandleWebSocket(ws *websocket.Conn) {
	var player *Player
	var game *Game

	defer func() {
		if player != nil {
			player.SafeClose()
		}
		if game != nil {
			g.mu.Lock()
			delete(g.Games, game.ID)
			g.mu.Unlock()
		}
	}()

	// Wait for join message
	msg, err := readMessage(ws)
	if err != nil {
		log.Printf("Failed to read join message: %v", err)
		return
	}

	if msg.Type != "join" {
		log.Printf("Expected join message, got: %s", msg.Type)
		return
	}

	username := msg.Username
	if username == "" {
		username = "Player"
	}

	// Create player
	g.mu.Lock()

	// Check if player exists in leaderboard
	entry, exists := g.Players[username]
	if !exists {
		g.Players[username] = &LeaderboardEntry{
			Username: username,
			Wins:     0,
		}
	} else {
		log.Printf("Player %s rejoined, current wins: %d", username, entry.Wins)
	}

	// Try to match with waiting player
	if g.Waiting != nil {
		// Found a match - create game with exactly 2 players
		player1 := g.Waiting
		player2 := NewPlayer(username, ws, 2)

		gameID := player1.Username + "_" + player2.Username + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
		game = NewGame(gameID, player1, player2)

		g.Games[gameID] = game
		g.Waiting = nil // Critical: clear waiting state to prevent 3+ player matching
		g.mu.Unlock()

		player = player2
		game.Player1.GameID = gameID
		game.Player2.GameID = gameID

		// Notify both players
		state := game.GetState()
		player1.SendUpdate(state)
		player2.SendUpdate(state)

		log.Printf("Game started: %s (Player1: %s vs Player2: %s)", gameID, player1.Username, player2.Username)

		// Handle players in parallel
		go g.handlePlayerMessages(game, player1, 1)
		g.handlePlayerMessages(game, player2, 2)

	} else {
		// No opponent waiting - this player becomes the waiting player
		player = NewPlayer(username, ws, 1)
		g.Waiting = player
		g.mu.Unlock()

		// Send waiting message
		player.SendMessage(map[string]interface{}{
			"type":    "waiting",
			"message": "Waiting for an opponent...",
		})

		log.Printf("Player %s waiting for opponent", username)

		// Set a timeout to match with bot if no opponent joins
		botTimeout := time.NewTimer(10 * time.Second)
		botMatched := false

		// Listen for incoming messages or timeout
		go func() {
			<-botTimeout.C

			// Check if still waiting
			g.mu.Lock()
			if g.Waiting == player && !botMatched {
				// Create bot opponent
				bot := NewBotPlayer(2)
				gameID := player.Username + "_Bot_" + strconv.FormatInt(time.Now().UnixNano(), 10)
				game = NewGame(gameID, player, bot)

				g.Games[gameID] = game
				g.Waiting = nil
				botMatched = true
				g.mu.Unlock()

				player.GameID = gameID
				bot.GameID = gameID

				// Notify human player about bot match
				state := game.GetState()
				player.SendUpdate(state)

				log.Printf("Game started: %s (Player1: %s vs Player2: Bot)", gameID, player.Username)

				// Handle messages from human, bot moves automatically
				go g.handlePlayerMessages(game, player, 1)
				go g.handleBotMoves(game, bot, 2)
			} else {
				g.mu.Unlock()
			}
		}()

		// Keep waiting player alive
		for {
			_, err := readMessage(ws)
			if err != nil {
				botTimeout.Stop()
				// Clean up waiting state if this waiting player disconnects
				g.mu.Lock()
				if g.Waiting == player {
					g.Waiting = nil
					log.Printf("Waiting player %s disconnected, cleared waiting state", username)
				}
				g.mu.Unlock()
				break
			}
		}
	}
}

// handlePlayerMessages handles messages from a player
func (g *GlobalState) handlePlayerMessages(game *Game, player *Player, playerID int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
		}
	}()

	for {
		msg, err := player.ReadMessage()
		if err != nil {
			log.Printf("Player %s disconnected: %v", player.Username, err)
			break
		}

		if msg.Type == "move" {
			if game.ApplyMove(playerID, msg.Column) {
				// Valid move
				state := game.GetState()

				// Notify both players
				game.Player1.SendUpdate(state)
				game.Player2.SendUpdate(state)

				if game.Winner != 0 {
					// Update leaderboard
					g.mu.Lock()
					winnerUsername := getWinnerUsername(game, game.Winner)
					entry := g.Players[winnerUsername]
					if entry != nil {
						entry.Wins++
					}
					g.mu.Unlock()

					log.Printf("Game %s won by Player%d (%s)", game.ID, game.Winner, winnerUsername)
				}
			}
		}
	}
}

// readMessage reads a JSON message from a WebSocket connection
func readMessage(ws *websocket.Conn) (*Message, error) {
	var msg Message
	err := ws.ReadJSON(&msg)
	return &msg, err
}

// getWinnerUsername returns the username of the winner
func getWinnerUsername(game *Game, playerID int) string {
	if playerID == 1 {
		return game.Player1.Username
	}
	return game.Player2.Username
}

// BroadcastGameUpdate sends the current game state to both players
func (g *Game) BroadcastGameUpdate() {
	state := g.GetState()
	g.Player1.SendUpdate(state)
	g.Player2.SendUpdate(state)
}

// handleBotMoves monitors game state and executes bot moves automatically
func (gs *GlobalState) handleBotMoves(game *Game, bot *Player, botPlayerID int) {
	for {
		game.mu.Lock()

		// Check if game is over
		if game.GameOver {
			game.mu.Unlock()
			log.Printf("Game %s finished, bot handler exiting", game.ID)
			break
		}

		// Check if it's the bot's turn
		if game.Turn == botPlayerID {
			// Get best column for bot to play
			column := game.GetBotMove(botPlayerID)
			game.mu.Unlock()

			// Add delay for better UX (500ms)
			time.Sleep(500 * time.Millisecond)

			// Apply the move
			game.mu.Lock()
			if game.Turn == botPlayerID && !game.GameOver { // Double-check state hasn't changed
				game.ApplyMove(botPlayerID, column)

				// Broadcast update to human player
				state := game.GetState()
				game.Player1.SendUpdate(state)
				game.Player2.SendUpdate(state) // Will be no-op for bot

				// Check for win
				if game.Winner != 0 {
					gs.mu.Lock()
					winnerUsername := getWinnerUsername(game, game.Winner)
					entry := gs.Players[winnerUsername]
					if entry != nil {
						entry.Wins++
					}
					gs.mu.Unlock()
					log.Printf("Game %s won by Player%d (%s)", game.ID, game.Winner, winnerUsername)
				}
			}
			game.mu.Unlock()
		} else {
			game.mu.Unlock()
			// Wait a bit before checking again
			time.Sleep(100 * time.Millisecond)
		}
	}
}
