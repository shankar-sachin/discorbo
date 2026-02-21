package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ─── Shared deck/card utilities ───────────────────────────────────────────────

var cardSuits = []string{"♠", "♥", "♦", "♣"}
var cardRanks = []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

func newDeck() []string {
	deck := make([]string, 0, 52)
	for _, s := range cardSuits {
		for _, r := range cardRanks {
			deck = append(deck, r+s)
		}
	}
	rand.Shuffle(len(deck), func(a, b int) { deck[a], deck[b] = deck[b], deck[a] })
	return deck
}

func drawCard(deck *[]string) string {
	if len(*deck) == 0 {
		*deck = newDeck()
	}
	card := (*deck)[0]
	*deck = (*deck)[1:]
	return card
}

func cardRankStr(card string) string {
	// rank is everything except the last rune (suit)
	runes := []rune(card)
	return string(runes[:len(runes)-1])
}

// cardRankValue returns 2-14 (A=14) for War/HighLow comparisons
func cardRankValue(card string) int {
	r := cardRankStr(card)
	switch r {
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	default:
		v := 0
		for _, ch := range r {
			v = v*10 + int(ch-'0')
		}
		return v
	}
}

// handValue returns blackjack value with ace adjustment
func handValue(hand []string) int {
	total := 0
	aces := 0
	for _, card := range hand {
		r := cardRankStr(card)
		switch r {
		case "A":
			total += 11
			aces++
		case "J", "Q", "K":
			total += 10
		default:
			v := 0
			for _, ch := range r {
				v = v*10 + int(ch-'0')
			}
			total += v
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func handStr(hand []string) string {
	return strings.Join(hand, " ")
}

// ─── Economy helpers ───────────────────────────────────────────────────────────

func getCoins(userID string) (int, string) {
	m := map[string]economyUser{}
	_ = readData("economy-users.json", &m)
	u := m[userID]
	return u.Coins, u.Username
}

func modifyCoins(userID, username string, delta int) int {
	m := map[string]economyUser{}
	_ = readData("economy-users.json", &m)
	u := m[userID]
	if username != "" {
		u.Username = username
	}
	u.Coins += delta
	if u.Coins < 0 {
		u.Coins = 0
	}
	m[userID] = u
	_ = writeData("economy-users.json", m)
	return u.Coins
}

// ─── Casino games router ───────────────────────────────────────────────────────

func handleCasinoGames(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	switch name {
	case "blackjack":
		handleBlackjack(s, i)
	case "slots":
		handleSlots(s, i)
	case "roulette":
		handleRoulette(s, i)
	case "russian-roulette":
		handleRussianRoulette(s, i)
	case "war":
		handleWar(s, i)
	}
}

// ─── Blackjack ─────────────────────────────────────────────────────────────────

func handleBlackjack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	opts := i.ApplicationCommandData().Options
	bet := int(optionInt(opts, "bet", 10))
	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have **%d coins** but tried to bet **%d**.", coins, bet)))
		return
	}

	deck := newDeck()
	playerHand := []string{drawCard(&deck), drawCard(&deck)}
	dealerHand := []string{drawCard(&deck), drawCard(&deck)}

	sess := &bjSession{
		UserID:     user.ID,
		Username:   user.Username,
		Bet:        bet,
		PlayerHand: playerHand,
		DealerHand: dealerHand,
		Deck:       deck,
	}

	// Check for blackjack
	if handValue(playerHand) == 21 {
		payout := int(float64(bet) * 1.5)
		newBal := modifyCoins(user.ID, user.Username, payout)
		embed := createFunEmbed("🃏 Blackjack!", fmt.Sprintf(
			"**Your hand:** %s (21)\n**Dealer:** %s %s\n\n🎉 **BLACKJACK!** You win **+%d coins**!\nBalance: **%d coins**",
			handStr(playerHand), dealerHand[0], "??", payout, newBal))
		respondEmbed(s, i, embed)
		return
	}

	// Deduct bet upfront
	modifyCoins(user.ID, user.Username, -bet)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{buildBJEmbed(sess, false)},
		Components: buildBJButtons(true), // first turn: show Double Down
	})
	if err != nil {
		return
	}

	gameMu.Lock()
	bjSessions[msg.ID] = sess
	gameMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		gameMu.Lock()
		if s2, ok := bjSessions[msg.ID]; ok && !s2.Done {
			// Timeout — refund
			modifyCoins(user.ID, user.Username, bet)
			delete(bjSessions, msg.ID)
		}
		gameMu.Unlock()
	}()
}

