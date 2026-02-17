package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
	if _, ok := afkMap[m.Author.ID]; ok {
		delete(afkMap, m.Author.ID)
		changed = true
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s> is no longer AFK.", m.Author.ID))
	}
	for _, u := range m.Mentions {
		if u == nil {
			continue
		}
		if st, ok := afkMap[u.ID]; ok {
			delta := time.Since(time.UnixMilli(st.Timestamp)).Round(time.Minute)
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s is AFK (%s ago): %s", u.Username, delta.String(), st.Reason))
		}
	}
	if changed {
		_ = writeData("afk-users.json", afkMap)
	}
}

func handleAIChat(s *discordgo.Session, m *discordgo.MessageCreate, isDM bool) {
	// Show typing indicator
	_ = s.ChannelTyping(m.ChannelID)

	// Remove bot mention from message if present
	cleanMessage := m.Content
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			cleanMessage = strings.Replace(cleanMessage, "<@"+user.ID+">", "", -1)
			cleanMessage = strings.Replace(cleanMessage, "<@!"+user.ID+">", "", -1)
		}
	}
	cleanMessage = strings.TrimSpace(cleanMessage)

	if cleanMessage == "" {
		embed := createInfoEmbed("👋 Hello!", "I'm Discorbo! How can I help you?\n\nTry asking me a question or use `/help` to see my commands!")
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Get AI response
	response, err := getAIChatResponse(m.Author.ID, m.Author.Username, cleanMessage, isDM)
	if err != nil {
		log.Printf("AI chat error: %v", err)
		embed := createErrorEmbed("Oops!", "I'm having trouble thinking right now. Try using `/help` to see my commands!")
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Send response as embed
	embed := createInfoEmbed("🤖 Discorbo", response)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
