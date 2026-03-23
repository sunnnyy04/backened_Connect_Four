package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

var globalState = &GlobalState{
	Games:   make(map[string]*Game),
	Players: make(map[string]*LeaderboardEntry),
	mu:      sync.RWMutex{},
}

func main() {
	router := gin.Default()

	// Add CORS middleware
	router.Use(CORSMiddleware())

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		HandleWSConnection(c)
	})

	// Leaderboard endpoint
	router.GET("/leaderboard", func(c *gin.Context) {
		HandleLeaderboard(c)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	log.Println("Server starting on http://localhost:8080")
	router.Run(":8080")
}

// HandleWSConnection handles a new WebSocket connection
func HandleWSConnection(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	log.Println("New WebSocket connection established")
	globalState.HandleWebSocket(ws)
}

// HandleLeaderboard returns the leaderboard
func HandleLeaderboard(c *gin.Context) {
	globalState.mu.RLock()
	defer globalState.mu.RUnlock()

	leaderboard := make([]LeaderboardEntry, 0)
	for _, entry := range globalState.Players {
		leaderboard = append(leaderboard, *entry)
	}

	c.JSON(http.StatusOK, leaderboard)
}

// CORSMiddleware adds CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
