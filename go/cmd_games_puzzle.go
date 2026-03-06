package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ─── Games group router ────────────────────────────────────────────────────────

func handleGamesCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	switch sub.Name {
	case "2048":
		handle2048(s, i)
	case "highlow":
		handleHighLow(s, i, sub.Options)
	case "maze":
		if len(sub.Options) == 0 {
			return
		}
		mazeSub := sub.Options[0]
		switch mazeSub.Name {
		case "play":
			handleMaze(s, i, mazeSub.Options)
		case "leaderboard":
			handleMazeLeaderboard(s, i, mazeSub.Options)
		}
	case "war":
		handleWar(s, i, sub.Options)
	case "snap":
		handleSnap(s, i, sub.Options)
	case "go-fish":
		handleGoFish(s, i, sub.Options)
	case "tag":
		handleTag(s, i, sub.Options)
	case "tictactoe":
		handleTicTacToe(s, i, sub.Options)
	case "connect4":
		handleConnect4(s, i, sub.Options)
	case "wordle":
		handleWordle(s, i)
	}
}

// ─── 2048 ──────────────────────────────────────────────────────────────────────

func handle2048(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	grid := [4][4]int{}
	spawn2048Tile(&grid)
	spawn2048Tile(&grid)

	sess := &g2048Session{
		UserID: user.ID,
		Grid:   grid,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{build2048Embed(sess)},
		Components: build2048Buttons(),
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	g2048Sessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(15 * time.Minute)
		gameMu.Lock()
		delete(g2048Sessions, msg.ID)
		gameMu.Unlock()
	}()
}

func spawn2048Tile(grid *[4][4]int) {
	empty := [][2]int{}
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if grid[r][c] == 0 {
				empty = append(empty, [2]int{r, c})
			}
		}
	}
	if len(empty) == 0 {
		return
	}
	pos := empty[rand.Intn(len(empty))]
	val := 2
	if rand.Intn(10) == 0 {
		val = 4
	}
	grid[pos[0]][pos[1]] = val
}

// update2048HighScore saves a new high score if it beats the previous one; returns the all-time high.
func update2048HighScore(userID string, score int) int {
	scores := map[string]int{}
	_ = readData("2048-scores.json", &scores)
	if score > scores[userID] {
		scores[userID] = score
		_ = writeData("2048-scores.json", scores)
	}
	return scores[userID]
}


func build2048Embed(sess *g2048Session) *discordgo.MessageEmbed {
	scores := map[string]int{}
	_ = readData("2048-scores.json", &scores)
	highScore := scores[sess.UserID]

	board := render2048Board(sess.Grid)
	tiles := render2048Emoji(sess.Grid)

	description := tiles + "\n" + board

	fields := []*discordgo.MessageEmbedField{
		{Name: "🏆 Score", Value: fmt.Sprintf("**%d**", sess.Score), Inline: true},
	}
	if highScore > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "👑 High Score", Value: fmt.Sprintf("**%d**", highScore), Inline: true})
	}

	return &discordgo.MessageEmbed{
		Title:       "🟨 2048",
		Description: description,
		Color:       ColorYellow,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Use the buttons to slide tiles!  •  " + botVersion},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// keep strings import used elsewhere in the file
var _ = strings.Join

func build2048Buttons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "⬆", Style: discordgo.PrimaryButton, CustomID: "g2048_up"},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "⬅", Style: discordgo.PrimaryButton, CustomID: "g2048_left"},
			discordgo.Button{Label: "Quit", Style: discordgo.DangerButton, CustomID: "g2048_quit"},
			discordgo.Button{Label: "➡", Style: discordgo.PrimaryButton, CustomID: "g2048_right"},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "⬇", Style: discordgo.PrimaryButton, CustomID: "g2048_down"},
		}},
	}
}

