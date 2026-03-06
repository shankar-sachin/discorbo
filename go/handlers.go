package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if strings.HasPrefix(data.CustomID, "wordle_modal_") {
		handleWordleModal(s, i)
		return
	}
}

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	data := i.MessageComponentData()
	if strings.HasPrefix(data.CustomID, "trivia_") {
		handleTriviaComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "poll_") {
		handlePollComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "trade_") {
		handleTradeComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "tag_") {
		handleTagComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "bj_") {
		handleBJComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "poker_") {
		handlePokerComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "fish_") {
		handleFishComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "snap_") {
		handleSnapComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "g2048_") {
		handle2048Component(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "hl_") {
		handleHLComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "ttt_") {
		handleTTTComponent(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "c4_") {
		handleC4Component(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "wordle_") {
		handleWordleComponent(s, i)
		return
	}
	if data.CustomID == "truthordare_reroll" {
		handleTruthOrDareButton(s, i)
		return
	}
	if !strings.HasPrefix(data.CustomID, "maze_") {
		return
	}

	sessionMu.Lock()
	sess, ok := sessions[i.Message.ID]
	if !ok {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	if sess.UserID != user.ID {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "Only the starter can control this maze.", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}
	if sess.State.GameOver {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	dir := strings.TrimPrefix(data.CustomID, "maze_")
	_ = step(&sess.State, dir)

	if sess.State.Won {
		completionTime := int((time.Now().UnixMilli() - sess.State.StartTime) / 1000)
		saveMazeCompletion(user.ID, user.Username, sess.State.Level, completionTime, sess.State.Coins)
	}

	gameOver := sess.State.GameOver
	stateCopy := sess.State
	levelCopy := sess.Level
	if gameOver {
		delete(sessions, i.Message.ID)
	}
	sessionMu.Unlock()

	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{buildMazeEmbed(stateCopy, levelCopy, false)}}}
	if gameOver {
		resp.Data.Components = []discordgo.MessageComponent{}
	} else {
		resp.Data.Components = mazeComponents()
	}
	_ = s.InteractionRespond(i.Interaction, resp)
}

func handlePollComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	// CustomID format: poll_vote_<pollID>_<optionIndex>
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 4 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	// Extract option index (last part)
	optIdx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	pollMu.Lock()
	sess, ok := pollSessions[i.Message.ID]
	if !ok || time.Now().UnixMilli() > sess.ExpiresAt {
		pollMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This poll has expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// Check double voting
	if prevIdx, voted := sess.Voters[user.ID]; voted {
		pollMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("You already voted for option %d. You can only vote once!", prevIdx+1),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if optIdx < 0 || optIdx >= len(sess.Votes) {
		pollMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	sess.Votes[optIdx]++
	sess.Voters[user.ID] = optIdx
	pollMu.Unlock()

	// Look up creator username
	creator := "unknown"
	if u, err := s.User(sess.CreatorID); err == nil && u != nil {
		creator = u.Username
	}

	embed := buildPollEmbedFromSession(sess, creator)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func handleTriviaComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 4 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	token := parts[1] + "_" + parts[2]
	selected, err := strconv.Atoi(parts[3])
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	triviaMu.Lock()
	sess, ok := triviaSessions[token]
	if ok && time.Now().UnixMilli() > sess.ExpiresAt {
		delete(triviaSessions, token)
		ok = false
	}
	if !ok {
		triviaMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This trivia question has expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if user.ID != sess.UserID {
		triviaMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Only the original player can answer.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	delete(triviaSessions, token)
	triviaMu.Unlock()

	scores := map[string]triviaScore{}
	_ = readData("trivia-scores.json", &scores)
	sc := scores[user.ID]
	sc.Username = user.Username
	sc.Total++
	correct := selected == sess.CorrectIndex
	if correct {
		sc.Correct++
		sc.Score += sess.Points
	}
	scores[user.ID] = sc
	_ = writeData("trivia-scores.json", scores)

	content := fmt.Sprintf("Incorrect. Correct answer: %s\nScore: %d", sess.CorrectAnswer, sc.Score)
	if correct {
		content = fmt.Sprintf("Correct! +%d points\nAnswer: %s\nScore: %d", sess.Points, sess.CorrectAnswer, sc.Score)
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil || m.Author.Bot {
		return
	}

	// Check auto-moderation rules
	CheckAutoMod(s, m)

	// Award XP for leveling system
	awardXP(s, m)

	// Check for DMs or bot mentions and handle with AI
	isDM := m.GuildID == ""
	isMention := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			isMention = true
			break
		}
	}

	if isDM || isMention {
		handleAIChat(s, m, isDM)
		return
	}

	// Handle AFK system
	afkMap := map[string]afkStatus{}
	if err := readData("afk-users.json", &afkMap); err != nil {
		return
	}
	changed := false
	if st, ok := afkMap[m.Author.ID]; ok {
		delete(afkMap, m.Author.ID)
		changed = true
		delta := time.Since(time.UnixMilli(st.Timestamp)).Round(time.Second)
		embed := createSuccessEmbed("👋 Welcome Back!", fmt.Sprintf("<@%s> is no longer AFK.", m.Author.ID))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "AFK Duration", Value: delta.String(), Inline: true},
		}
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
	}
	for _, u := range m.Mentions {
		if u == nil {
			continue
		}
		if st, ok := afkMap[u.ID]; ok {
			delta := time.Since(time.UnixMilli(st.Timestamp)).Round(time.Minute)
			embed := createWarningEmbed(fmt.Sprintf("💤 %s is AFK", u.Username), st.Reason)
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "AFK Since", Value: delta.String() + " ago", Inline: true},
			}
			_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		}
	}
	if changed {
		_ = writeData("afk-users.json", afkMap)
	}
}

func handleAIChat(s *discordgo.Session, m *discordgo.MessageCreate, isDM bool) {
	_ = s.ChannelTyping(m.ChannelID)

	// Strip bot mention from message
	cleanMessage := m.Content
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			cleanMessage = strings.ReplaceAll(cleanMessage, "<@"+user.ID+">", "")
			cleanMessage = strings.ReplaceAll(cleanMessage, "<@!"+user.ID+">", "")
		}
	}
	cleanMessage = strings.TrimSpace(cleanMessage)

	response := getLocalAIResponse(cleanMessage)
	embed := createInfoEmbed("🤖 Discorbo", response)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleTradeComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, "_")
	if len(parts) < 3 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	action := parts[1]
	tradeID := strings.Join(parts[2:], "_")

	tradeMu.Lock()
	sess, ok := tradeSessions[i.Message.ID]
	if !ok {
		tradeMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Trade session not found or expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if sess.TradeID != tradeID {
		tradeMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	switch action {
	case "accept":
		// Target must accept
		if user.ID != sess.TargetID {
			tradeMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Only the trade target can accept.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		// Capture session info before unlocking
		initiatorID := sess.InitiatorID
		targetID := sess.TargetID
		initOffer := sess.InitOffer
		targetOffer := sess.TargetOffer
		tradeID := sess.TradeID
		delete(tradeSessions, i.Message.ID)
		tradeMu.Unlock()

		// Load economy data
		economyUsers := map[string]economyUser{}
		_ = readData("economy-users.json", &economyUsers)

		initUser := economyUsers[initiatorID]
		if initUser.Inventory == nil {
			initUser.Inventory = []inventoryItem{}
		}
		targUser := economyUsers[targetID]
		if targUser.Inventory == nil {
			targUser.Inventory = []inventoryItem{}
		}

		// Transfer coins
		initUser.Coins -= initOffer.Coins
		initUser.Coins += targetOffer.Coins
		targUser.Coins -= targetOffer.Coins
		targUser.Coins += initOffer.Coins

		// Transfer items from initiator to target
		for _, itemID := range initOffer.ItemIDs {
			// Remove from initiator
			for idx, inv := range initUser.Inventory {
				if inv.ItemID == itemID && inv.Quantity > 0 {
					initUser.Inventory[idx].Quantity--
					if initUser.Inventory[idx].Quantity == 0 {
						initUser.Inventory = append(initUser.Inventory[:idx], initUser.Inventory[idx+1:]...)
					}
					break
				}
			}
			// Add to target
			found := false
			for idx, inv := range targUser.Inventory {
				if inv.ItemID == itemID {
					targUser.Inventory[idx].Quantity++
					found = true
					break
				}
			}
			if !found {
				targUser.Inventory = append(targUser.Inventory, inventoryItem{ItemID: itemID, Quantity: 1, PurchasedAt: time.Now().Unix()})
			}
		}

		// Transfer items from target to initiator
		for _, itemID := range targetOffer.ItemIDs {
			for idx, inv := range targUser.Inventory {
				if inv.ItemID == itemID && inv.Quantity > 0 {
					targUser.Inventory[idx].Quantity--
					if targUser.Inventory[idx].Quantity == 0 {
						targUser.Inventory = append(targUser.Inventory[:idx], targUser.Inventory[idx+1:]...)
					}
					break
				}
			}
			found := false
			for idx, inv := range initUser.Inventory {
				if inv.ItemID == itemID {
					initUser.Inventory[idx].Quantity++
					found = true
					break
				}
			}
			if !found {
				initUser.Inventory = append(initUser.Inventory, inventoryItem{ItemID: itemID, Quantity: 1, PurchasedAt: time.Now().Unix()})
			}
		}

		// Record trade history
		now := time.Now().Unix()
		initUser.TradeHistory = append(initUser.TradeHistory, tradeRecord{
			TradeID:   tradeID,
			OtherUser: targetID,
			GaveCoins: initOffer.Coins,
			GaveItems: initOffer.ItemIDs,
			GotCoins:  targetOffer.Coins,
			GotItems:  targetOffer.ItemIDs,
			Timestamp: now,
		})
		targUser.TradeHistory = append(targUser.TradeHistory, tradeRecord{
			TradeID:   tradeID,
			OtherUser: initiatorID,
			GaveCoins: targetOffer.Coins,
			GaveItems: targetOffer.ItemIDs,
			GotCoins:  initOffer.Coins,
			GotItems:  initOffer.ItemIDs,
			Timestamp: now,
		})

		economyUsers[initiatorID] = initUser
		economyUsers[targetID] = targUser
		_ = writeData("economy-users.json", economyUsers)

		// Build summary
		summaryLines := fmt.Sprintf("<@%s> gave: %s", initiatorID, coinDisplay(initOffer.Coins))
		if len(initOffer.ItemIDs) > 0 {
			summaryLines += fmt.Sprintf(" + %d item(s)", len(initOffer.ItemIDs))
		}
		summaryLines += fmt.Sprintf("\n<@%s> gave: %s", targetID, coinDisplay(targetOffer.Coins))
		if len(targetOffer.ItemIDs) > 0 {
			summaryLines += fmt.Sprintf(" + %d item(s)", len(targetOffer.ItemIDs))
		}

		embed := createSuccessEmbed("✅ Trade Complete!", summaryLines)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})

	case "decline":
		if user.ID != sess.TargetID {
			tradeMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Only the trade target can decline.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		delete(tradeSessions, i.Message.ID)
		tradeMu.Unlock()

		embed := createErrorEmbed("Trade Declined", "The trade request was declined.")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})

	default:
		tradeMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	}
}

func handleTagComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	data := i.MessageComponentData()
	parts := strings.Split(data.CustomID, "_")
	if len(parts) < 2 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	action := parts[1]

	sessionMu.Lock()
	sess, ok := tagSessions[i.Message.ID]
	if !ok {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Game session not found or expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// Determine current player
	currentPlayerID := sess.Player1ID
	if sess.CurrentTurn == 1 {
		currentPlayerID = sess.Player2ID
	}

	// Verify it's the current player's turn
	if user.ID != currentPlayerID {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "It's not your turn!", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	switch action {
	case "accept":
		// Accept challenge
		if user.ID != sess.Player2ID {
			sessionMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Only the challenged player can accept.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		// Generate board and start game
		sess.Board = generateTagBoard()
		sess.Player1X = 0
		sess.Player1Y = 0
		sess.Player2X = 4
		sess.Player2Y = 4
		sess.StartTime = time.Now().UnixMilli()
		sess.CurrentTurn = 0
		sess.Moves = 0
		sess.MaxMoves = 20

		embed := buildTagEmbed(sess)
		buttons := buildTagButtons(sess.SessionID)

		sessionMu.Unlock()

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: buttons},
		})

	case "decline":
		if user.ID != sess.Player2ID {
			sessionMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Only the challenged player can decline.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		delete(tagSessions, i.Message.ID)
		sessionMu.Unlock()

		embed := createErrorEmbed("Challenge Declined", "The tag challenge was declined.")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{}},
		})

	case "up", "down", "left", "right", "stay":
		// Process move
		moved := processTagMove(sess, action)
		if !moved {
			sessionMu.Unlock()
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Invalid move!", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}

		sess.Moves++

		// Check win condition
		gameOver := false
		winner := ""
		if sess.Player1X == sess.Player2X && sess.Player1Y == sess.Player2Y {
			gameOver = true
			if sess.CurrentTurn == 0 {
				winner = sess.Player1Name
			} else {
				winner = sess.Player2Name
			}
		} else if sess.Moves >= sess.MaxMoves {
			gameOver = true
			winner = "draw"
		}

		if !gameOver {
			// Switch turn
			sess.CurrentTurn = 1 - sess.CurrentTurn
		}

		embed := buildTagEmbed(sess)
		var buttons []discordgo.MessageComponent

		if gameOver {
			delete(tagSessions, i.Message.ID)
			sessionMu.Unlock()

			// Award coins
			if winner != "draw" {
				remainingMoves := sess.MaxMoves - sess.Moves
				coins := 50 + (remainingMoves * 10)

				winnerID := sess.Player1ID
				loserID := sess.Player2ID
				if winner == sess.Player2Name {
					winnerID = sess.Player2ID
					loserID = sess.Player1ID
				}

				// Update economy
				ecoUsers := map[string]economyUser{}
				_ = readData("economy-users.json", &ecoUsers)
				eu := ecoUsers[winnerID]
				eu.Coins += coins
				eu.Username = winner
				ecoUsers[winnerID] = eu
				_ = writeData("economy-users.json", ecoUsers)

				// Update tag stats
				tagStatsMap := map[string]tagStats{}
				_ = readData("tag-stats.json", &tagStatsMap)

				ws := tagStatsMap[winnerID]
				ws.Username = winner
				ws.Wins++
				ws.TotalGames++
				ws.CoinsEarned += coins
				tagStatsMap[winnerID] = ws

				loserName := sess.Player2Name
				if winner == sess.Player2Name {
					loserName = sess.Player1Name
				}
				ls := tagStatsMap[loserID]
				ls.Username = loserName
				ls.Losses++
				ls.TotalGames++
				tagStatsMap[loserID] = ls

				_ = writeData("tag-stats.json", tagStatsMap)

				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "🏆 Winner",
					Value:  fmt.Sprintf("%s wins and earns %s!", winner, coinDisplay(coins)),
					Inline: false,
				})
			} else {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "⚖️ Draw",
					Value:  fmt.Sprintf("Maximum moves reached! Both players earn %s.", coinDisplay(10)),
					Inline: false,
				})

				// Give both players 10 coins
				ecoUsers := map[string]economyUser{}
				_ = readData("economy-users.json", &ecoUsers)
				for _, pid := range []string{sess.Player1ID, sess.Player2ID} {
					eu := ecoUsers[pid]
					eu.Coins += 10
					ecoUsers[pid] = eu
				}
				_ = writeData("economy-users.json", ecoUsers)
			}
		} else {
			buttons = buildTagButtons(sess.SessionID)
			sessionMu.Unlock()
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: buttons},
		})

	default:
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
	}
}