func buildBJEmbed(sess *bjSession, reveal bool) *discordgo.MessageEmbed {
	dealerStr := sess.DealerHand[0] + " ??"
	dealerVal := ""
	if reveal {
		dealerStr = handStr(sess.DealerHand)
		dealerVal = fmt.Sprintf(" (%d)", handValue(sess.DealerHand))
	}
	return &discordgo.MessageEmbed{
		Title: "🃏 Blackjack",
		Color: ColorPurple,
		Fields: []*discordgo.MessageEmbedField{
			{Name: fmt.Sprintf("Your Hand (%d)", handValue(sess.PlayerHand)), Value: handStr(sess.PlayerHand), Inline: true},
			{Name: "Dealer's Hand" + dealerVal, Value: dealerStr, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d coins", sess.Bet), Inline: false},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Discorbo"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func buildBJButtons(firstTurn bool) []discordgo.MessageComponent {
	btns := []discordgo.MessageComponent{
		discordgo.Button{Label: "Hit", Style: discordgo.PrimaryButton, CustomID: "bj_hit"},
		discordgo.Button{Label: "Stand", Style: discordgo.SecondaryButton, CustomID: "bj_stand"},
	}
	if firstTurn {
		btns = append(btns, discordgo.Button{Label: "Double Down", Style: discordgo.SuccessButton, CustomID: "bj_double"})
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: btns}}
}

func handleBJComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	action := strings.TrimPrefix(i.MessageComponentData().CustomID, "bj_")

	gameMu.Lock()
	sess, ok := bjSessions[i.Message.ID]
	if !ok || sess.Done {
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
	case "hit":
		sess.PlayerHand = append(sess.PlayerHand, drawCard(&sess.Deck))
		pv := handValue(sess.PlayerHand)
		if pv > 21 {
			sess.Done = true
			delete(bjSessions, i.Message.ID)
			gameMu.Unlock()
			coins, _ := getCoins(user.ID)
			embed := buildBJEmbed(sess, true)
			embed.Color = ColorRed
			embed.Description = fmt.Sprintf("💥 **Bust!** You went over 21.\nYou lose **%d coins**.\nBalance: **%d coins**", sess.Bet, coins)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
			})
			return
		}
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{buildBJEmbed(sess, false)}, Components: buildBJButtons(false)},
		})

	case "double":
		// Double down: double the bet, draw one card, then stand
		extraCoins, _ := getCoins(user.ID)
		if extraCoins < sess.Bet {
			gameMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Not enough coins to double down!", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}
		modifyCoins(user.ID, user.Username, -sess.Bet)
		sess.Bet *= 2
		sess.PlayerHand = append(sess.PlayerHand, drawCard(&sess.Deck))
		fallthrough

	case "stand":
		// Dealer plays
		for handValue(sess.DealerHand) < 17 {
			sess.DealerHand = append(sess.DealerHand, drawCard(&sess.Deck))
		}
		pv := handValue(sess.PlayerHand)
		dv := handValue(sess.DealerHand)
		sess.Done = true
		delete(bjSessions, i.Message.ID)
		gameMu.Unlock()

		var result string
		var newBal int
		embed := buildBJEmbed(sess, true)
		if pv > 21 {
			embed.Color = ColorRed
			coins, _ := getCoins(user.ID)
			result = fmt.Sprintf("💥 **Bust!** You lose **%d coins**.", sess.Bet)
			newBal = coins
		} else if dv > 21 || pv > dv {
			newBal = modifyCoins(user.ID, user.Username, sess.Bet*2)
			result = fmt.Sprintf("🎉 **You win!** +**%d coins**!", sess.Bet)
			embed.Color = ColorGreen
		} else if pv == dv {
			newBal = modifyCoins(user.ID, user.Username, sess.Bet)
			result = "🤝 **Push!** Bet refunded."
			embed.Color = ColorGray
		} else {
			coins, _ := getCoins(user.ID)
			result = fmt.Sprintf("😞 **Dealer wins.** You lose **%d coins**.", sess.Bet)
			embed.Color = ColorRed
			newBal = coins
		}
		embed.Description = result + fmt.Sprintf("\nBalance: **%d coins**", newBal)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})

	default:
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	}
}