func handle2048Component(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	dir := strings.TrimPrefix(i.MessageComponentData().CustomID, "g2048_")

	gameMu.Lock()
	sess, ok := g2048Sessions[i.Message.ID]
	if !ok {
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	if sess.UserID != user.ID {
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This isn't your game!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if dir == "quit" {
		userID := sess.UserID
		score := sess.Score
		delete(g2048Sessions, i.Message.ID)
		gameMu.Unlock()
		hs := update2048HighScore(userID, score)
		embed := build2048Embed(sess)
		hsText := ""
		if score >= hs {
			hsText = " 🏆 **New high score!**"
		}
		embed.Description += fmt.Sprintf("\n\n🏳️ Game ended. Final score: **%d**%s", score, hsText)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	moved, score := slide2048(&sess.Grid, dir)
	sess.Score += score

	// Check win
	won := false
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if sess.Grid[r][c] == 2048 {
				won = true
			}
		}
	}

	if won {
		userID := sess.UserID
		score := sess.Score
		delete(g2048Sessions, i.Message.ID)
		gameMu.Unlock()
		update2048HighScore(userID, score)
		embed := build2048Embed(sess)
		embed.Color = ColorGreen
		embed.Description += fmt.Sprintf("\n\n🌟 **YOU REACHED 2048!** Score: **%d** 🏆", score)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	if moved {
		spawn2048Tile(&sess.Grid)
	}

	// Check game over
	if !has2048Moves(&sess.Grid) {
		userID := sess.UserID
		score := sess.Score
		delete(g2048Sessions, i.Message.ID)
		gameMu.Unlock()
		hs := update2048HighScore(userID, score)
		embed := build2048Embed(sess)
		embed.Color = ColorRed
		hsText := ""
		if score >= hs {
			hsText = " 🏆 **New high score!**"
		}
		embed.Description += fmt.Sprintf("\n\n💀 **Game Over!** No moves left. Score: **%d**%s", score, hsText)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	gameMu.Unlock()
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{build2048Embed(sess)},
			Components: build2048Buttons(),
		},
	})
}

// slide2048 moves all tiles in the given direction, merges, returns (moved, score)
func slide2048(grid *[4][4]int, dir string) (bool, int) {
	orig := *grid
	score := 0

	merge := func(row [4]int) ([4]int, int) {
		// Compact non-zero to left
		out := [4]int{}
		pos := 0
		for _, v := range row {
			if v != 0 {
				out[pos] = v
				pos++
			}
		}
		// Merge adjacent equal
		s := 0
		for idx := 0; idx < 3; idx++ {
			if out[idx] != 0 && out[idx] == out[idx+1] {
				out[idx] *= 2
				s += out[idx]
				out[idx+1] = 0
			}
		}
		// Compact again
		compact := [4]int{}
		pos = 0
		for _, v := range out {
			if v != 0 {
				compact[pos] = v
				pos++
			}
		}
		return compact, s
	}

	switch dir {
	case "left":
		for r := 0; r < 4; r++ {
			row := grid[r]
			merged, s := merge(row)
			grid[r] = merged
			score += s
		}
	case "right":
		for r := 0; r < 4; r++ {
			// Reverse row
			row := [4]int{grid[r][3], grid[r][2], grid[r][1], grid[r][0]}
			merged, s := merge(row)
			grid[r] = [4]int{merged[3], merged[2], merged[1], merged[0]}
			score += s
		}
	case "up":
		for c := 0; c < 4; c++ {
			col := [4]int{grid[0][c], grid[1][c], grid[2][c], grid[3][c]}
			merged, s := merge(col)
			for r := 0; r < 4; r++ {
				grid[r][c] = merged[r]
			}
			score += s
		}
	case "down":
		for c := 0; c < 4; c++ {
			col := [4]int{grid[3][c], grid[2][c], grid[1][c], grid[0][c]}
			merged, s := merge(col)
			for r := 0; r < 4; r++ {
				grid[r][c] = merged[3-r]
			}
			score += s
		}
	}
	return *grid != orig, score
}

func has2048Moves(grid *[4][4]int) bool {
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if grid[r][c] == 0 {
				return true
			}
			if c+1 < 4 && grid[r][c] == grid[r][c+1] {
				return true
			}
			if r+1 < 4 && grid[r][c] == grid[r+1][c] {
				return true
			}
		}
	}
	return false
}

