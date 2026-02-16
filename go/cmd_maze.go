package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func handleMaze(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for maze.")
		return
	}
	levelIndex := int64(0)
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "level" {
			levelIndex = o.IntValue()
		}
	}
	if levelIndex < 0 || int(levelIndex) >= len(levels) {
		levelIndex = 0
	}
	level := levels[levelIndex]

	state := gameState{
		UserID:    user.ID,
		Level:     int(levelIndex),
		PlayerX:   level.StartX,
		PlayerY:   level.StartY,
		Coins:     0,
		Lives:     3,
		Moves:     0,
		StartTime: time.Now().UnixMilli(),
		Board:     toBoard(level.Maze),
		GameOver:  false,
		Won:       false,
	}

	embed := buildMazeEmbed(state, level, false)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: mazeComponents(),
		},
	})
	if err != nil {
		log.Printf("maze respond error: %v", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Printf("maze fetch response error: %v", err)
		return
	}

	sessionMu.Lock()
	sessions[msg.ID] = &mazeSession{
		State:     state,
		ChannelID: msg.ChannelID,
		MessageID: msg.ID,
		UserID:    user.ID,
		Level:     level,
	}
	sessionMu.Unlock()

	go func(channelID, messageID string, timeoutSec int) {
		<-time.After(time.Duration(timeoutSec) * time.Second)

		sessionMu.Lock()
		sess, ok := sessions[messageID]
		if !ok || sess.State.GameOver {
			sessionMu.Unlock()
			return
		}
		sess.State.GameOver = true
		sess.State.Won = false
		stateCopy := sess.State
		levelCopy := sess.Level
		delete(sessions, messageID)
		sessionMu.Unlock()

		embed := buildMazeEmbed(stateCopy, levelCopy, true)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         messageID,
			Channel:    channelID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &[]discordgo.MessageComponent{},
		})
	}(msg.ChannelID, msg.ID, level.TimeLimit)
}

func handleMazeLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	levelFilter := int64(-1)
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "level" {
			levelFilter = o.IntValue()
		}
	}
	data, _ := readMazeScores()
	if len(data) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "No maze completions yet."}})
		return
	}

	if levelFilter == -1 {
		type row struct {
			Name        string
			Coins       int
			Completions int
		}
		rows := make([]row, 0, len(data))
		for _, v := range data {
			rows = append(rows, row{Name: v.Username, Coins: v.TotalCoins, Completions: len(v.Completions)})
		}
		sort.Slice(rows, func(a, b int) bool {
			if rows[a].Coins != rows[b].Coins {
				return rows[a].Coins > rows[b].Coins
			}
			return rows[a].Completions > rows[b].Completions
		})
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		for i2, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. %s - Coins:%d Completions:%d", i2+1, r.Name, r.Coins, r.Completions))
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: strings.Join(lines, "\n")}})
		return
	}

	type entry struct {
		Name  string
		Time  int
		Coins int
	}
	rows := []entry{}
	for _, u := range data {
		for _, c := range u.Completions {
			if c.Level == int(levelFilter) {
				rows = append(rows, entry{Name: u.Username, Time: c.Time, Coins: c.Coins})
			}
		}
	}
	if len(rows) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "No completions for this level yet."}})
		return
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].Time != rows[b].Time {
			return rows[a].Time < rows[b].Time
		}
		return rows[a].Coins > rows[b].Coins
	})
	if len(rows) > 10 {
		rows = rows[:10]
	}
	lines := []string{}
	for i2, r := range rows {
		lines = append(lines, fmt.Sprintf("%d. %s - %ds (%d coins)", i2+1, r.Name, r.Time, r.Coins))
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: strings.Join(lines, "\n")}})
}

