package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Info utility commands
func handleInfoUtility(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name
	opts := i.ApplicationCommandData().Options

	switch cmd {
	case "ping":
		respondText(s, i, fmt.Sprintf("Pong! Gateway ping: %dms", s.HeartbeatLatency().Milliseconds()))
	case "help":
		handleHelp(s, i, opts)
	case "avatar":
		handleAvatar(s, i, opts)
	case "userinfo":
		handleUserInfo(s, i, opts)
	case "serverinfo":
		handleServerInfo(s, i)
	case "channel-info":
		handleChannelInfo(s, i, opts)
	case "role-info":
		handleRoleInfo(s, i, opts)
	case "stats":
		handleStats(s, i)
	}
}

// Tool utility commands
func handleToolUtility(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name
	opts := i.ApplicationCommandData().Options

	switch cmd {
	case "calc":
		handleCalc(s, i, opts)
	case "translate":
		handleTranslate(s, i, opts)
	case "poll":
		handlePoll(s, i, opts)
	case "timer":
		handleTimer(s, i, opts)
	}
}

// Data utility commands
func handleDataUtility(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name
	opts := i.ApplicationCommandData().Options

	switch cmd {
	case "remind":
		handleRemind(s, i, opts)
	case "reminders":
		handleReminders(s, i, opts)
	case "afk":
		handleAFK(s, i, opts)
	case "clear-my-data":
		handleClearMyData(s, i, opts)
	}
}

func handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	category := strings.ToLower(optionString(opts, "category", "all"))
	all := allCommands()
	funSet := map[string]bool{}
	for _, c := range funCommands() {
		funSet[c.Name] = true
	}

	funLines := []string{}
	utilLines := []string{}
	for _, c := range all {
		line := fmt.Sprintf("`/%s` - %s", c.Name, c.Description)
		if funSet[c.Name] {
			funLines = append(funLines, line)
		} else {
			utilLines = append(utilLines, line)
		}
	}
	sort.Strings(funLines)
	sort.Strings(utilLines)

	embed := &discordgo.MessageEmbed{
		Title:       "Discorbo Commands",
		Description: "Go runtime command list",
		Color:       0x5865F2,
	}
	if category == "all" || category == "fun" {
		value := "No fun commands found."
		if len(funLines) > 0 {
			value = strings.Join(funLines, "\n")
			if len(value) > 1024 {
				value = value[:1021] + "..."
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Fun & Games", Value: value})
	}
	if category == "all" || category == "utility" {
		value := "No utility commands found."
		if len(utilLines) > 0 {
			value = strings.Join(utilLines, "\n")
			if len(value) > 1024 {
				value = value[:1021] + "..."
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Utility", Value: value})
	}
	respondEmbed(s, i, embed)
}

func handleAvatar(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := interactionUser(i)
	if u == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	if picked := optionUser(opts, "user"); picked != nil {
		u = picked
	}
	url := u.AvatarURL("1024")
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's Avatar", u.Username),
		Description: fmt.Sprintf("[Open image](%s)", url),
		Color:       0x5865F2,
	}
	if url != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: url}
	}
	respondEmbed(s, i, embed)
}

func handleUserInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := interactionUser(i)
	if u == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	if picked := optionUser(opts, "user"); picked != nil {
		u = picked
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("User Information: %s", u.String()),
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Username", Value: u.Username, Inline: true},
			{Name: "User ID", Value: u.ID, Inline: true},
			{Name: "Bot Account", Value: strconv.FormatBool(u.Bot), Inline: true},
			{Name: "Created", Value: fmt.Sprintf("<t:%d:R>", snowflakeUnix(u.ID)), Inline: true},
		},
	}
	if avatar := u.AvatarURL("1024"); avatar != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: avatar}
	}
	if i.GuildID != "" {
		if m, err := s.GuildMember(i.GuildID, u.ID); err == nil && m != nil {
			if !m.JoinedAt.IsZero() {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Joined Server", Value: fmt.Sprintf("<t:%d:R>", m.JoinedAt.Unix()), Inline: true})
			}
			if m.Nick != "" {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Nickname", Value: m.Nick, Inline: true})
			}
		}
	}
	respondEmbed(s, i, embed)
}

func handleServerInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		respondText(s, i, "This command only works in a server.")
		return
	}
	g, err := s.Guild(i.GuildID)
	if err != nil || g == nil {
		respondText(s, i, "Failed to fetch server info.")
		return
	}
	channels, _ := s.GuildChannels(i.GuildID)
	textChannels, voiceChannels, categories := 0, 0, 0
	for _, c := range channels {
		switch c.Type {
		case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
			textChannels++
		case discordgo.ChannelTypeGuildVoice, discordgo.ChannelTypeGuildStageVoice:
			voiceChannels++
		case discordgo.ChannelTypeGuildCategory:
			categories++
		}
	}
	owner := "Unknown"
	if g.OwnerID != "" {
		owner = "<@" + g.OwnerID + ">"
	}
	embed := &discordgo.MessageEmbed{
		Title: g.Name,
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Server ID", Value: g.ID, Inline: true},
			{Name: "Owner", Value: owner, Inline: true},
			{Name: "Created", Value: fmt.Sprintf("<t:%d:R>", snowflakeUnix(g.ID)), Inline: true},
			{Name: "Members", Value: strconv.Itoa(g.MemberCount), Inline: true},
			{Name: "Channels", Value: fmt.Sprintf("Text: %d\nVoice: %d\nCategories: %d", textChannels, voiceChannels, categories), Inline: true},
			{Name: "Roles", Value: strconv.Itoa(len(g.Roles)), Inline: true},
		},
	}
	if icon := g.IconURL("1024"); icon != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: icon}
	}
	respondEmbed(s, i, embed)
}

func handleChannelInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID
	for _, o := range opts {
		if o.Name == "channel" && o.Value != nil {
			if v, ok := o.Value.(string); ok && v != "" {
				channelID = v
			}
		}
	}
	ch, err := s.Channel(channelID)
	if err != nil || ch == nil {
		respondText(s, i, "Failed to fetch channel.")
		return
	}
	topic := ch.Topic
	if topic == "" {
		topic = "No topic."
	}
	if len(topic) > 1024 {
		topic = topic[:1021] + "..."
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Channel Information: #%s", ch.Name),
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Channel ID", Value: ch.ID, Inline: true},
			{Name: "Type", Value: fmt.Sprintf("%d", ch.Type), Inline: true},
			{Name: "Created", Value: fmt.Sprintf("<t:%d:R>", snowflakeUnix(ch.ID)), Inline: true},
			{Name: "Topic", Value: topic, Inline: false},
		},
	}
	respondEmbed(s, i, embed)
}

func handleRoleInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.GuildID == "" {
		respondText(s, i, "This command only works in a server.")
		return
	}
	roleID := ""
	for _, o := range opts {
		if o.Name == "role" && o.Value != nil {
			if v, ok := o.Value.(string); ok {
				roleID = v
			}
		}
	}
	if roleID == "" {
		respondText(s, i, "Role is required.")
		return
	}
	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		respondText(s, i, "Failed to fetch roles.")
		return
	}
	var role *discordgo.Role
	for _, r := range roles {
		if r.ID == roleID {
			role = r
			break
		}
	}
	if role == nil {
		respondText(s, i, "Role not found.")
		return
	}
	color := 0x5865F2
	if role.Color != 0 {
		color = role.Color
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Role Information: %s", role.Name),
		Color: color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Role ID", Value: role.ID, Inline: true},
			{Name: "Color", Value: fmt.Sprintf("#%06X", role.Color), Inline: true},
			{Name: "Mentionable", Value: strconv.FormatBool(role.Mentionable), Inline: true},
			{Name: "Hoisted", Value: strconv.FormatBool(role.Hoist), Inline: true},
			{Name: "Created", Value: fmt.Sprintf("<t:%d:R>", snowflakeUnix(role.ID)), Inline: true},
			{Name: "Mention", Value: "<@&" + role.ID + ">", Inline: true},
		},
	}
	respondEmbed(s, i, embed)
}

func handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	uptime := time.Since(botStartedAt).Round(time.Second)
	heap := &runtime.MemStats{}
	runtime.ReadMemStats(heap)
	guilds := len(s.State.Guilds)
	users := 0
	for _, g := range s.State.Guilds {
		users += g.MemberCount
	}
	embed := &discordgo.MessageEmbed{
		Title: "Bot Statistics",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Servers", Value: strconv.Itoa(guilds), Inline: true},
			{Name: "Users", Value: strconv.Itoa(users), Inline: true},
			{Name: "Commands", Value: strconv.Itoa(len(allCommands())), Inline: true},
			{Name: "Uptime", Value: uptime.String(), Inline: true},
			{Name: "Memory", Value: fmt.Sprintf("%.2f MB", float64(heap.HeapAlloc)/(1024*1024)), Inline: true},
			{Name: "Go", Value: runtime.Version(), Inline: true},
		},
	}
	respondEmbed(s, i, embed)
}

func handleCalc(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	expr := optionString(opts, "expression", "")
	result, err := evalMath(expr)
	if err != nil {
		respondText(s, i, "Invalid expression. Example: `5*5+10` or `sqrt`-style functions are not supported in Go mode.")
		return
	}
	respondText(s, i, fmt.Sprintf("Expression: `%s`\nResult: `%s`", expr, result))
}