// ─── Slots ─────────────────────────────────────────────────────────────────────

var slotsSymbols = []string{"🍒", "🍋", "🍇", "🔔", "💎", "7️⃣"}
var slotsWeights = []int{30, 25, 20, 15, 7, 3} // relative weights

func weightedSlot() string {
	total := 0
	for _, w := range slotsWeights {
		total += w
	}
	r := rand.Intn(total)
	cum := 0
	for idx, w := range slotsWeights {
		cum += w
		if r < cum {
			return slotsSymbols[idx]
		}
	}
	return slotsSymbols[0]
}

func handleSlots(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	opts := i.ApplicationCommandData().Options
	bet := int(optionInt(opts, "bet", 10))
	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have **%d coins** but tried to bet **%d**.", coins, bet)))
		return
	}

	r1, r2, r3 := weightedSlot(), weightedSlot(), weightedSlot()
	display := fmt.Sprintf("[ %s | %s | %s ]", r1, r2, r3)

	var mult float64
	var resultText string
	switch {
	case r1 == "7️⃣" && r2 == r1 && r3 == r1:
		mult = 50
		resultText = "🎰 **JACKPOT!!!** Three 7s!"
	case r1 == "💎" && r2 == r1 && r3 == r1:
		mult = 25
		resultText = "💎 **DIAMONDS!** Three diamonds!"
	case r1 == "🔔" && r2 == r1 && r3 == r1:
		mult = 10
		resultText = "🔔 **DING DING!** Three bells!"
	case r2 == r1 && r3 == r1:
		mult = 5
		resultText = "✨ Three of a kind!"
	case r1 == r2 || r2 == r3 || r1 == r3:
		mult = 1.5
		resultText = "🎯 Two of a kind!"
	default:
		mult = 0
		resultText = "💸 No match. Better luck next time!"
	}

	var newBal int
	var coinResult string
	if mult > 0 {
		win := int(float64(bet) * mult)
		newBal = modifyCoins(user.ID, user.Username, win-bet)
		coinResult = fmt.Sprintf("+**%d coins** (%.1fx multiplier)", win-bet, mult)
	} else {
		newBal = modifyCoins(user.ID, user.Username, -bet)
		coinResult = fmt.Sprintf("-**%d coins**", bet)
	}

	embed := createFunEmbed("🎰 Slot Machine", fmt.Sprintf(
		"%s\n\n%s\n%s\nBalance: **%d coins**", display, resultText, coinResult, newBal))
	respondEmbed(s, i, embed)
}

// ─── Roulette ──────────────────────────────────────────────────────────────────

func handleRoulette(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	opts := i.ApplicationCommandData().Options
	bet := int(optionInt(opts, "bet", 10))
	choice := strings.ToLower(optionString(opts, "choice", "red"))

	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have **%d coins** but tried to bet **%d**.", coins, bet)))
		return
	}

	spin := rand.Intn(37) // 0-36
	var spinColor string
	redNums := map[int]bool{1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true, 18: true, 19: true, 21: true, 23: true, 25: true, 27: true, 30: true, 32: true, 34: true, 36: true}
	if spin == 0 {
		spinColor = "green"
	} else if redNums[spin] {
		spinColor = "red"
	} else {
		spinColor = "black"
	}

	spinEmoji := "🟥"
	if spinColor == "black" {
		spinEmoji = "⬛"
	} else if spinColor == "green" {
		spinEmoji = "🟩"
	}

	var mult float64
	win := false

	// Parse choice
	if choice == "red" || choice == "black" || choice == "green" {
		if choice == spinColor {
			win = true
			switch choice {
			case "red", "black":
				mult = 2
			case "green":
				mult = 14
			}
		}
	} else {
		// numeric choice
		choiceNum := 0
		_, err := fmt.Sscanf(choice, "%d", &choiceNum)
		if err == nil && choiceNum >= 0 && choiceNum <= 36 {
			if choiceNum == spin {
				win = true
				mult = 35
			}
		} else {
			respondEmbed(s, i, createErrorEmbed("Invalid Choice", "Choose `red`, `black`, `green`, or a number `0-36`."))
			return
		}
	}

	var newBal int
	var resultText string
	if win {
		payout := int(float64(bet) * mult)
		newBal = modifyCoins(user.ID, user.Username, payout-bet)
		resultText = fmt.Sprintf("🎉 **You win!** +**%d coins** (%.0fx)\nBalance: **%d coins**", payout-bet, mult, newBal)
	} else {
		newBal = modifyCoins(user.ID, user.Username, -bet)
		resultText = fmt.Sprintf("😞 **You lose** **%d coins**.\nBalance: **%d coins**", bet, newBal)
	}

	embed := createFunEmbed("🎱 Roulette", fmt.Sprintf(
		"The ball lands on **%s %d** (%s)\nYou bet on: **%s**\n\n%s",
		spinEmoji, spin, spinColor, choice, resultText))
	respondEmbed(s, i, embed)
}

