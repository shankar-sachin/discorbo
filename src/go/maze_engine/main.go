package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type gameState struct {
	UserID    string     `json:"userId"`
	Level     int        `json:"level"`
	PlayerX   int        `json:"playerX"`
	PlayerY   int        `json:"playerY"`
	Coins     int        `json:"coins"`
	Lives     int        `json:"lives"`
	Moves     int        `json:"moves"`
	StartTime int64      `json:"startTime"`
	Board     [][]string `json:"board"`
	GameOver  bool       `json:"gameOver"`
	Won       bool       `json:"won"`
}

type stepRequest struct {
	Op        string    `json:"op"`
	Direction string    `json:"direction"`
	State     gameState `json:"state"`
}

type stepResponse struct {
	State gameState `json:"state,omitempty"`
	Error string    `json:"error,omitempty"`
}

func main() {
	decoder := json.NewDecoder(os.Stdin)
	var req stepRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(fmt.Sprintf("invalid request JSON: %v", err))
		return
	}

	if req.Op != "step" {
		writeError("unsupported op")
		return
	}

	next, err := step(req.State, req.Direction)
	if err != nil {
		writeError(err.Error())
		return
	}

	_ = json.NewEncoder(os.Stdout).Encode(stepResponse{State: next})
}

func step(state gameState, direction string) (gameState, error) {
	if state.GameOver {
		return state, nil
	}

	if len(state.Board) == 0 {
		return state, fmt.Errorf("board is empty")
	}

	dx, dy, ok := dirDelta(direction)
	if !ok {
		return state, fmt.Errorf("invalid direction")
	}

	newX := state.PlayerX + dx
	newY := state.PlayerY + dy

	if newY < 0 || newY >= len(state.Board) {
		return state, nil
	}
	if newX < 0 || newX >= len(state.Board[newY]) {
		return state, nil
	}

	targetCell := state.Board[newY][newX]

	// Wall blocks movement.
	if targetCell == "#" {
		return state, nil
	}

	// Clear current tile.
	if state.PlayerY >= 0 && state.PlayerY < len(state.Board) &&
		state.PlayerX >= 0 && state.PlayerX < len(state.Board[state.PlayerY]) {
		state.Board[state.PlayerY][state.PlayerX] = "."
	}

	switch targetCell {
	case "C":
		state.Coins++
	case "S", "E":
		state.Lives--
		if state.Lives <= 0 {
			state.GameOver = true
			state.Won = false
		}
	case "G":
		state.GameOver = true
		state.Won = true
	}

	state.PlayerX = newX
	state.PlayerY = newY
	state.Board[newY][newX] = "P"
	state.Moves++

	return state, nil
}

func dirDelta(direction string) (int, int, bool) {
	switch direction {
	case "up":
		return 0, -1, true
	case "down":
		return 0, 1, true
	case "left":
		return -1, 0, true
	case "right":
		return 1, 0, true
	default:
		return 0, 0, false
	}
}

func writeError(message string) {
	_ = json.NewEncoder(os.Stdout).Encode(stepResponse{Error: message})
}

