package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ─── Card games (poker routed via /casino, go-fish/snap via /games) ────────────

// ─── Poker (5-card video poker draw variant) ───────────────────────────────────

func handlePoker(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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
	hand := make([]string, 5)
	for idx := range hand {
		hand[idx] = drawCard(&deck)
	}

	sess := &pokerSession{
		UserID:   user.ID,
		Username: user.Username,
		Bet:      bet,
		Hand:     hand,
		Deck:     deck,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     buildPokerEmbeds(sess),
		Components: buildPokerHoldButtons(sess),
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	pokerSessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		gameMu.Lock()
		if s2, ok := pokerSessions[msg.ID]; ok && !s2.Drawn {
			modifyCoins(user.ID, user.Username, bet)
			delete(pokerSessions, msg.ID)
		}
		gameMu.Unlock()
	}()
}

func buildPokerEmbeds(sess *pokerSession) []*discordgo.MessageEmbed {
	phase := "Select cards to **HOLD**, then click **Draw**"
	if sess.Drawn {
		phase = "🏆 " + evaluatePokerHand(sess.Hand)
	}

	handStr := renderPokerHand(sess.Hand, sess.Held[:])

	embed := &discordgo.MessageEmbed{
		Title:       "♠️ Video Poker",
		Description: handStr,
		Color:       ColorPurple,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📋 Result", Value: phase, Inline: true},
			{Name: "💰 Bet", Value: fmt.Sprintf("**%d** coins", sess.Bet), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "✅ = held | Discorbo"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return []*discordgo.MessageEmbed{embed}
}

func buildPokerHoldButtons(sess *pokerSession) []discordgo.MessageComponent {
	holdRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{}}
	for idx := 0; idx < 5; idx++ {
		label := fmt.Sprintf("Card %d", idx+1)
		style := discordgo.SecondaryButton
		if sess.Held[idx] {
			style = discordgo.SuccessButton
			label = fmt.Sprintf("✅ %d", idx+1)
		}
		holdRow.Components = append(holdRow.Components, discordgo.Button{
			Label:    label,
			Style:    style,
			CustomID: fmt.Sprintf("poker_hold_%d", idx),
		})
	}
	drawRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "🎴 Draw", Style: discordgo.PrimaryButton, CustomID: "poker_draw"},
	}}
	return []discordgo.MessageComponent{holdRow, drawRow}
}

func handlePokerComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	customID := i.MessageComponentData().CustomID
	action := strings.TrimPrefix(customID, "poker_")

	gameMu.Lock()
	sess, ok := pokerSessions[i.Message.ID]
	if !ok || sess.Drawn {
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

	if strings.HasPrefix(action, "hold_") {
		var idx int
		_, err := fmt.Sscanf(action, "hold_%d", &idx)
		if err == nil && idx >= 0 && idx < 5 {
			sess.Held[idx] = !sess.Held[idx]
		}
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     buildPokerEmbeds(sess),
				Components: buildPokerHoldButtons(sess),
			},
		})
		return
	}

	if action == "draw" {
		// Replace non-held cards
		for idx := range sess.Hand {
			if !sess.Held[idx] {
				sess.Hand[idx] = drawCard(&sess.Deck)
			}
		}
		sess.Drawn = true
		handName := evaluatePokerHand(sess.Hand)
		mult := pokerPayoutMult(handName)
		var newBal int
		var resultLine string
		if mult > 0 {
			win := int(float64(sess.Bet) * float64(mult))
			newBal = modifyCoins(user.ID, user.Username, win)
			resultLine = fmt.Sprintf("🎉 **%s** — +**%d coins** (%dx)\nBalance: **%d coins**", handName, win, mult, newBal)
		} else {
			coins, _ := getCoins(user.ID)
			resultLine = fmt.Sprintf("😞 **%s** — No win.\nBalance: **%d coins**", handName, coins)
			newBal = coins
		}

		delete(pokerSessions, i.Message.ID)
		gameMu.Unlock()

		embeds := buildPokerEmbeds(sess)
		embeds[0].Description = strings.Join(sess.Hand, "  ")
		embeds[0].Fields[0].Value = handName
		embeds[0].Fields = append(embeds[0].Fields, &discordgo.MessageEmbedField{Name: "Result", Value: resultLine, Inline: false})
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: embeds, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	gameMu.Unlock()
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
}

