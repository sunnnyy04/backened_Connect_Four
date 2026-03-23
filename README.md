# Connect 4 Backend (Go)

A WebSocket-based Go backend for the Connect 4 game featuring real-time multiplayer support and leaderboard tracking.

## Features

- **WebSocket Support**: Real-time bidirectional communication with game clients
- **Multiplayer Matchmaking**: Automatically pairs waiting players
- **Game Logic**: Full Connect 4 rules implementation (horizontal, vertical, diagonal wins)
- **Leaderboard**: Track player wins across sessions
- **CORS Enabled**: Works with frontend on different origins
- **Concurrent Game Sessions**: Support for multiple simultaneous games

## Architecture

### Files

- **main.go** - Entry point, HTTP routes, and server initialization
- **models.go** - Data structures for games, players, and messages
- **game.go** - Core game logic (move validation, win detection)
- **handler.go** - WebSocket message handling and game orchestration

### Key Components

#### GameState
Represents current board state including:
- 6x7 game board
- Current player turn (1 or 2)
- Winner (0 = no winner yet)

#### Game Session
Manages two-player game with:
- Player connections
- Board state
- Turn tracking
- Win detection

#### Player Matching
- First player waits for opponent
- Second player automatically starts game
- Real-time board updates to both players

## API

### WebSocket (`/ws`)

#### Join Message
```json
{
  "type": "join",
  "username": "PlayerName"
}
```

#### Move Message
```json
{
  "type": "move",
  "column": 3
}
```

#### Game Update (received from server)
```json
{
  "type": "update",
  "board": [[0,0,1,2,...], ...],
  "turn": 1,
  "winner": 0
}
```

### HTTP

#### Leaderboard (`GET /leaderboard`)
Returns JSON array of players and their win counts:
```json
[
  {"username": "Player1", "wins": 5},
  {"username": "Player2", "wins": 3}
]
```

#### Health Check (`GET /health`)
```json
{"status": "ok"}
```

## Building

```bash
cd backened
go mod tidy
go build -o server
```

## Running

```bash
./server
```

Server starts on `http://localhost:8080`

## WebSocket URL

```
ws://localhost:8080/ws
```

## Game Rules

- Connect 4 pieces in a row (horizontal, vertical, or diagonal)
- Players alternate turns
- Column must not be full
- First player to connect 4 wins
- Game ends on win or board full (draw)

## Current Limitations

- In-memory player storage (resets on server restart)
- No persistent database
- Single-threaded game matching (could be optimized)

## Future Enhancements

- Database integration (PostgreSQL/MongoDB)
- Player authentication
- Rating system
- Game replay system
- Spectator mode
# backened_Connect_Four