func handleTranslate(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	deferReply(s, i)
	text := optionString(opts, "text", "")
	toLang := strings.TrimSpace(optionString(opts, "to", ""))
	fromLang := strings.TrimSpace(optionString(opts, "from", "auto"))
	if text == "" || toLang == "" {
		editDeferredText(s, i, "Text and target language are required.")
		return
	}
	payload := map[string]string{
		"q":      text,
		"source": fromLang,
		"target": strings.ToLower(toLang),
		"format": "text",
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "https://libretranslate.com/translate", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 7 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		editDeferredText(s, i, "Translation service is unavailable.")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		TranslatedText   string `json:"translatedText"`
		DetectedLanguage struct {
			Language string `json:"language"`
		} `json:"detectedLanguage"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.TranslatedText == "" {
		editDeferredText(s, i, "Failed to translate text.")
		return
	}
	detected := out.DetectedLanguage.Language
	if detected == "" {
		detected = fromLang
	}
	editDeferredText(s, i, fmt.Sprintf("Original (%s): %s\n\nTranslated (%s): %s", strings.ToUpper(detected), text, strings.ToUpper(toLang), out.TranslatedText))
}

func handlePoll(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	question := optionString(opts, "question", "")
	if question == "" {
		respondText(s, i, "Question is required.")
		return
	}
	numberEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	lines := []string{}
	for idx := 1; idx <= 10; idx++ {
		key := fmt.Sprintf("option%d", idx)
		val := strings.TrimSpace(optionString(opts, key, ""))
		if val != "" {
			lines = append(lines, fmt.Sprintf("%s %s", numberEmojis[idx-1], val))
		}
	}
	if len(lines) < 2 {
		respondText(s, i, "At least two options are required.")
		return
	}
	user := interactionUser(i)
	author := "unknown"
	if user != nil {
		author = user.String()
	}
	embed := &discordgo.MessageEmbed{
		Title:       question,
		Description: strings.Join(lines, "\n"),
		Color:       0x5865F2,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Poll by " + author},
	}
	respondEmbed(s, i, embed)
}

func handleTimer(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	seconds := optionInt(opts, "seconds", 0)
	if seconds < 1 || seconds > 300 {
		respondText(s, i, "Seconds must be between 1 and 300.")
		return
	}
	user := interactionUser(i)
	mention := ""
	if user != nil {
		mention = "<@" + user.ID + ">"
	}
	end := time.Now().Add(time.Duration(seconds) * time.Second)
	respondText(s, i, fmt.Sprintf("Timer started. Ends <t:%d:R>.", end.Unix()))
	go func(channelID string, duration int64, tag string) {
		<-time.After(time.Duration(duration) * time.Second)
		msg := "Timer complete!"
		if tag != "" {
			msg = fmt.Sprintf("%s timer complete!", tag)
		}
		_, _ = s.ChannelMessageSend(channelID, msg)
	}(i.ChannelID, seconds, mention)
}

func handleRemind(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	timeText := strings.ToLower(strings.TrimSpace(optionString(opts, "time", "")))
	message := strings.TrimSpace(optionString(opts, "message", ""))
	if message == "" {
		respondText(s, i, "Reminder message is required.")
		return
	}
	duration, ok := parseReminderDuration(timeText)
	if !ok || duration <= 0 {
		respondText(s, i, "Invalid time format. Use like `30s`, `5m`, `2h`, `1d`.")
		return
	}
	if duration > 30*24*time.Hour {
		respondText(s, i, "Maximum reminder time is 30 days.")
		return
	}

	reminders := readReminders()
	count := 0
	for _, r := range reminders {
		if r.UserID == user.ID {
			count++
		}
	}
	if count >= 10 {
		respondText(s, i, "You already have 10 active reminders.")
		return
	}
	now := time.Now().UnixMilli()
	entry := reminderEntry{
		UserID:    user.ID,
		Message:   message,
		CreatedAt: now,
		DueTime:   now + duration.Milliseconds(),
		GuildID:   i.GuildID,
	}
	reminders = append(reminders, entry)
	writeReminders(reminders)
	respondText(s, i, fmt.Sprintf("Reminder set for <t:%d:R>.\nMessage: %s", entry.DueTime/1000, message))
}

func handleReminders(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "list"
	}
	reminders := readReminders()
	userReminders := make([]reminderEntry, 0)
	for _, r := range reminders {
		if r.UserID == user.ID {
			userReminders = append(userReminders, r)
		}
	}
	sort.Slice(userReminders, func(a, b int) bool { return userReminders[a].DueTime < userReminders[b].DueTime })

	switch sub {
	case "list":
		if len(userReminders) == 0 {
			respondText(s, i, "No active reminders. Use `/remind` to add one.")
			return
		}
		lines := make([]string, 0, len(userReminders))
		for idx, r := range userReminders {
			lines = append(lines, fmt.Sprintf("%d. %s (due <t:%d:R>)", idx+1, r.Message, r.DueTime/1000))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	case "clear":
		kept := make([]reminderEntry, 0, len(reminders))
		for _, r := range reminders {
			if r.UserID != user.ID {
				kept = append(kept, r)
			}
		}
		writeReminders(kept)
		respondText(s, i, fmt.Sprintf("Cleared %d reminders.", len(userReminders)))
	case "remove":
		num := int(optionInt(subOpts, "number", 0))
		if num < 1 || num > len(userReminders) {
			respondText(s, i, "Invalid reminder number.")
			return
		}
		target := userReminders[num-1]
		kept := make([]reminderEntry, 0, len(reminders)-1)
		removed := false
		for _, r := range reminders {
			if !removed && r.UserID == target.UserID && r.CreatedAt == target.CreatedAt && r.Message == target.Message {
				removed = true
				continue
			}
			kept = append(kept, r)
		}
		writeReminders(kept)
		respondText(s, i, "Reminder removed.")
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleAFK(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	reason := strings.TrimSpace(optionString(opts, "reason", "No reason provided"))
	afkMap := map[string]afkStatus{}
	_ = readData("afk-users.json", &afkMap)
	afkMap[user.ID] = afkStatus{Reason: reason, Timestamp: time.Now().UnixMilli()}
	_ = writeData("afk-users.json", afkMap)
	respondText(s, i, fmt.Sprintf("AFK set: %s", reason))
}

func handleClearMyData(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	if !optionBool(opts, "confirm", false) {
		respondText(s, i, "Set `confirm` to true to delete your data.")
		return
	}

	clearUserKeyInObject("trivia-scores.json", user.ID)
	clearUserKeyInObject("daily-rewards.json", user.ID)
	clearUserKeyInObject("loot.json", user.ID)
	clearUserKeyInObject("quests.json", user.ID)
	clearUserKeyInObject("battle-stats.json", user.ID)
	clearUserKeyInObject("afk-users.json", user.ID)
	clearUserKeyInObject("maze-leaderboard.json", user.ID)

	reminders := readReminders()
	kept := make([]reminderEntry, 0, len(reminders))
	for _, r := range reminders {
		if r.UserID != user.ID {
			kept = append(kept, r)
		}
	}
	writeReminders(kept)
	respondText(s, i, "Your stored bot data has been deleted.")
}