// evaluatePokerHand returns human-readable hand name
func evaluatePokerHand(hand []string) string {
	rankCounts := map[string]int{}
	suitCounts := map[string]int{}
	rankVals := []int{}

	for _, card := range hand {
		r := cardRankStr(card)
		runes := []rune(card)
		suit := string(runes[len(runes)-1:])
		rankCounts[r]++
		suitCounts[suit]++
		rankVals = append(rankVals, cardRankValue(card))
	}

	sort.Ints(rankVals)
	isFlush := len(suitCounts) == 1
	isStraight := false
	if rankVals[4]-rankVals[0] == 4 && len(rankCounts) == 5 {
		isStraight = true
	}
	// Wheel straight A-2-3-4-5
	if rankVals[4] == 14 && rankVals[0] == 2 && rankVals[1] == 3 && rankVals[2] == 4 && rankVals[3] == 5 {
		isStraight = true
	}

	counts := []int{}
	for _, c := range rankCounts {
		counts = append(counts, c)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(counts)))

	switch {
	case isFlush && isStraight && rankVals[0] == 10:
		return "Royal Flush"
	case isFlush && isStraight:
		return "Straight Flush"
	case counts[0] == 4:
		return "Four of a Kind"
	case counts[0] == 3 && counts[1] == 2:
		return "Full House"
	case isFlush:
		return "Flush"
	case isStraight:
		return "Straight"
	case counts[0] == 3:
		return "Three of a Kind"
	case counts[0] == 2 && counts[1] == 2:
		return "Two Pair"
	case counts[0] == 2:
		// Jacks or better check
		for r, c := range rankCounts {
			if c == 2 {
				v := cardRankValue(r + "♠")
				if v >= 11 {
					return "Jacks or Better"
				}
			}
		}
		return "One Pair (No payout)"
	default:
		return "High Card"
	}
}

func pokerPayoutMult(hand string) int {
	switch hand {
	case "Royal Flush":
		return 250
	case "Straight Flush":
		return 50
	case "Four of a Kind":
		return 25
	case "Full House":
		return 9
	case "Flush":
		return 6
	case "Straight":
		return 4
	case "Three of a Kind":
		return 3
	case "Two Pair":
		return 2
	case "Jacks or Better":
		return 1
	default:
		return 0
	}
}

// ─── Go Fish ───────────────────────────────────────────────────────────────────

func handleGoFish(s *discordgo.Session, i *discordgo.InteractionCreate, _ []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	deck := newDeck()
	playerHand := make([]string, 7)
	botHand := make([]string, 7)
	for idx := range playerHand {
		playerHand[idx] = drawCard(&deck)
	}
	for idx := range botHand {
		botHand[idx] = drawCard(&deck)
	}

	sess := &fishSession{
		UserID:     user.ID,
		PlayerHand: playerHand,
		BotHand:    botHand,
		Deck:       deck,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildFishEmbed(sess, "")},
		Components: buildFishButtons(sess),
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	fishSessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(15 * time.Minute)
		gameMu.Lock()
		delete(fishSessions, msg.ID)
		gameMu.Unlock()
	}()
}

