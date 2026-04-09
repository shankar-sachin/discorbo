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

func handleCasinoCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	switch sub.Name {
	case "blackjack":
		handleBlackjack(s, i, sub.Options)
	case "slots":
		handleSlots(s, i, sub.Options)
	case "roulette":
		handleRoulette(s, i, sub.Options)
	case "russian-roulette":
		handleRussianRoulette(s, i, sub.Options)
	case "poker":
		handlePoker(s, i, sub.Options)
	}
}

// ─── Blackjack ─────────────────────────────────────────────────────────────────

func handleBlackjack(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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
		embed := createCasinoEmbed("🃏 Blackjack!", fmt.Sprintf(
			"**Your hand:** %s (21)\n**Dealer:** %s %s\n\n🎉 **BLACKJACK!** You win +%s!\nBalance: %s",
			handStr(playerHand), dealerHand[0], "??", coinDisplay(payout), coinDisplay(newBal)))
		respondEmbed(s, i, embed)
		return
	}

	// Deduct bet upfront
	modifyCoins(user.ID, user.Username, -bet)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     buildBJEmbeds(sess, false),
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

<<<<<<< HEAD
// bjGalleryURL removed — cards are now displayed as text, not images.

=======
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
func buildBJEmbeds(sess *bjSession, reveal bool) []*discordgo.MessageEmbed {
	playerCards := renderHand(sess.PlayerHand)
	playerVal := handValue(sess.PlayerHand)

	var dealerCards string
	var dealerValStr string
	if reveal {
		dealerCards = renderHand(sess.DealerHand)
		dealerValStr = fmt.Sprintf(" (%d)", handValue(sess.DealerHand))
	} else {
		dealerCards = renderHandHidden(sess.DealerHand)
		dealerValStr = ""
	}
<<<<<<< HEAD
=======

>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
	embed := &discordgo.MessageEmbed{
		Title: "🃏 Blackjack",
		Color: ColorCasino,
		Fields: []*discordgo.MessageEmbedField{
<<<<<<< HEAD
			{Name: fmt.Sprintf("Your Hand (%d)", handValue(sess.PlayerHand)), Value: handStr(sess.PlayerHand), Inline: true},
			{Name: "Dealer's Hand" + dealerVal, Value: dealerStr, Inline: true},
			{Name: "💰 Bet", Value: fmt.Sprintf("%d coins", sess.Bet), Inline: false},
=======
			{Name: fmt.Sprintf("🎯 Your Hand (%d)", playerVal), Value: playerCards, Inline: false},
			{Name: "🏠 Dealer" + dealerValStr, Value: dealerCards, Inline: false},
			{Name: "💰 Bet", Value: coinDisplay(sess.Bet), Inline: true},
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
		},
		Footer:    embedFooter("🎰 Casino"),
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return []*discordgo.MessageEmbed{embed}
}

func buildBJButtons(firstTurn bool) []discordgo.MessageComponent {
	btns := []discordgo.MessageComponent{
		discordgo.Button{Label: "🎯 Hit", Style: discordgo.PrimaryButton, CustomID: "bj_hit"},
		discordgo.Button{Label: "✋ Stand", Style: discordgo.SecondaryButton, CustomID: "bj_stand"},
	}
	if firstTurn {
		btns = append(btns, discordgo.Button{Label: "⬇️ Double Down", Style: discordgo.SuccessButton, CustomID: "bj_double"})
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
			embeds := buildBJEmbeds(sess, true)
			embeds[0].Color = ColorRed
			embeds[0].Description = fmt.Sprintf("💥 **Bust!** You went over 21.\nYou lose %s.\nBalance: %s", coinDisplay(sess.Bet), coinDisplay(coins))
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{Embeds: embeds, Components: []discordgo.MessageComponent{}},
			})
			return
		}
		gameMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: buildBJEmbeds(sess, false), Components: buildBJButtons(false)},
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
		embeds := buildBJEmbeds(sess, true)
		if pv > 21 {
			embeds[0].Color = ColorRed
			coins, _ := getCoins(user.ID)
			result = fmt.Sprintf("💥 **Bust!** You lose %s.", coinDisplay(sess.Bet))
			newBal = coins
		} else if dv > 21 || pv > dv {
			newBal = modifyCoins(user.ID, user.Username, sess.Bet*2)
			result = fmt.Sprintf("🎉 **You win!** +%s!", coinDisplay(sess.Bet))
			embeds[0].Color = ColorGreen
		} else if pv == dv {
			newBal = modifyCoins(user.ID, user.Username, sess.Bet)
			result = "🤝 **Push!** Bet refunded."
			embeds[0].Color = ColorGray
		} else {
			coins, _ := getCoins(user.ID)
			result = fmt.Sprintf("😞 **Dealer wins.** You lose %s.", coinDisplay(sess.Bet))
			embeds[0].Color = ColorRed
			newBal = coins
		}
		embeds[0].Description = result + fmt.Sprintf("\nBalance: %s", coinDisplay(newBal))
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: embeds, Components: []discordgo.MessageComponent{}},
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

