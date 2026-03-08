package main

import (
	"fmt"
	"math/rand"
	"strconv"
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
	}
}

// ─── 2048 ──────────────────────────────────────────────────────────────────────

func handle2048(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
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

// build2048Grid renders the 4×4 grid as a monospace code block for Discord.
func build2048Grid(grid [4][4]int) string {
	var rows []string
	for r := 0; r < 4; r++ {
		var cells []string
		for c := 0; c < 4; c++ {
			v := grid[r][c]
			if v == 0 {
				cells = append(cells, "   ·")
			} else {
				cells = append(cells, fmt.Sprintf("%4s", strconv.Itoa(v)))
			}
		}
		rows = append(rows, strings.Join(cells, "  "))
	}
	return "```\n" + strings.Join(rows, "\n") + "\n```"
}

func build2048Embed(sess *g2048Session) *discordgo.MessageEmbed {
	scores := map[string]int{}
	_ = readData("2048-scores.json", &scores)
	highScore := scores[sess.UserID]

	fields := []*discordgo.MessageEmbedField{
		{Name: "Score", Value: fmt.Sprintf("%d", sess.Score), Inline: true},
	}
	if highScore > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Best", Value: fmt.Sprintf("%d", highScore), Inline: true})
	}

	return &discordgo.MessageEmbed{
		Title:       "🟨 2048",
		Description: build2048Grid(sess.Grid),
		Color:       ColorYellow,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Reach 2048 to win! | Discorbo"},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

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
		respondText(s, i, "Unable to identify user.")
		return
	}
	bet := int(optionInt(opts, "bet", 10))
	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have **%d coins** but tried to bet **%d**.", coins, bet)))
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
		streakStr = fmt.Sprintf(" 🔥 x%d Streak!", sess.Streak)
	} else if sess.Streak > 0 {
		streakStr = fmt.Sprintf(" (streak: %d)", sess.Streak)
	}

	desc := fmt.Sprintf("**Current card:** %s (value: %d)%s\n**Potential win:** %d coins (%.1fx)",
		sess.CurrentCard, cardRankValue(sess.CurrentCard), streakStr, potentialWin, mult)
	if lastEvent != "" {
		desc += "\n\n" + lastEvent
	}
	return &discordgo.MessageEmbed{
		Title:       "🔢 Higher or Lower",
		Description: desc,
		Color:       ColorBlue,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Bet", Value: fmt.Sprintf("%d coins", sess.Bet), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Guess correctly to multiply your winnings | Discorbo"},
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

		embed := buildHLEmbed(sess, fmt.Sprintf("💰 **Cashed out!** +**%d coins** (%.1fx)\nBalance: **%d coins**", win, mult, newBal))
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

	event := fmt.Sprintf("Next card: **%s** (value: %d)", nextCard, nv)

	if tie {
		// Tie = push, keep going
		event += "\n🤝 **Tie!** Try again."
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
		event += fmt.Sprintf("\n❌ **Wrong!** You lose **%d coins**.\nBalance: **%d coins**", sess.Bet, coins)
		embed := buildHLEmbed(sess, event)
		embed.Color = ColorRed
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
	}
}