func buildFishEmbed(sess *fishSession, lastEvent string) *discordgo.MessageEmbed {
	// Show player hand as compact cards grouped by rank
	rankSet := map[string]int{}
	for _, card := range sess.PlayerHand {
		rankSet[cardRankStr(card)]++
	}
	handDisplay := []string{}
	for r, c := range rankSet {
		handDisplay = append(handDisplay, fmt.Sprintf("`%s`×%d", r, c))
	}
	sort.Strings(handDisplay)

	handCards := renderHand(sess.PlayerHand)

	desc := fmt.Sprintf("🃏 **Your Cards:** %s\n📊 **Ranks:** %s\n\n📦 **Your Books:** %d | **Bot Books:** %d\n🃏 **Your Cards:** %d | **Bot Cards:** %d",
		handCards, strings.Join(handDisplay, " "), sess.PlayerBooks, sess.BotBooks, len(sess.PlayerHand), len(sess.BotHand))
	if lastEvent != "" {
		desc += "\n\n" + lastEvent
	}
	desc += "\n\n*Ask the bot for a rank:*"

	return &discordgo.MessageEmbed{
		Title:       "🐟 Go Fish",
		Description: desc,
		Color:       ColorBlue,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Collect 4 of a rank to make a book! | Discorbo"},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func buildFishButtons(sess *fishSession) []discordgo.MessageComponent {
	// Only show ranks the player currently has
	rankSet := map[string]bool{}
	for _, card := range sess.PlayerHand {
		rankSet[cardRankStr(card)] = true
	}
	ranks := []string{}
	for r := range rankSet {
		ranks = append(ranks, r)
	}
	sort.Strings(ranks)

	if len(ranks) == 0 {
		return []discordgo.MessageComponent{}
	}

	// Max 5 buttons per row, up to 3 rows = 15 buttons
	rows := []discordgo.MessageComponent{}
	btns := []discordgo.MessageComponent{}
	for _, r := range ranks {
		if len(btns) == 5 {
			rows = append(rows, discordgo.ActionsRow{Components: btns})
			btns = []discordgo.MessageComponent{}
		}
		if len(rows) >= 3 {
			break
		}
		btns = append(btns, discordgo.Button{
			Label:    r,
			Style:    discordgo.PrimaryButton,
			CustomID: "fish_ask_" + r,
		})
	}
	if len(btns) > 0 && len(rows) < 3 {
		rows = append(rows, discordgo.ActionsRow{Components: btns})
	}
	return rows
}

func handleFishComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	customID := i.MessageComponentData().CustomID
	if !strings.HasPrefix(customID, "fish_ask_") {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	askedRank := strings.TrimPrefix(customID, "fish_ask_")

	gameMu.Lock()
	sess, ok := fishSessions[i.Message.ID]
	if !ok {
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Game session not found.", Flags: discordgo.MessageFlagsEphemeral},
		})
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

	var event string
	// Check if bot has the asked rank
	given := []string{}
	remaining := []string{}
	for _, card := range sess.BotHand {
		if cardRankStr(card) == askedRank {
			given = append(given, card)
		} else {
			remaining = append(remaining, card)
		}
	}

	if len(given) > 0 {
		sess.BotHand = remaining
		sess.PlayerHand = append(sess.PlayerHand, given...)
		event = fmt.Sprintf("🎣 Bot had %d **%s**! You took them.", len(given), askedRank)
	} else {
		event = fmt.Sprintf("🐟 **Go Fish!** Bot had no **%s**.", askedRank)
		if len(sess.Deck) > 0 {
			drawn := drawCard(&sess.Deck)
			sess.PlayerHand = append(sess.PlayerHand, drawn)
			event += fmt.Sprintf(" You drew **%s**.", drawn)
		}
	}

	// Check and remove books from player hand
	playerBooks := extractBooks(&sess.PlayerHand)
	sess.PlayerBooks += playerBooks
	if playerBooks > 0 {
		event += fmt.Sprintf("\n📚 **Book%s!** +%d book%s", pluralS(playerBooks), playerBooks, pluralS(playerBooks))
	}

	// Bot's turn: ask for a random rank from bot's hand
	botEvent := botFishTurn(sess)
	if botEvent != "" {
		event += "\n" + botEvent
	}

	// Check game over
	gameOver := len(sess.PlayerHand) == 0 && len(sess.BotHand) == 0
	if gameOver {
		var endMsg string
		if sess.PlayerBooks > sess.BotBooks {
			endMsg = fmt.Sprintf("🏆 **You win!** %d books vs %d books.", sess.PlayerBooks, sess.BotBooks)
		} else if sess.BotBooks > sess.PlayerBooks {
			endMsg = fmt.Sprintf("😞 **Bot wins.** %d books vs %d books.", sess.BotBooks, sess.PlayerBooks)
		} else {
			endMsg = fmt.Sprintf("🤝 **Tie!** Both have %d books.", sess.PlayerBooks)
		}
		delete(fishSessions, i.Message.ID)
		gameMu.Unlock()

		embed := buildFishEmbed(sess, event+"\n\n"+endMsg)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})
		return
	}

	// If player has no cards, draw
	if len(sess.PlayerHand) == 0 && len(sess.Deck) > 0 {
		sess.PlayerHand = append(sess.PlayerHand, drawCard(&sess.Deck))
		event += "\n🃏 You have no cards — drew one from the deck."
	}

	gameMu.Unlock()
	embed := buildFishEmbed(sess, event)
	components := buildFishButtons(sess)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}

func extractBooks(hand *[]string) int {
	rankCounts := map[string][]string{}
	for _, card := range *hand {
		r := cardRankStr(card)
		rankCounts[r] = append(rankCounts[r], card)
	}
	books := 0
	newHand := []string{}
	for r, cards := range rankCounts {
		if len(cards) == 4 {
			books++
			_ = r
		} else {
			newHand = append(newHand, cards...)
		}
	}
	*hand = newHand
	return books
}