func handleSlots(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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

	r1, r2, r3 := weightedSlot(), weightedSlot(), weightedSlot()

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
		coinResult = fmt.Sprintf("+%s (%.1fx multiplier)", coinDisplay(win-bet), mult)
	} else {
		newBal = modifyCoins(user.ID, user.Username, -bet)
		coinResult = fmt.Sprintf("-%s", coinDisplay(bet))
	}

<<<<<<< HEAD
	embed := createFunEmbed("🎰 Slot Machine", fmt.Sprintf("[ %s | %s | %s ]", r1, r2, r3))
	embed.Color = 0xF1C40F
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Result", Value: resultText, Inline: false},
		{Name: "Coins", Value: coinResult, Inline: true},
		{Name: "Balance", Value: fmt.Sprintf("%d coins", newBal), Inline: true},
	}
=======
	embed := createCasinoEmbed("🎰 Slot Machine", fmt.Sprintf(
		"**━━━━━━━━━━━━━━━**\n  %s\n**━━━━━━━━━━━━━━━**\n\n%s\n%s\n💳 Balance: %s", display, resultText, coinResult, coinDisplay(newBal)))
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
	respondEmbed(s, i, embed)
}

// ─── Roulette ──────────────────────────────────────────────────────────────────

func handleRoulette(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}
	bet := int(optionInt(opts, "bet", 10))
	choice := strings.ToLower(optionString(opts, "choice", "red"))

	coins, _ := getCoins(user.ID)
	if coins < bet {
		respondEmbed(s, i, createErrorEmbed("Insufficient Funds", fmt.Sprintf("You only have %s but tried to bet %s.", coinDisplay(coins), coinDisplay(bet))))
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
	if win {
		payout := int(float64(bet) * mult)
		newBal = modifyCoins(user.ID, user.Username, payout-bet)
<<<<<<< HEAD
	} else {
		newBal = modifyCoins(user.ID, user.Username, -bet)
	}

	resultLabel := "😞 Lose"
	if win {
		resultLabel = "🎉 Win"
	}
	embed := createFunEmbed("🎱 Roulette", fmt.Sprintf("The ball lands on %s **%d** (%s)", spinEmoji, spin, spinColor))
	embed.Color = 0xF1C40F
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Your Bet", Value: fmt.Sprintf("**%s**", choice), Inline: true},
		{Name: "Result", Value: resultLabel, Inline: true},
		{Name: "Balance", Value: fmt.Sprintf("%d coins", newBal), Inline: true},
	}
=======
		resultText = fmt.Sprintf("🎉 **You win!** +%s (%.0fx)\nBalance: %s", coinDisplay(payout-bet), mult, coinDisplay(newBal))
	} else {
		newBal = modifyCoins(user.ID, user.Username, -bet)
		resultText = fmt.Sprintf("😞 **You lose** %s.\nBalance: %s", coinDisplay(bet), coinDisplay(newBal))
	}

	embed := createCasinoEmbed("🎱 Roulette", fmt.Sprintf(
		"🎡 The wheel spins...\n\nThe ball lands on %s\n\n🎲 You bet on: **%s**\n\n%s",
		renderRouletteResult(spin), choice, resultText))
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
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