// ─── Higher or Lower ───────────────────────────────────────────────────────────

func handleHighLow(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}
	bet := int(optionInt(opts, "bet", 10))
	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have %s but tried to bet %s.", coinDisplay(coins), coinDisplay(bet))))
		return
	}
	modifyCoins(user.ID, user.Username, -bet)

	deck := newDeck()
	currentCard := drawCard(&deck)

	sess := &hlSession{
		UserID:      user.ID,
		Username:    user.Username,
		Bet:         bet,
		CurrentCard: currentCard,
		Deck:        deck,
		Streak:      0,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildHLEmbed(sess, "")},
		Components: buildHLButtons(sess),
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	hlSessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		gameMu.Lock()
		if s2, ok := hlSessions[msg.ID]; ok {
			modifyCoins(s2.UserID, s2.Username, s2.Bet)
			delete(hlSessions, msg.ID)
		}
		gameMu.Unlock()
	}()
}

func buildHLEmbed(sess *hlSession, lastEvent string) *discordgo.MessageEmbed {
	mult := 1.0 + float64(sess.Streak)*0.5
	potentialWin := int(float64(sess.Bet) * mult)

	streakStr := ""
	if sess.Streak >= 3 {
		streakStr = fmt.Sprintf(" 🔥 **x%d Streak!**", sess.Streak)
	} else if sess.Streak > 0 {
		streakStr = fmt.Sprintf(" (streak: %d)", sess.Streak)
	}

	desc := fmt.Sprintf("**Current card:** %s%s\n\n💰 **Potential win:** %s (%.1fx)",
		renderCard(sess.CurrentCard), streakStr, coinDisplay(potentialWin), mult)
	if lastEvent != "" {
		desc += "\n\n" + lastEvent
	}
	return &discordgo.MessageEmbed{
		Title:       "🔢 Higher or Lower",
		Description: desc,
		Color:       ColorCasino,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "💰 Bet", Value: coinDisplay(sess.Bet), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Guess correctly to multiply winnings!  •  " + botVersion},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func buildHLButtons(sess *hlSession) []discordgo.MessageComponent {
	btns := []discordgo.MessageComponent{
		discordgo.Button{Label: "⬆ Higher", Style: discordgo.PrimaryButton, CustomID: "hl_higher"},
		discordgo.Button{Label: "⬇ Lower", Style: discordgo.PrimaryButton, CustomID: "hl_lower"},
	}
	if sess.Streak > 0 {
		btns = append(btns, discordgo.Button{Label: "💰 Cash Out", Style: discordgo.SuccessButton, CustomID: "hl_cashout"})
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: btns}}
}

func handleHLComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	action := strings.TrimPrefix(i.MessageComponentData().CustomID, "hl_")

	gameMu.Lock()
	sess, ok := hlSessions[i.Message.ID]
	if !ok {
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	if sess.UserID != user.ID {
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This isn't your game!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if action == "cashout" {
		mult := 1.0 + float64(sess.Streak)*0.5
		win := int(float64(sess.Bet) * mult)
		newBal := modifyCoins(user.ID, user.Username, win)
		delete(hlSessions, i.Message.ID)
		gameMu.Unlock()

		embed := buildHLEmbed(sess, fmt.Sprintf("💰 **Cashed out!** +%s (%.1fx)\nBalance: %s", coinDisplay(win), mult, coinDisplay(newBal)))
		embed.Color = ColorGreen
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	// Draw next card
	if len(sess.Deck) == 0 {
		sess.Deck = newDeck()
	}
	nextCard := drawCard(&sess.Deck)
	cv := cardRankValue(sess.CurrentCard)
	nv := cardRankValue(nextCard)

	correct := (action == "higher" && nv > cv) || (action == "lower" && nv < cv)
	tie := nv == cv

	event := fmt.Sprintf("Next card: %s", renderCard(nextCard))

	if tie {
		// Tie = push, draw new card and keep going
		sess.CurrentCard = nextCard
		event += "\n🤝 **Tie!** New card drawn. Try again."
		gameMu.Unlock()
		embed := buildHLEmbed(sess, event)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: buildHLButtons(sess),
			},
		})
		return
	}

	if correct {
		sess.Streak++
		sess.CurrentCard = nextCard
		mult := 1.0 + float64(sess.Streak)*0.5
		event += fmt.Sprintf("\n✅ **Correct!** Streak: %d (%.1fx multiplier)", sess.Streak, mult)
		gameMu.Unlock()
		embed := buildHLEmbed(sess, event)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: buildHLButtons(sess),
			},
		})
	} else {
		delete(hlSessions, i.Message.ID)
		gameMu.Unlock()
		coins, _ := getCoins(user.ID)
		event += fmt.Sprintf("\n❌ **Wrong!** You lose %s.\nBalance: %s", coinDisplay(sess.Bet), coinDisplay(coins))
		embed := buildHLEmbed(sess, event)
		embed.Color = ColorRed
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
	}
}