func step(state *gameState, direction string) error {
	if state.GameOver {
		return nil
	}
	dx, dy, ok := dirDelta(direction)
	if !ok {
		return errors.New("invalid direction")
	}
	newX := state.PlayerX + dx
	newY := state.PlayerY + dy
	if newY < 0 || newY >= len(state.Board) {
		return nil
	}
	if newX < 0 || newX >= len(state.Board[newY]) {
		return nil
	}
	target := state.Board[newY][newX]
	if target == "#" {
		return nil
	}

	state.Board[state.PlayerY][state.PlayerX] = "."
	switch target {
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
	return nil
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

func toBoard(rows []string) [][]string {
	board := make([][]string, 0, len(rows))
	for _, row := range rows {
		cur := make([]string, 0, len(row))
		for _, ch := range row {
			cur = append(cur, string(ch))
		}
		board = append(board, cur)
	}
	return board
}

func buildMazeEmbed(state gameState, level levelDef, timeout bool) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(state.Board))
	for _, row := range state.Board {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			cells = append(cells, mazeCellEmoji(cell))
		}
		lines = append(lines, strings.Join(cells, ""))
	}
	board := strings.Join(lines, "\n")
	elapsed := int((time.Now().UnixMilli() - state.StartTime) / 1000)
	timeLeft := level.TimeLimit - elapsed
	if timeLeft < 0 {
		timeLeft = 0
	}

	title := fmt.Sprintf("\U0001F9E9 Maze: %s (%s)", level.Name, level.Difficulty)
	color := 0xEB459E
	result := ""
	if state.GameOver {
		if state.Won {
			title = "\U0001F3C6 Maze: Victory"
			color = 0x57F287
			result = "\U0001F973 You escaped the maze."
		}
		if timeout {
			title = "\u23F0 Maze: Time Up"
			color = 0xFEE75C
			result = "\u23F1\uFE0F Time ran out."
		}
		if !state.Won && !timeout {
			title = "\u2620\uFE0F Maze: Game Over"
			color = 0xED4245
			result = "\U0001F494 You lost all lives."
		}
	}

	desc := fmt.Sprintf("```\n%s\n```\n", board)
	if result != "" {
		desc += result + "\n\n"
	}
	desc += fmt.Sprintf("\u2764\uFE0F Lives: %d/3\n\U0001FA99 Coins: %d\n\u23F3 Time: %ds\n\U0001F3C3 Moves: %d\n\nKey: \U0001F7E6=You, \u2B1C=Path, \U0001F7EB=Wall, \U0001F3C1=Goal, \U0001FA99=Coin, \U0001F525=Spike, \U0001F47E=Enemy", state.Lives, state.Coins, timeLeft, state.Moves)

	return &discordgo.MessageEmbed{Title: title, Description: desc, Color: color}
}

func mazeCellEmoji(cell string) string {
	switch cell {
	case "P":
		return "\U0001F7E6"
	case ".":
		return "\u2B1C"
	case "#":
		return "\U0001F7EB"
	case "G":
		return "\U0001F3C1"
	case "C":
		return "\U0001FA99"
	case "S":
		return "\U0001F525"
	case "E":
		return "\U0001F47E"
	default:
		return cell
	}
}

func mazeComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "\u2B06\uFE0F Up", Style: discordgo.PrimaryButton, CustomID: "maze_up"}}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "\u2B05\uFE0F Left", Style: discordgo.PrimaryButton, CustomID: "maze_left"},
			discordgo.Button{Label: "\u2B07\uFE0F Down", Style: discordgo.PrimaryButton, CustomID: "maze_down"},
			discordgo.Button{Label: "\u27A1\uFE0F Right", Style: discordgo.PrimaryButton, CustomID: "maze_right"},
		}},
	}
}

func mazeScoresPath() string { return filepath.Join("..", "src", "data", "maze-scores.json") }

func readMazeScores() (map[string]userMazeData, error) {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := mazeScoresPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(path, []byte("{}"), 0o644)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]userMazeData{}
	if strings.TrimSpace(string(raw)) == "" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]userMazeData{}, nil
	}
	return out, nil
}

func saveMazeCompletion(userID, username string, level, sec, coins int) {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := mazeScoresPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	raw, _ := os.ReadFile(path)
	all := map[string]userMazeData{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		_ = json.Unmarshal(raw, &all)
	}
	u := all[userID]
	if u.Username == "" {
		u.Username = username
	}
	u.Username = username
	u.Completions = append(u.Completions, completion{Level: level, Time: sec, Coins: coins, Timestamp: time.Now().UnixMilli()})
	u.TotalCoins += coins
	all[userID] = u
	out, _ := json.MarshalIndent(all, "", "  ")
	_ = os.WriteFile(path, out, 0o644)
}