func handleRussianRoulette(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondError(s, i, "Error", "Unable to identify user.")
		return
	}
	chambers := int(optionInt(opts, "chambers", 6))
	if chambers < 2 || chambers > 6 {
		chambers = 6
	}

	survive := rand.Intn(chambers) != 0 // 1/chambers chance of dying

	// Scale reward by risk: 6 chambers = +300, 2 chambers = +600
	reward := 300 + (6-chambers)*75
	loss := 150 + (6-chambers)*25

	var embed *discordgo.MessageEmbed
	chamberDisplay := renderChambers(chambers, chambers-1)
	if survive {
		coins := modifyCoins(user.ID, user.Username, reward)
		text := rrSurviveTexts[rand.Intn(len(rrSurviveTexts))]
<<<<<<< HEAD
		embed = createSuccessEmbed("🔫 Russian Roulette", text)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Chambers", Value: fmt.Sprintf("%d (1 bullet)", chambers), Inline: true},
			{Name: "Odds", Value: fmt.Sprintf("%d/%d survival", chambers-1, chambers), Inline: true},
			{Name: "Coins", Value: fmt.Sprintf("+%d → **%d**", reward, coins), Inline: false},
		}
	} else {
		coins := modifyCoins(user.ID, user.Username, -loss)
		text := rrDeathTexts[rand.Intn(len(rrDeathTexts))]
		embed = createErrorEmbed("🔫 Russian Roulette", text)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Chambers", Value: fmt.Sprintf("%d (1 bullet)", chambers), Inline: true},
			{Name: "Odds", Value: fmt.Sprintf("%d/%d survival", chambers-1, chambers), Inline: true},
			{Name: "Coins", Value: fmt.Sprintf("-%d → **%d**", loss, coins), Inline: false},
		}
=======
		embed = createSuccessEmbed("🔫 Russian Roulette",
			fmt.Sprintf("%s\n\n%s\n\n💰 +%s! Balance: %s", chamberDisplay, text, coinDisplay(reward), coinDisplay(coins)))
	} else {
		coins := modifyCoins(user.ID, user.Username, -loss)
		text := rrDeathTexts[rand.Intn(len(rrDeathTexts))]
		embed = createErrorEmbed("🔫 Russian Roulette",
			fmt.Sprintf("%s\n\n%s\n\n💸 -%s. Balance: %s", chamberDisplay, text, coinDisplay(loss), coinDisplay(coins)))
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Chambers", Value: fmt.Sprintf("%d (1 bullet)", chambers), Inline: true},
		{Name: "Survival Odds", Value: fmt.Sprintf("%d/%d", chambers-1, chambers), Inline: true},
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
	}
	respondEmbed(s, i, embed)
}

// ─── War ───────────────────────────────────────────────────────────────────────

func handleWar(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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

	deck := newDeck()
	playerCard := drawCard(&deck)
	botCard := drawCard(&deck)
	pv := cardRankValue(playerCard)
	bv := cardRankValue(botCard)

	var embed *discordgo.MessageEmbed
	warDisplay := renderWarCards(playerCard, botCard)
	if pv > bv {
		newBal := modifyCoins(user.ID, user.Username, bet)
<<<<<<< HEAD
		embed = createSuccessEmbed("⚔️ War — You Win!", fmt.Sprintf("Higher card! **+%d coins**", bet))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Your Card", Value: fmt.Sprintf("%s (value %d)", playerCard, pv), Inline: true},
			{Name: "Bot's Card", Value: fmt.Sprintf("%s (value %d)", botCard, bv), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d coins", newBal), Inline: false},
		}
	} else if bv > pv {
		newBal := modifyCoins(user.ID, user.Username, -bet)
		embed = createErrorEmbed("⚔️ War — You Lose", fmt.Sprintf("Lower card. **-%d coins**", bet))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Your Card", Value: fmt.Sprintf("%s (value %d)", playerCard, pv), Inline: true},
			{Name: "Bot's Card", Value: fmt.Sprintf("%s (value %d)", botCard, bv), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d coins", newBal), Inline: false},
		}
	} else {
		embed = createInfoEmbed("⚔️ War — Tie!", "Same value — bet refunded.")
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Your Card", Value: fmt.Sprintf("%s (value %d)", playerCard, pv), Inline: true},
			{Name: "Bot's Card", Value: fmt.Sprintf("%s (value %d)", botCard, bv), Inline: true},
		}
=======
		embed = createSuccessEmbed("⚔️ War — You Win!",
			fmt.Sprintf("%s\n\n🎉 Higher card! +%s\n💳 Balance: %s",
				warDisplay, coinDisplay(bet), coinDisplay(newBal)))
	} else if bv > pv {
		newBal := modifyCoins(user.ID, user.Username, -bet)
		embed = createErrorEmbed("⚔️ War — You Lose",
			fmt.Sprintf("%s\n\n😞 Lower card. -%s\n💳 Balance: %s",
				warDisplay, coinDisplay(bet), coinDisplay(newBal)))
	} else {
		embed = createInfoEmbed("⚔️ War — Tie!",
			fmt.Sprintf("%s\n\n🤝 Tie! Bet refunded.",
				warDisplay))
>>>>>>> eea998cd24a8dcc9df10cac370475191eed5c8a5
	}
	respondEmbed(s, i, embed)
}