// ─── Tic-Tac-Toe ───────────────────────────────────────────────────────────────

func handleTicTacToe(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}
	opponent := optionUser(opts, "opponent")
	if opponent == nil {
		respondError(s, i, "Error", "You must specify an opponent.")
		return
	}
	if opponent.Bot {
		respondError(s, i, "Error", "You cannot play against bots.")
		return
	}
	if opponent.ID == user.ID {
		respondError(s, i, "Error", "You cannot play against yourself.")
		return
	}

	sess := &tttSession{
		Player1ID:   user.ID,
		Player1Name: user.Username,
		Player2ID:   opponent.ID,
		Player2Name: opponent.Username,
		Board:       [3][3]int{},
		CurrentTurn: 1,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildTTTEmbed(sess, "")},
		Components: buildTTTButtons(sess),
	})
	if err != nil {
		return
	}

	sess.MessageID = msg.ID
	sess.ChannelID = i.ChannelID

	tttMu.Lock()
	tttSessions[msg.ID] = sess
	tttMu.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		tttMu.Lock()
		delete(tttSessions, msg.ID)
		tttMu.Unlock()
	}()
}

func buildTTTEmbed(sess *tttSession, extra string) *discordgo.MessageEmbed {
	board := renderTicTacToe(sess.Board)
	currentName := sess.Player1Name
	if sess.CurrentTurn == 2 {
		currentName = sess.Player2Name
	}
	desc := fmt.Sprintf("%s\n\n❌ %s vs ⭕ %s\n🎯 **%s's turn**",
		board, sess.Player1Name, sess.Player2Name, currentName)
	if extra != "" {
		desc += "\n\n" + extra
	}
	return &discordgo.MessageEmbed{
		Title:       "❌⭕ Tic-Tac-Toe",
		Description: desc,
		Color:       ColorPurple,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Click a square to place your mark  •  " + botVersion},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func buildTTTButtons(sess *tttSession) []discordgo.MessageComponent {
	rows := []discordgo.MessageComponent{}
	for r := 0; r < 3; r++ {
		btns := []discordgo.MessageComponent{}
		for c := 0; c < 3; c++ {
			label := "⬜"
			disabled := false
			if sess.Board[r][c] == 1 {
				label = "❌"
				disabled = true
			} else if sess.Board[r][c] == 2 {
				label = "⭕"
				disabled = true
			}
			btns = append(btns, discordgo.Button{
				Label:    label,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("ttt_%d_%d", r, c),
				Disabled: disabled,
			})
		}
		rows = append(rows, discordgo.ActionsRow{Components: btns})
	}
	return rows
}

func checkTTTWin(board [3][3]int, player int) bool {
	for i := 0; i < 3; i++ {
		if board[i][0] == player && board[i][1] == player && board[i][2] == player {
			return true
		}
		if board[0][i] == player && board[1][i] == player && board[2][i] == player {
			return true
		}
	}
	if board[0][0] == player && board[1][1] == player && board[2][2] == player {
		return true
	}
	if board[0][2] == player && board[1][1] == player && board[2][0] == player {
		return true
	}
	return false
}

func isTTTFull(board [3][3]int) bool {
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if board[r][c] == 0 {
				return false
			}
		}
	}
	return true
}

func handleTTTComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	cid := i.MessageComponentData().CustomID
	var r, c int
	fmt.Sscanf(cid, "ttt_%d_%d", &r, &c)

	tttMu.Lock()
	sess, ok := tttSessions[i.Message.ID]
	if !ok {
		tttMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	expectedID := sess.Player1ID
	if sess.CurrentTurn == 2 {
		expectedID = sess.Player2ID
	}
	if user.ID != expectedID {
		tttMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "It's not your turn!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if r < 0 || r > 2 || c < 0 || c > 2 || sess.Board[r][c] != 0 {
		tttMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	sess.Board[r][c] = sess.CurrentTurn

	if checkTTTWin(sess.Board, sess.CurrentTurn) {
		winnerName := sess.Player1Name
		if sess.CurrentTurn == 2 {
			winnerName = sess.Player2Name
		}
		delete(tttSessions, i.Message.ID)
		tttMu.Unlock()
		embed := buildTTTEmbed(sess, fmt.Sprintf("🎉 **%s wins!**", winnerName))
		embed.Color = ColorGreen
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	if isTTTFull(sess.Board) {
		delete(tttSessions, i.Message.ID)
		tttMu.Unlock()
		embed := buildTTTEmbed(sess, "🤝 **It's a draw!**")
		embed.Color = ColorGray
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	if sess.CurrentTurn == 1 {
		sess.CurrentTurn = 2
	} else {
		sess.CurrentTurn = 1
	}
	tttMu.Unlock()

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{buildTTTEmbed(sess, "")},
			Components: buildTTTButtons(sess),
		},
	})
}

// ─── Connect Four ──────────────────────────────────────────────────────────────

func handleConnect4(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}
	opponent := optionUser(opts, "opponent")
	if opponent == nil {
		respondError(s, i, "Error", "You must specify an opponent.")
		return
	}
	if opponent.Bot {
		respondError(s, i, "Error", "You cannot play against bots.")
		return
	}
	if opponent.ID == user.ID {
		respondError(s, i, "Error", "You cannot play against yourself.")
		return
	}

	sess := &c4Session{
		Player1ID:   user.ID,
		Player1Name: user.Username,
		Player2ID:   opponent.ID,
		Player2Name: opponent.Username,
		Board:       [6][7]int{},
		CurrentTurn: 1,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildC4Embed(sess, "")},
		Components: buildC4Buttons(sess),
	})
	if err != nil {
		return
	}

	sess.MessageID = msg.ID
	sess.ChannelID = i.ChannelID

	c4Mu.Lock()
	c4Sessions[msg.ID] = sess
	c4Mu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		c4Mu.Lock()
		delete(c4Sessions, msg.ID)
		c4Mu.Unlock()
	}()
}

func buildC4Embed(sess *c4Session, extra string) *discordgo.MessageEmbed {
	board := renderConnect4Board(sess.Board)
	currentName := sess.Player1Name
	if sess.CurrentTurn == 2 {
		currentName = sess.Player2Name
	}
	desc := fmt.Sprintf("%s\n🔴 %s vs 🟡 %s\n🎯 **%s's turn**",
		board, sess.Player1Name, sess.Player2Name, currentName)
	if extra != "" {
		desc += "\n\n" + extra
	}
	return &discordgo.MessageEmbed{
		Title:       "🔴🟡 Connect Four",
		Description: desc,
		Color:       ColorPurple,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Click a column to drop your piece  •  " + botVersion},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func buildC4Buttons(sess *c4Session) []discordgo.MessageComponent {
	colLabels := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣"}
	btns := []discordgo.MessageComponent{}
	for c := 0; c < 7; c++ {
		disabled := sess.Board[0][c] != 0
		btns = append(btns, discordgo.Button{
			Label:    colLabels[c],
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("c4_%d", c),
			Disabled: disabled,
		})
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: btns[:5]},
		discordgo.ActionsRow{Components: btns[5:]},
	}
}

