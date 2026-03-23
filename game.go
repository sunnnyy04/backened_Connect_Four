package main

// ApplyMove places a piece in the given column and returns success
func (g *Game) ApplyMove(playerID int, column int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Winner != 0 || g.GameOver {
		return false
	}

	if column < 0 || column >= COLS {
		return false
	}

	if g.Turn != playerID {
		return false
	}

	// Find the lowest empty row in the column
	targetRow := -1
	for row := ROWS - 1; row >= 0; row-- {
		if g.Board[row][column] == 0 {
			targetRow = row
			break
		}
	}

	if targetRow == -1 {
		return false // Column is full
	}

	g.Board[targetRow][column] = playerID

	// Check for win
	if g.hasConnectFour(targetRow, column, playerID) {
		g.Winner = playerID
		g.GameOver = true
		return true
	}

	// Check if board is full
	if g.isBoardFull() {
		g.GameOver = true
		return true
	}

	// Switch turns
	if g.Turn == 1 {
		g.Turn = 2
	} else {
		g.Turn = 1
	}

	return true
}

// hasConnectFour checks if a player has four in a row
func (g *Game) hasConnectFour(row, col, player int) bool {
	directions := [][2]int{
		{1, 0},  // vertical
		{0, 1},  // horizontal
		{1, 1},  // diagonal down-right
		{1, -1}, // diagonal down-left
	}

	for _, dir := range directions {
		dr, dc := dir[0], dir[1]
		count := 1

		// Count forward
		r, c := row+dr, col+dc
		for r >= 0 && r < ROWS && c >= 0 && c < COLS && g.Board[r][c] == player {
			count++
			r += dr
			c += dc
		}

		// Count backward
		r, c = row-dr, col-dc
		for r >= 0 && r < ROWS && c >= 0 && c < COLS && g.Board[r][c] == player {
			count++
			r -= dr
			c -= dc
		}

		if count >= 4 {
			return true
		}
	}

	return false
}

// isBoardFull checks if the board is completely filled
func (g *Game) isBoardFull() bool {
	for _, row := range g.Board {
		for _, cell := range row {
			if cell == 0 {
				return false
			}
		}
	}
	return true
}

// GetBotMove returns the best column for a bot to play
func (g *Game) GetBotMove(botPlayerID int) int {
	// Try to win
	for col := 0; col < COLS; col++ {
		if g.canWinInColumn(col, botPlayerID) {
			return col
		}
	}

	// Try to block opponent
	opponentID := 3 - botPlayerID // If bot is 1, opponent is 2; if bot is 2, opponent is 1
	for col := 0; col < COLS; col++ {
		if g.canWinInColumn(col, opponentID) {
			return col
		}
	}

	// Play center columns (strategic)
	centerCols := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerCols {
		if g.isColumnValid(col) {
			return col
		}
	}

	// Fallback to first available column
	for col := 0; col < COLS; col++ {
		if g.isColumnValid(col) {
			return col
		}
	}

	return -1
}

// canWinInColumn checks if playing in this column results in a win
func (g *Game) canWinInColumn(col int, playerID int) bool {
	if col < 0 || col >= COLS {
		return false
	}

	// Find where the piece would land
	targetRow := -1
	for row := ROWS - 1; row >= 0; row-- {
		if g.Board[row][col] == 0 {
			targetRow = row
			break
		}
	}

	if targetRow == -1 {
		return false
	}

	// Check if this move would create 4 in a row
	return g.hasConnectFour(targetRow, col, playerID)
}

// isColumnValid checks if a column is playable
func (g *Game) isColumnValid(col int) bool {
	if col < 0 || col >= COLS {
		return false
	}
	return g.Board[0][col] == 0
}