// ─── Russian Roulette ──────────────────────────────────────────────────────────

var rrSurviveTexts = []string{
	"*click* — Empty chamber. You live to see another day.",
	"The gun whispers silence. Lucky this time.",
	"Your heart nearly stopped, but the chamber was empty.",
	"Fortune smiles upon you. The barrel spins away.",
	"Sweat drips. The click echoes. You survive.",
}

var rrDeathTexts = []string{
	"💥 BANG! The bullet was waiting for you.",
	"The cylinder was not on your side today.",
	"💀 Your luck ran out this time.",
	"BANG! The universe has spoken.",
	"The chamber was loaded. Goodbye, brave soul.",
}

func handleRussianRoulette(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	opts := i.ApplicationCommandData().Options
	chambers := int(optionInt(opts, "chambers", 6))
	if chambers < 2 || chambers > 6 {
		chambers = 6
	}

	survive := rand.Intn(chambers) != 0 // 1/chambers chance of dying

	// Scale reward by risk: 6 chambers = +300, 2 chambers = +600
	reward := 300 + (6-chambers)*75
	loss := 150 + (6-chambers)*25

	var embed *discordgo.MessageEmbed
	if survive {
		coins := modifyCoins(user.ID, user.Username, reward)
		text := rrSurviveTexts[rand.Intn(len(rrSurviveTexts))]
		embed = createSuccessEmbed("🔫 Russian Roulette",
			fmt.Sprintf("%s\n\n**+%d coins!** Balance: **%d coins**", text, reward, coins))
	} else {
		coins := modifyCoins(user.ID, user.Username, -loss)
		text := rrDeathTexts[rand.Intn(len(rrDeathTexts))]
		embed = createErrorEmbed("🔫 Russian Roulette",
			fmt.Sprintf("%s\n\n**-%d coins.** Balance: **%d coins**", text, loss, coins))
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Chambers", Value: fmt.Sprintf("%d (1 bullet)", chambers), Inline: true},
		{Name: "Survival Odds", Value: fmt.Sprintf("%d/%d", chambers-1, chambers), Inline: true},
	}
	respondEmbed(s, i, embed)
}

// ─── War ───────────────────────────────────────────────────────────────────────

func handleWar(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	opts := i.ApplicationCommandData().Options
	bet := int(optionInt(opts, "bet", 10))
	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have **%d coins** but tried to bet **%d**.", coins, bet)))
		return
	}

	deck := newDeck()
	playerCard := drawCard(&deck)
	botCard := drawCard(&deck)
	pv := cardRankValue(playerCard)
	bv := cardRankValue(botCard)

	var embed *discordgo.MessageEmbed
	if pv > bv {
		newBal := modifyCoins(user.ID, user.Username, bet)
		embed = createSuccessEmbed("⚔️ War — You Win!",
			fmt.Sprintf("**Your card:** %s (%d)\n**Bot's card:** %s (%d)\n\n🎉 Higher card! **+%d coins**\nBalance: **%d coins**",
				playerCard, pv, botCard, bv, bet, newBal))
	} else if bv > pv {
		newBal := modifyCoins(user.ID, user.Username, -bet)
		embed = createErrorEmbed("⚔️ War — You Lose",
			fmt.Sprintf("**Your card:** %s (%d)\n**Bot's card:** %s (%d)\n\n😞 Lower card. **-%d coins**\nBalance: **%d coins**",
				playerCard, pv, botCard, bv, bet, newBal))
	} else {
		embed = createInfoEmbed("⚔️ War — Tie!",
			fmt.Sprintf("**Your card:** %s (%d)\n**Bot's card:** %s (%d)\n\n🤝 Tie! Bet refunded.",
				playerCard, pv, botCard, bv))
	}
	respondEmbed(s, i, embed)
}