func c4DropPiece(board *[6][7]int, col, player int) int {
	for r := 5; r >= 0; r-- {
		if board[r][col] == 0 {
			board[r][col] = player
			return r
		}
	}
	return -1
}

func checkC4Win(board [6][7]int, player int) bool {
	for r := 0; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r][c+1] == player && board[r][c+2] == player && board[r][c+3] == player {
				return true
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c < 7; c++ {
			if board[r][c] == player && board[r+1][c] == player && board[r+2][c] == player && board[r+3][c] == player {
				return true
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r+1][c+1] == player && board[r+2][c+2] == player && board[r+3][c+3] == player {
				return true
			}
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r-1][c+1] == player && board[r-2][c+2] == player && board[r-3][c+3] == player {
				return true
			}
		}
	}
	return false
}

func isC4Full(board [6][7]int) bool {
	for c := 0; c < 7; c++ {
		if board[0][c] == 0 {
			return false
		}
	}
	return true
}

func handleC4Component(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	cid := i.MessageComponentData().CustomID
	var col int
	fmt.Sscanf(cid, "c4_%d", &col)

	c4Mu.Lock()
	sess, ok := c4Sessions[i.Message.ID]
	if !ok {
		c4Mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	expectedID := sess.Player1ID
	if sess.CurrentTurn == 2 {
		expectedID = sess.Player2ID
	}
	if user.ID != expectedID {
		c4Mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "It's not your turn!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if col < 0 || col > 6 {
		c4Mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	row := c4DropPiece(&sess.Board, col, sess.CurrentTurn)
	if row == -1 {
		c4Mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "That column is full!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if checkC4Win(sess.Board, sess.CurrentTurn) {
		winnerName := sess.Player1Name
		if sess.CurrentTurn == 2 {
			winnerName = sess.Player2Name
		}
		delete(c4Sessions, i.Message.ID)
		c4Mu.Unlock()
		embed := buildC4Embed(sess, fmt.Sprintf("🎉 **%s wins!**", winnerName))
		embed.Color = ColorGreen
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	if isC4Full(sess.Board) {
		delete(c4Sessions, i.Message.ID)
		c4Mu.Unlock()
		embed := buildC4Embed(sess, "🤝 **It's a draw!**")
		embed.Color = ColorGray
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	if sess.CurrentTurn == 1 {
		sess.CurrentTurn = 2
	} else {
		sess.CurrentTurn = 1
	}
	c4Mu.Unlock()

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{buildC4Embed(sess, "")},
			Components: buildC4Buttons(sess),
		},
	})
}

// ─── Wordle ────────────────────────────────────────────────────────────────────

var wordleWords = []string{
	"apple", "brave", "charm", "dance", "eagle", "flame", "grace", "house",
	"ivory", "jolly", "knife", "lemon", "magic", "noble", "ocean", "piano",
	"queen", "river", "stone", "tiger", "unity", "vivid", "water", "youth",
	"angel", "bloom", "crane", "drift", "ember", "frost", "globe", "hazel",
	"input", "joker", "kneel", "lotus", "maple", "nerve", "olive", "pearl",
	"quiet", "realm", "solar", "thorn", "ultra", "valve", "wheat", "xenon",
	"yacht", "zebra", "blaze", "clash", "drown", "flair",
}

func handleWordle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}

	word := wordleWords[rand.Intn(len(wordleWords))]
	sess := &wordleSession{
		UserID:  user.ID,
		Word:    word,
		Guesses: []string{},
		Results: [][]int{},
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildWordleEmbed(sess, "")},
		Components: buildWordleButtons(),
	})
	if err != nil {
		return
	}

	sess.MessageID = msg.ID
	sess.ChannelID = i.ChannelID

	wordleMu.Lock()
	wordleSessions[msg.ID] = sess
	wordleMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		wordleMu.Lock()
		delete(wordleSessions, msg.ID)
		wordleMu.Unlock()
	}()
}