func botFishTurn(sess *fishSession) string {
	if len(sess.BotHand) == 0 {
		return ""
	}
	// Bot picks a random rank from its hand to ask for
	askedRank := cardRankStr(sess.BotHand[rand.Intn(len(sess.BotHand))])

	given := []string{}
	remaining := []string{}
	for _, card := range sess.PlayerHand {
		if cardRankStr(card) == askedRank {
			given = append(given, card)
		} else {
			remaining = append(remaining, card)
		}
	}

	var event string
	if len(given) > 0 {
		sess.PlayerHand = remaining
		sess.BotHand = append(sess.BotHand, given...)
		event = fmt.Sprintf("🤖 Bot asks for **%s** — you had %d! Bot takes them.", askedRank, len(given))
	} else {
		event = fmt.Sprintf("🤖 Bot asks for **%s** — Go Fish!", askedRank)
		if len(sess.Deck) > 0 {
			sess.BotHand = append(sess.BotHand, drawCard(&sess.Deck))
		}
	}

	botBooks := extractBooks(&sess.BotHand)
	sess.BotBooks += botBooks
	if botBooks > 0 {
		event += fmt.Sprintf(" 🤖 Bot got %d book%s!", botBooks, pluralS(botBooks))
	}
	return event
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ─── Snap ──────────────────────────────────────────────────────────────────────

func handleSnap(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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
	card1 := drawCard(&deck)
	card2 := drawCard(&deck)

	sess := &snapSession{
		UserID:   user.ID,
		Username: user.Username,
		Bet:      bet,
		Deck:     deck,
		LastTwo:  []string{card1, card2},
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildSnapEmbed(sess, "")},
		Components: buildSnapButtons(),
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	snapSessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		gameMu.Lock()
		if _, ok := snapSessions[msg.ID]; ok {
			modifyCoins(user.ID, user.Username, bet)
			delete(snapSessions, msg.ID)
		}
		gameMu.Unlock()
	}()
}

func buildSnapEmbed(sess *snapSession, lastEvent string) *discordgo.MessageEmbed {
	top := ""
	if len(sess.LastTwo) >= 2 {
		top = fmt.Sprintf("**Pile:** %s  |  %s", renderCard(sess.LastTwo[0]), renderCard(sess.LastTwo[1]))
	}
	desc := fmt.Sprintf("%s\n🃏 **Deck:** %d cards remaining\n💰 **Bet:** %d coins", top, len(sess.Deck), sess.Bet)
	if lastEvent != "" {
		desc += "\n\n" + lastEvent
	}
	return &discordgo.MessageEmbed{
		Title:       "⚡ Snap!",
		Description: desc,
		Color:       ColorYellow,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Press SNAP when top two cards match! | Discorbo"},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func buildSnapButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "⚡ SNAP!", Style: discordgo.DangerButton, CustomID: "snap_snap"},
			discordgo.Button{Label: "Flip Card", Style: discordgo.PrimaryButton, CustomID: "snap_flip"},
		}},
	}
}

func handleSnapComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	action := strings.TrimPrefix(i.MessageComponentData().CustomID, "snap_")

	gameMu.Lock()
	sess, ok := snapSessions[i.Message.ID]
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

	switch action {
	case "snap":
		isMatch := len(sess.LastTwo) >= 2 && cardRankStr(sess.LastTwo[0]) == cardRankStr(sess.LastTwo[1])
		delete(snapSessions, i.Message.ID)
		gameMu.Unlock()

		if isMatch {
			newBal := modifyCoins(user.ID, user.Username, sess.Bet*2)
			embed := buildSnapEmbed(sess, fmt.Sprintf("⚡ **SNAP!** Correct match! **+%d coins**!\nBalance: **%d coins**", sess.Bet, newBal))
			embed.Color = ColorGreen
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
			})
		} else {
			coins, _ := getCoins(user.ID)
			embed := buildSnapEmbed(sess, fmt.Sprintf("❌ **False Snap!** Cards don't match. **-%d coins**.\nBalance: **%d coins**", sess.Bet, coins))
			embed.Color = ColorRed
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
			})
		}

	case "flip":
		// Bot has 40% chance to snap first if it's a match
		isMatch := len(sess.LastTwo) >= 2 && cardRankStr(sess.LastTwo[0]) == cardRankStr(sess.LastTwo[1])
		if isMatch && rand.Float64() < 0.4 {
			// Bot snaps first
			delete(snapSessions, i.Message.ID)
			gameMu.Unlock()
			coins, _ := getCoins(user.ID)
			embed := buildSnapEmbed(sess, fmt.Sprintf("🤖 **Bot snapped first!** You lose **%d coins**.\nBalance: **%d coins**", sess.Bet, coins))
			embed.Color = ColorRed
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
			})
			return
		}

		// Flip a new card
		if len(sess.Deck) == 0 {
			delete(snapSessions, i.Message.ID)
			gameMu.Unlock()
			newBal := modifyCoins(user.ID, user.Username, sess.Bet)
			embed := buildSnapEmbed(sess, fmt.Sprintf("🃏 Deck empty! Bet refunded. Balance: **%d coins**", newBal))
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
			})
			return
		}

		newCard := drawCard(&sess.Deck)
		sess.LastTwo = []string{sess.LastTwo[len(sess.LastTwo)-1], newCard}
		gameMu.Unlock()

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{buildSnapEmbed(sess, fmt.Sprintf("🃏 New card flipped: **%s**", newCard))},
				Components: buildSnapButtons(),
			},
		})

	default:
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	}
}