func buildWordleEmbed(sess *wordleSession, extra string) *discordgo.MessageEmbed {
	board := renderWordleBoard(sess.Guesses, sess.Results)
	desc := fmt.Sprintf("%s\nGuesses: **%d/6**", board, len(sess.Guesses))
	if extra != "" {
		desc += "\n\n" + extra
	}
	return &discordgo.MessageEmbed{
		Title:       "📝 Wordle",
		Description: desc,
		Color:       ColorPurple,
		Footer:      &discordgo.MessageEmbedFooter{Text: "🟩 Correct  •  🟨 Wrong position  •  ⬛ Not in word  •  " + botVersion},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func buildWordleButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "✏️ Enter Guess", Style: discordgo.PrimaryButton, CustomID: "wordle_guess"},
		}},
	}
}

func evaluateWordle(guess, word string) []int {
	results := make([]int, 5)
	wordRunes := []rune(strings.ToLower(word))
	guessRunes := []rune(strings.ToLower(guess))
	used := [5]bool{}

	// First pass: correct position
	for i := 0; i < 5; i++ {
		if guessRunes[i] == wordRunes[i] {
			results[i] = 2
			used[i] = true
		}
	}

	// Second pass: wrong position
	for i := 0; i < 5; i++ {
		if results[i] == 2 {
			continue
		}
		for j := 0; j < 5; j++ {
			if !used[j] && guessRunes[i] == wordRunes[j] {
				results[i] = 1
				used[j] = true
				break
			}
		}
	}
	return results
}

func handleWordleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	wordleMu.Lock()
	sess, ok := wordleSessions[i.Message.ID]
	if !ok {
		wordleMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	if sess.UserID != user.ID {
		wordleMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This isn't your game!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	wordleMu.Unlock()

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("wordle_modal_%s", i.Message.ID),
			Title:    "Enter your guess",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "wordle_input",
						Label:       "5-letter word",
						Style:       discordgo.TextInputShort,
						Placeholder: "Enter a 5-letter word",
						Required:    true,
						MinLength:   5,
						MaxLength:   5,
					},
				}},
			},
		},
	})
}

func handleWordleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	parts := strings.Split(data.CustomID, "_")
	if len(parts) < 3 {
		return
	}
	msgID := parts[2]

	guess := ""
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range row.Components {
			if ti, ok := c.(*discordgo.TextInput); ok && ti.CustomID == "wordle_input" {
				guess = strings.ToLower(strings.TrimSpace(ti.Value))
			}
		}
	}

	if len([]rune(guess)) != 5 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Please enter exactly 5 letters.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	wordleMu.Lock()
	sess, ok := wordleSessions[msgID]
	if !ok {
		wordleMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Game session expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	results := evaluateWordle(guess, sess.Word)
	sess.Guesses = append(sess.Guesses, guess)
	sess.Results = append(sess.Results, results)

	won := true
	for _, r := range results {
		if r != 2 {
			won = false
			break
		}
	}

	if won {
		delete(wordleSessions, msgID)
		wordleMu.Unlock()
		embed := buildWordleEmbed(sess, fmt.Sprintf("🎉 **You got it in %d guesses!** The word was **%s**!", len(sess.Guesses), sess.Word))
		embed.Color = ColorGreen
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "✅ Correct!", Flags: discordgo.MessageFlagsEphemeral},
		})
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    sess.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &[]discordgo.MessageComponent{},
		})
		return
	}

	if len(sess.Guesses) >= 6 {
		delete(wordleSessions, msgID)
		wordleMu.Unlock()
		embed := buildWordleEmbed(sess, fmt.Sprintf("💀 **Game over!** The word was **%s**.", sess.Word))
		embed.Color = ColorRed
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Out of guesses!", Flags: discordgo.MessageFlagsEphemeral},
		})
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    sess.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &[]discordgo.MessageComponent{},
		})
		return
	}

	wordleMu.Unlock()
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Guess recorded: **%s** — %d/6 guesses used.", strings.ToUpper(guess), len(sess.Guesses)),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    sess.ChannelID,
		ID:         msgID,
		Embeds:     &[]*discordgo.MessageEmbed{buildWordleEmbed(sess, "")},
		Components: &[]discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "✏️ Enter Guess", Style: discordgo.PrimaryButton, CustomID: "wordle_guess"}}}},
	})
}
