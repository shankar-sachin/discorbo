package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Snipe cache for deleted messages
type snipedMessage struct {
	Content      string
	AuthorName   string
	AuthorAvatar string
	Time         time.Time
}

var snipeCache = map[string]*snipedMessage{}
var snipeMu sync.Mutex

// handleUtilCmd is the top-level /util group router.
func handleUtilCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	subOpts := sub.Options

	switch sub.Name {
	case "ping":
		respondText(s, i, fmt.Sprintf("Pong! Gateway ping: %dms", s.HeartbeatLatency().Milliseconds()))
	case "help":
		handleHelp(s, i, subOpts)
	case "avatar":
		handleAvatar(s, i, subOpts)
	case "userinfo":
		handleUserInfo(s, i, subOpts)
	case "serverinfo":
		handleServerInfo(s, i)
	case "channel-info":
		handleChannelInfo(s, i, subOpts)
	case "role-info":
		handleRoleInfo(s, i, subOpts)
	case "stats":
		handleStats(s, i)
	case "poll":
		handlePoll(s, i, subOpts)
	case "translate":
		handleTranslate(s, i, subOpts)
	case "timer":
		handleTimer(s, i, subOpts)
	case "remind":
		handleRemind(s, i, subOpts)
	case "reminders":
		handleReminders(s, i, subOpts)
	case "afk":
		handleAFK(s, i, subOpts)
	case "clear-my-data":
		handleClearMyData(s, i, subOpts)
	case "color":
		handleColor(s, i, subOpts)
	case "define":
		handleDefine(s, i, subOpts)
	case "github":
		handleGitHub(s, i, subOpts)
	case "snipe":
		handleSnipe(s, i)
	case "embed-builder":
		handleEmbedBuilder(s, i, subOpts)
	case "giveaway":
		handleGiveaway(s, i, subOpts)
	case "weather":
		handleWeather(s, i, subOpts)
	case "sticky":
		handleSticky(s, i, subOpts)
	}
}

// handleMathCmd is the top-level /math group router.
func handleMathCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	subOpts := sub.Options

	switch sub.Name {
	case "calc":
		handleCalc(s, i, subOpts)
	case "convert":
		handleConvert(s, i, subOpts)
	}
}

func handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	category := strings.ToLower(optionString(opts, "category", "all"))

	// Build subcommand lists per top-level group
	groups := map[string][]string{}
	for _, cmd := range allCommands() {
		lines := []string{}
		for _, sub := range cmd.Options {
			lines = append(lines, fmt.Sprintf("`/%s %s` - %s", cmd.Name, sub.Name, sub.Description))
		}
		sort.Strings(lines)
		groups[cmd.Name] = lines
	}

	truncate := func(lines []string) string {
		v := strings.Join(lines, "\n")
		if len(v) > 1024 {
			v = v[:1021] + "..."
		}
		return v
	}

	type groupDef struct {
		key   string
		icon  string
		label string
	}
	allGroups := []groupDef{
		{"fun", "🎉", "Fun (Creative/Social/Party)"},
		{"games", "🎮", "Games"},
		{"casino", "🎰", "Casino"},
		{"economy", "💰", "Economy"},
		{"mod", "🛡️", "Moderation"},
		{"util", "🛠️", "Utility"},
		{"math", "🔢", "Math"},
		{"music", "🎵", "Music"},
		{"level", "⭐", "Leveling"},
		{"welcome", "👋", "Welcome"},
	}

	total := 0
	for _, g := range allGroups {
		total += len(groups[g.key])
	}

	embed := richEmbed(richEmbedOpts{
		Title:       "🤖 Discorbo Command Center",
		Description: fmt.Sprintf("**%d commands** across **%d categories**\nUse `/category subcommand` to run a command.", total, len(allGroups)),
		Color:       ColorBlue,
		Category:    "📖 Help",
	})

	for _, g := range allGroups {
		if category != "all" && category != g.key {
			continue
		}
		lines := groups[g.key]
		value := "*No commands found.*"
		if len(lines) > 0 {
			if category == "all" {
				// In overview mode, show count only for brevity
				value = fmt.Sprintf("**%d** commands — `/util help category:%s`", len(lines), g.key)
			} else {
				value = truncate(lines)
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("%s %s (%d)", g.icon, g.label, len(lines)), Value: value,
		})
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
		respondText(s, i, fmt.Sprintf("Invalid expression: %v\n\nSupported:\n• Basic: +, -, *, /, ^, %%\n• Functions: sqrt, sin, cos, tan, log, ln, abs, etc.\n• Constants: pi, e, tau, phi\n• Scientific: 1e6, 2.5e-3", err))
		return
	}
	embed := createSuccessEmbed("Calculator", fmt.Sprintf("**Expression:** `%s`\n**Result:** `%s`", expr, result))
	respondEmbed(s, i, embed)
}

func handleTranslate(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	deferReply(s, i)
	text := optionString(opts, "text", "")
	toLang := strings.TrimSpace(optionString(opts, "to", ""))
	fromLang := strings.TrimSpace(optionString(opts, "from", "auto"))
	if text == "" || toLang == "" {
		editDeferredEmbed(s, i, createErrorEmbed("Missing Input", "Text and target language are required."), nil)
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
		editDeferredEmbed(s, i, createErrorEmbed("Translation Unavailable", "The translation service is currently unavailable. Try again later."), nil)
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
		editDeferredEmbed(s, i, createErrorEmbed("Translation Failed", "Could not translate the text. The translation service may be unavailable."), nil)
		return
	}
	detected := out.DetectedLanguage.Language
	if detected == "" {
		detected = fromLang
	}
	embed := createInfoEmbed("🌐 Translation", "")
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: fmt.Sprintf("Original (%s)", strings.ToUpper(detected)), Value: text, Inline: false},
		{Name: fmt.Sprintf("Translated (%s)", strings.ToUpper(toLang)), Value: out.TranslatedText, Inline: false},
	}
	editDeferredEmbed(s, i, embed, nil)
}

func handlePoll(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	question := optionString(opts, "question", "")
	if question == "" {
		respondEmbed(s, i, createErrorEmbed("Poll Error", "Question is required."))
		return
	}
	numberEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	options := []string{}
	for idx := 1; idx <= 10; idx++ {
		key := fmt.Sprintf("option%d", idx)
		val := strings.TrimSpace(optionString(opts, key, ""))
		if val != "" {
			options = append(options, val)
		}
	}
	if len(options) < 2 {
		respondEmbed(s, i, createErrorEmbed("Poll Error", "At least two options are required."))
		return
	}
	user := interactionUser(i)
	creator := "unknown"
	if user != nil {
		creator = user.Username
	}

	// Build embed showing options with 0 votes
	buildPollEmbed := func(question string, options []string, votes []int, totalVotes int) *discordgo.MessageEmbed {
		fields := make([]*discordgo.MessageEmbedField, 0, len(options))
		for idx, opt := range options {
			var barStr string
			v := 0
			if idx < len(votes) {
				v = votes[idx]
			}
			pct := 0.0
			if totalVotes > 0 {
				pct = float64(v) / float64(totalVotes) * 100
			}
			filled := int(pct / 10)
			barStr = strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s %s", numberEmojis[idx], opt),
				Value:  fmt.Sprintf("`%s` %d votes (%.0f%%)", barStr, v, pct),
				Inline: false,
			})
		}
		embed := &discordgo.MessageEmbed{
			Title:     "📊 " + question,
			Color:     ColorBlue,
			Fields:    fields,
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Poll by %s • %d total votes | Discorbo", creator, totalVotes)},
		}
		return embed
	}

	votes := make([]int, len(options))
	embed := buildPollEmbed(question, options, votes, 0)

	// Add vote buttons for up to 4 options
	var components []discordgo.MessageComponent
	if len(options) <= 4 {
		pollID := fmt.Sprintf("%s_%d", user.ID, time.Now().UnixMilli())
		btns := []discordgo.MessageComponent{}
		for idx, opt := range options {
			label := opt
			if len(label) > 20 {
				label = label[:17] + "..."
			}
			btns = append(btns, discordgo.Button{
				Label:    fmt.Sprintf("%s %s", numberEmojis[idx], label),
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("poll_vote_%s_%d", pollID, idx),
			})
		}
		components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: btns}}

		// Store session - will be keyed by message ID after followup
		// Use deferred response to get message ID
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			return
		}

		pollMu.Lock()
		pollSessions[msg.ID] = &pollSession{
			Question:  question,
			Options:   options,
			Votes:     votes,
			Voters:    map[string]int{},
			CreatorID: user.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour).UnixMilli(),
		}
		pollMu.Unlock()

		// Store the buildPollEmbed function data for updates - pass creator name
		// Clean up after 24 hours
		go func(msgID string) {
			time.Sleep(24 * time.Hour)
			pollMu.Lock()
			delete(pollSessions, msgID)
			pollMu.Unlock()
		}(msg.ID)
		return
	}

	// More than 4 options: just show embed without buttons
	respondEmbed(s, i, embed)
}

// pollVoteBarStr builds a vote bar for use in poll embed updates (shared helper)
func buildPollEmbedFromSession(sess *pollSession, creator string) *discordgo.MessageEmbed {
	numberEmojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	totalVotes := 0
	for _, v := range sess.Votes {
		totalVotes += v
	}
	fields := make([]*discordgo.MessageEmbedField, 0, len(sess.Options))
	for idx, opt := range sess.Options {
		v := sess.Votes[idx]
		pct := 0.0
		if totalVotes > 0 {
			pct = float64(v) / float64(totalVotes) * 100
		}
		filled := int(pct / 10)
		barStr := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", numberEmojis[idx], opt),
			Value:  fmt.Sprintf("`%s` %d votes (%.0f%%)", barStr, v, pct),
			Inline: false,
		})
	}
	return &discordgo.MessageEmbed{
		Title:     "📊 " + sess.Question,
		Color:     ColorBlue,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Poll by %s • %d total votes | Discorbo", creator, totalVotes)},
	}
}

func handleTimer(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	seconds := optionInt(opts, "seconds", 0)
	if seconds < 1 || seconds > 300 {
		respondEmbed(s, i, createErrorEmbed("Invalid Duration", "Timer must be between **1** and **300** seconds."))
		return
	}
	user := interactionUser(i)
	mention := ""
	if user != nil {
		mention = "<@" + user.ID + ">"
	}
	end := time.Now().Add(time.Duration(seconds) * time.Second)
	embed := createInfoEmbed("⏱️ Timer Set", fmt.Sprintf("Your timer will go off <t:%d:R>.", end.Unix()))
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Duration", Value: fmt.Sprintf("%d seconds", seconds), Inline: true},
		{Name: "Ends At", Value: fmt.Sprintf("<t:%d:T>", end.Unix()), Inline: true},
	}
	respondEmbed(s, i, embed)
	go func(channelID string, duration int64, tag string) {
		<-time.After(time.Duration(duration) * time.Second)
		completeEmbed := createSuccessEmbed("⏱️ Timer Complete!", "Your timer has finished!")
		if tag != "" {
			completeEmbed.Description = fmt.Sprintf("%s your timer has finished!", tag)
		}
		_, _ = s.ChannelMessageSendEmbed(channelID, completeEmbed)
	}(i.ChannelID, seconds, mention)
}

func handleRemind(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
		return
	}
	timeText := strings.ToLower(strings.TrimSpace(optionString(opts, "time", "")))
	message := strings.TrimSpace(optionString(opts, "message", ""))
	if message == "" {
		respondEmbed(s, i, createErrorEmbed("Missing Input", "Reminder message is required."))
		return
	}
	duration, ok := parseReminderDuration(timeText)
	if !ok || duration <= 0 {
		respondEmbed(s, i, createErrorEmbed("Invalid Time", "Use formats like `30s`, `5m`, `2h`, `1d`."))
		return
	}
	if duration > 30*24*time.Hour {
		respondEmbed(s, i, createErrorEmbed("Too Far", "Maximum reminder time is **30 days**."))
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
		respondEmbed(s, i, createErrorEmbed("Too Many Reminders", "You already have **10 active reminders**. Clear some with `/reminders clear`."))
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
	embed := createSuccessEmbed("⏰ Reminder Set!", "")
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📝 Message", Value: message, Inline: false},
		{Name: "⏰ Fires", Value: fmt.Sprintf("<t:%d:R> (<t:%d:T>)", entry.DueTime/1000, entry.DueTime/1000), Inline: false},
	}
	respondEmbed(s, i, embed)
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
			respondEmbed(s, i, createInfoEmbed("⏰ Your Reminders", "No active reminders. Use `/remind` to add one!"))
			return
		}
		fields := make([]*discordgo.MessageEmbedField, 0, len(userReminders))
		for idx, r := range userReminders {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("#%d", idx+1),
				Value:  fmt.Sprintf("%s\n⏰ <t:%d:R>", r.Message, r.DueTime/1000),
				Inline: false,
			})
		}
		embed := createInfoEmbed("⏰ Your Reminders", "")
		embed.Fields = fields
		embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("%d active reminders | Discorbo", len(userReminders))}
		respondEmbed(s, i, embed)
	case "clear":
		kept := make([]reminderEntry, 0, len(reminders))
		for _, r := range reminders {
			if r.UserID != user.ID {
				kept = append(kept, r)
			}
		}
		writeReminders(kept)
		respondEmbed(s, i, createSuccessEmbed("Reminders Cleared", fmt.Sprintf("Cleared **%d** reminder(s).", len(userReminders))))
	case "remove":
		num := int(optionInt(subOpts, "number", 0))
		if num < 1 || num > len(userReminders) {
			respondEmbed(s, i, createErrorEmbed("Invalid Number", fmt.Sprintf("Please enter a number between 1 and %d.", len(userReminders))))
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
		respondEmbed(s, i, createSuccessEmbed("Reminder Removed", fmt.Sprintf("Removed reminder #%d: *%s*", num, target.Message)))
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: list, clear, remove"))
	}
}

func handleAFK(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
		return
	}
	reason := strings.TrimSpace(optionString(opts, "reason", "No reason provided"))
	afkMap := map[string]afkStatus{}
	_ = readData("afk-users.json", &afkMap)
	afkMap[user.ID] = afkStatus{Reason: reason, Timestamp: time.Now().UnixMilli()}
	_ = writeData("afk-users.json", afkMap)
	embed := createInfoEmbed("💤 AFK Set", reason)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Since", Value: fmt.Sprintf("<t:%d:T>", time.Now().Unix()), Inline: true},
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Send a message to return from AFK | Discorbo"}
	respondEmbed(s, i, embed)
}

func handleConvert(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	value := optionFloat(opts, "value", 0)
	fromUnit := strings.ToLower(strings.TrimSpace(optionString(opts, "from", "")))
	toUnit := strings.ToLower(strings.TrimSpace(optionString(opts, "to", "")))

	if fromUnit == "" || toUnit == "" {
		respondText(s, i, "Both from and to units are required.")
		return
	}

	result, err := convertUnits(value, fromUnit, toUnit)
	if err != nil {
		respondText(s, i, fmt.Sprintf("Conversion error: %v", err))
		return
	}

	embed := createSuccessEmbed("Unit Conversion", result)
	respondEmbed(s, i, embed)
}

func convertUnits(value float64, from, to string) (string, error) {
	// Temperature conversions
	if (from == "c" || from == "f" || from == "k") && (to == "c" || to == "f" || to == "k") {
		return convertTemperature(value, from, to), nil
	}

	// Define conversion multipliers
	conversions := map[string]float64{
		"ft_to_m":  0.3048,
		"m_to_ft":  3.28084,
		"mi_to_km": 1.60934,
		"km_to_mi": 0.621371,
		"in_to_cm": 2.54,
		"cm_to_in": 0.393701,
		"lb_to_kg": 0.453592,
		"kg_to_lb": 2.20462,
		"oz_to_g":  28.3495,
		"g_to_oz":  0.035274,
		"gal_to_l": 3.78541,
		"l_to_gal": 0.264172,
	}

	key := from + "_to_" + to
	if multiplier, ok := conversions[key]; ok {
		result := value * multiplier
		return fmt.Sprintf("%.2f %s = %.2f %s", value, from, result, to), nil
	}

	return "", fmt.Errorf("unsupported conversion: %s to %s", from, to)
}

func convertTemperature(value float64, from, to string) string {
	var celsius float64

	// Convert to Celsius first
	switch from {
	case "f":
		celsius = (value - 32) * 5 / 9
	case "k":
		celsius = value - 273.15
	case "c":
		celsius = value
	default:
		return "Invalid temperature unit"
	}

	// Convert from Celsius to target
	var result float64
	switch to {
	case "f":
		result = celsius*9/5 + 32
	case "k":
		result = celsius + 273.15
	case "c":
		result = celsius
	default:
		return "Invalid temperature unit"
	}

	return fmt.Sprintf("%.2f°%s = %.2f°%s", value, strings.ToUpper(from), result, strings.ToUpper(to))
}

func handleClearMyData(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}
	if !optionBool(opts, "confirm", false) {
		respondEmbed(s, i, createWarningEmbed("Confirm Required", "Set the `confirm` option to `true` to permanently delete all your bot data.\n\n⚠️ **This action cannot be undone!**"))
		return
	}

	clearUserKeyInObject("trivia-scores.json", user.ID)
	clearUserKeyInObject("loot.json", user.ID)
	clearUserKeyInObject("quests.json", user.ID)
	clearUserKeyInObject("battle-stats.json", user.ID)
	clearUserKeyInObject("afk-users.json", user.ID)
	clearUserKeyInObject("maze-leaderboard.json", user.ID)
	clearUserKeyInObject("economy-users.json", user.ID)
	clearUserKeyInObject("tag-stats.json", user.ID)

	// Clear from transactions array
	transactions := []transactionLog{}
	_ = readData("transactions.json", &transactions)
	kept := []transactionLog{}
	for _, t := range transactions {
		if t.UserID != user.ID {
			kept = append(kept, t)
		}
	}
	_ = writeData("transactions.json", kept)

	reminders := readReminders()
	keptReminders := make([]reminderEntry, 0, len(reminders))
	for _, r := range reminders {
		if r.UserID != user.ID {
			keptReminders = append(keptReminders, r)
		}
	}
	writeReminders(keptReminders)
	embed := createSuccessEmbed("Data Deleted", "All your stored bot data has been permanently deleted.")
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Deleted", Value: "• Trivia scores\n• Daily rewards\n• Loot history\n• Quest data\n• Battle stats\n• AFK status\n• Maze records\n• Economy data\n• Reminders\n• Tag stats", Inline: false},
	}
	respondEmbed(s, i, embed)
}

// ============================================================================
// NICK COMMAND
// ============================================================================

func handleNick(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageNicknames) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Nicknames` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	target := optionUser(opts, "user")
	if target == nil {
		respondText(s, i, "You must specify a user.")
		return
	}

	nickname := optionString(opts, "nickname", "")

	err := s.GuildMemberNickname(i.GuildID, target.ID, nickname)
	if err != nil {
		embed := createErrorEmbed("Failed", fmt.Sprintf("Could not change nickname: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	msg := fmt.Sprintf("Reset **%s**'s nickname.", target.Username)
	if nickname != "" {
		msg = fmt.Sprintf("Changed **%s**'s nickname to **%s**.", target.Username, nickname)
	}
	embed := createSuccessEmbed("Nickname Updated", msg)
	respondEmbed(s, i, embed)
}

// ============================================================================
// ROLE COMMAND
// ============================================================================

func handleRole(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	sub, subOpts := getSubcommand(opts)

	switch sub {
	case "add", "remove":
		if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageRoles) {
			embed := createErrorEmbed("Missing Permissions", "You need the `Manage Roles` permission to use this command.")
			respondEmbed(s, i, embed)
			return
		}

		target := optionUser(subOpts, "user")
		if target == nil {
			respondText(s, i, "You must specify a user.")
			return
		}

		role := optionRole(subOpts, "role")
		if role == nil {
			respondText(s, i, "You must specify a role.")
			return
		}

		var err error
		var action string
		if sub == "add" {
			err = s.GuildMemberRoleAdd(i.GuildID, target.ID, role.ID)
			action = "added to"
		} else {
			err = s.GuildMemberRoleRemove(i.GuildID, target.ID, role.ID)
			action = "removed from"
		}

		if err != nil {
			embed := createErrorEmbed("Failed", fmt.Sprintf("Could not modify role: %v", err))
			respondEmbed(s, i, embed)
			return
		}

		embed := createSuccessEmbed("Role Updated", fmt.Sprintf("Role **%s** has been %s **%s**.", role.Name, action, target.Username))
		respondEmbed(s, i, embed)

	case "list":
		guild, err := s.Guild(i.GuildID)
		if err != nil {
			respondText(s, i, "Failed to fetch guild information.")
			return
		}

		if len(guild.Roles) == 0 {
			respondText(s, i, "This server has no roles.")
			return
		}

		// Sort roles by position (highest first)
		roles := guild.Roles
		sort.Slice(roles, func(a, b int) bool {
			return roles[a].Position > roles[b].Position
		})

		var roleList string
		count := 0
		for _, r := range roles {
			if r.Name == "@everyone" {
				continue
			}
			roleList += fmt.Sprintf("<@&%s> - `%s`\n", r.ID, r.ID)
			count++
			if count >= 20 {
				roleList += fmt.Sprintf("*...and %d more*", len(roles)-count-1)
				break
			}
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("📋 Server Roles (%d)", len(roles)-1),
			Description: roleList,
			Color:       ColorBlue,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		respondEmbed(s, i, embed)

	default:
		respondText(s, i, "Invalid subcommand. Use add, remove, or list.")
	}
}

// ============================================================================
// ANNOUNCE COMMAND
// ============================================================================

func handleAnnounce(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageMessages) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Messages` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	channel := optionChannel(opts, "channel")
	if channel == nil {
		respondText(s, i, "You must specify a channel.")
		return
	}

	message := optionString(opts, "message", "")
	if message == "" {
		respondText(s, i, "You must provide a message.")
		return
	}

	ping := optionBool(opts, "ping", false)

	// Build announcement embed
	guild, _ := s.Guild(i.GuildID)
	guildName := "Announcement"
	if guild != nil {
		guildName = guild.Name
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📢 %s", guildName),
		Description: message,
		Color:       ColorBlue,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Announced by %s", user.Username)},
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	content := ""
	if ping {
		content = "@everyone"
	}

	_, err := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	if err != nil {
		errEmbed := createErrorEmbed("Announcement Failed", fmt.Sprintf("Could not send announcement: %v", err))
		respondEmbed(s, i, errEmbed)
		return
	}

	confirmEmbed := createSuccessEmbed("Announcement Sent", fmt.Sprintf("Your announcement has been posted in <#%s>.", channel.ID))
	respondEmbed(s, i, confirmEmbed)
}

// ============================================================================
// SERVER UTILITY HANDLER
// ============================================================================

func handleServerUtility(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	switch i.ApplicationCommandData().Name {
	case "nick":
		handleNick(s, i, opts)
	case "role":
		handleRole(s, i, opts)
	case "announce":
		handleAnnounce(s, i, opts)
	}
}

// ============================================================================
// COLOR PREVIEW
// ============================================================================

func handleColor(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	hex := strings.TrimPrefix(optionString(opts, "hex", ""), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		respondEmbed(s, i, createErrorEmbed("Invalid Color", "Provide a valid hex color, e.g. `#FF5733` or `FF5733`."))
		return
	}
	val, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		respondEmbed(s, i, createErrorEmbed("Invalid Color", "Could not parse hex value."))
		return
	}
	r := (val >> 16) & 0xFF
	g := (val >> 8) & 0xFF
	b := val & 0xFF

	commonColors := map[string]string{
		"FF0000": "Red", "00FF00": "Green", "0000FF": "Blue",
		"FFFFFF": "White", "000000": "Black", "FFFF00": "Yellow",
		"FF00FF": "Magenta", "00FFFF": "Cyan", "FFA500": "Orange",
		"800080": "Purple", "FFC0CB": "Pink", "808080": "Gray",
	}
	name := commonColors[strings.ToUpper(hex)]
	desc := fmt.Sprintf("**Hex:** #%s\n**RGB:** %d, %d, %d", strings.ToUpper(hex), r, g, b)
	if name != "" {
		desc += fmt.Sprintf("\n**Name:** %s", name)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎨 Color Preview",
		Description: desc,
		Color:       int(val),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	respondEmbed(s, i, embed)
}

// ============================================================================
// DICTIONARY LOOKUP
// ============================================================================

func handleDefine(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	word := optionString(opts, "word", "")
	if word == "" {
		respondEmbed(s, i, createErrorEmbed("Missing Word", "Please provide a word to look up."))
		return
	}
	deferReply(s, i)

	data, err := httpGet("https://api.dictionaryapi.dev/api/v2/entries/en/" + word)
	if err != nil {
		editDeferredEmbed(s, i, createErrorEmbed("API Error", "Could not reach the dictionary API."), nil)
		return
	}

	var entries []struct {
		Word      string `json:"word"`
		Phonetic  string `json:"phonetic"`
		Meanings  []struct {
			PartOfSpeech string `json:"partOfSpeech"`
			Definitions  []struct {
				Definition string `json:"definition"`
			} `json:"definitions"`
		} `json:"meanings"`
	}
	if err := json.Unmarshal(data, &entries); err != nil || len(entries) == 0 {
		editDeferredEmbed(s, i, createErrorEmbed("Not Found", fmt.Sprintf("No definition found for **%s**.", word)), nil)
		return
	}

	entry := entries[0]
	desc := ""
	if entry.Phonetic != "" {
		desc += fmt.Sprintf("*%s*\n\n", entry.Phonetic)
	}
	for _, m := range entry.Meanings {
		desc += fmt.Sprintf("**%s**\n", m.PartOfSpeech)
		for j, d := range m.Definitions {
			if j >= 3 {
				break
			}
			desc += fmt.Sprintf("• %s\n", d.Definition)
		}
		desc += "\n"
	}
	if len(desc) > 4000 {
		desc = desc[:4000] + "…"
	}

	embed := createInfoEmbed("📖 "+entry.Word, desc)
	editDeferredEmbed(s, i, embed, nil)
}

// ============================================================================
// GITHUB REPO INFO
// ============================================================================

func handleGitHub(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	repo := optionString(opts, "repo", "")
	if repo == "" || !strings.Contains(repo, "/") {
		respondEmbed(s, i, createErrorEmbed("Invalid Repo", "Provide a repo in `owner/repo` format."))
		return
	}
	deferReply(s, i)

	data, err := httpGet("https://api.github.com/repos/" + repo)
	if err != nil {
		editDeferredEmbed(s, i, createErrorEmbed("API Error", "Could not reach the GitHub API."), nil)
		return
	}

	var gh struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Stars       int    `json:"stargazers_count"`
		Forks       int    `json:"forks_count"`
		Language    string `json:"language"`
		HTMLURL     string `json:"html_url"`
		OpenIssues  int    `json:"open_issues_count"`
		Owner       struct {
			AvatarURL string `json:"avatar_url"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(data, &gh); err != nil || gh.FullName == "" {
		editDeferredEmbed(s, i, createErrorEmbed("Not Found", "Repository not found or API error."), nil)
		return
	}

	lang := gh.Language
	if lang == "" {
		lang = "N/A"
	}
	desc := gh.Description
	if desc == "" {
		desc = "*No description*"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📦 %s", gh.FullName),
		URL:         gh.HTMLURL,
		Description: desc,
		Color:       ColorBlue,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: gh.Owner.AvatarURL},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Stars", Value: fmt.Sprintf("%d", gh.Stars), Inline: true},
			{Name: "🍴 Forks", Value: fmt.Sprintf("%d", gh.Forks), Inline: true},
			{Name: "💻 Language", Value: lang, Inline: true},
			{Name: "❗ Issues", Value: fmt.Sprintf("%d", gh.OpenIssues), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	editDeferredEmbed(s, i, embed, nil)
}

// ============================================================================
// SNIPE - RECOVER LAST DELETED MESSAGE
// ============================================================================

// HandleMessageDelete caches deleted messages for the snipe command.
func HandleMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	if m.BeforeDelete == nil {
		return
	}
	msg := m.BeforeDelete
	if msg.Author == nil || msg.Author.Bot {
		return
	}
	snipeMu.Lock()
	snipeCache[m.ChannelID] = &snipedMessage{
		Content:      msg.Content,
		AuthorName:   msg.Author.Username,
		AuthorAvatar: msg.Author.AvatarURL("128"),
		Time:         msg.Timestamp,
	}
	snipeMu.Unlock()
}

func handleSnipe(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	snipeMu.Lock()
	msg, ok := snipeCache[channelID]
	snipeMu.Unlock()

	if !ok || msg == nil {
		respondEmbed(s, i, createErrorEmbed("Nothing to Snipe", "No recently deleted messages in this channel."))
		return
	}

	content := msg.Content
	if content == "" {
		content = "*[embed or attachment]*"
	}
	if len(content) > 1024 {
		content = content[:1021] + "…"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔫 Sniped Message",
		Description: content,
		Color:       ColorBlue,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    msg.AuthorName,
			IconURL: msg.AuthorAvatar,
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Deleted %s", msg.Time.Format("Jan 2 15:04:05"))},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	respondEmbed(s, i, embed)
}

// ============================================================================
// EMBED BUILDER
// ============================================================================

func handleEmbedBuilder(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	title := optionString(opts, "title", "Embed")
	description := optionString(opts, "description", "")
	colorHex := strings.TrimPrefix(optionString(opts, "color", "5865F2"), "#")
	footer := optionString(opts, "footer", "")

	colorVal, err := strconv.ParseInt(colorHex, 16, 64)
	if err != nil {
		colorVal = ColorBlue
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       int(colorVal),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	if footer != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: footer}
	}
	respondEmbed(s, i, embed)
}

// ============================================================================
// GIVEAWAY
// ============================================================================

func handleGiveaway(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	prize := optionString(opts, "prize", "")
	duration := optionInt(opts, "duration", 5)
	winners := optionInt(opts, "winners", 1)

	if duration < 1 || duration > 1440 {
		respondEmbed(s, i, createErrorEmbed("Invalid Duration", "Duration must be between 1 and 1440 minutes."))
		return
	}
	if winners < 1 || winners > 20 {
		respondEmbed(s, i, createErrorEmbed("Invalid Winners", "Winners must be between 1 and 20."))
		return
	}

	endTime := time.Now().Add(time.Duration(duration) * time.Minute)
	user := interactionUser(i)
	hostName := "Unknown"
	if user != nil {
		hostName = user.Username
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎉 GIVEAWAY 🎉",
		Description: fmt.Sprintf("**Prize:** %s\n**Winners:** %d\n**Hosted by:** %s\n**Ends:** <t:%d:R>\n\nReact with 🎉 to enter!", prize, winners, hostName, endTime.Unix()),
		Color:       ColorPurple,
		Timestamp:   endTime.Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Ends at"},
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})

	// Get the response message and add the reaction
	resp, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		return
	}
	_ = s.MessageReactionAdd(i.ChannelID, resp.ID, "🎉")

	// Background goroutine to pick winners after duration
	go func(channelID, messageID string, winnerCount int64, endAt time.Time) {
		time.Sleep(time.Until(endAt))
		users, err := s.MessageReactions(channelID, messageID, "🎉", 100, "", "")
		if err != nil {
			return
		}
		// Filter out bots
		var candidates []string
		for _, u := range users {
			if !u.Bot {
				candidates = append(candidates, u.ID)
			}
		}
		if len(candidates) == 0 {
			endEmbed := createErrorEmbed("Giveaway Ended", "No valid participants. Nobody won **"+prize+"**.")
			_, _ = s.ChannelMessageSendEmbed(channelID, endEmbed)
			return
		}
		// Shuffle and pick winners
		rand.Shuffle(len(candidates), func(a, b int) { candidates[a], candidates[b] = candidates[b], candidates[a] })
		if int(winnerCount) > len(candidates) {
			winnerCount = int64(len(candidates))
		}
		picked := candidates[:winnerCount]
		mentions := make([]string, len(picked))
		for idx, uid := range picked {
			mentions[idx] = fmt.Sprintf("<@%s>", uid)
		}
		winEmbed := createSuccessEmbed("🎉 Giveaway Ended", fmt.Sprintf("**Prize:** %s\n**Winner(s):** %s", prize, strings.Join(mentions, ", ")))
		_, _ = s.ChannelMessageSendEmbed(channelID, winEmbed)
	}(i.ChannelID, resp.ID, winners, endTime)
}

// ============================================================================
// WEATHER
// ============================================================================

func handleWeather(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	city := optionString(opts, "city", "")
	if city == "" {
		respondEmbed(s, i, createErrorEmbed("Missing City", "Please provide a city name."))
		return
	}
	deferReply(s, i)

	url := "https://wttr.in/" + strings.ReplaceAll(city, " ", "+") + "?format=j1"
	data, err := httpGet(url)
	if err != nil {
		editDeferredEmbed(s, i, createErrorEmbed("API Error", "Could not reach the weather API."), nil)
		return
	}

	var weather struct {
		CurrentCondition []struct {
			TempC       string `json:"temp_C"`
			TempF       string `json:"temp_F"`
			Humidity    string `json:"humidity"`
			WindspeedKm string `json:"windspeedKmph"`
			WeatherDesc []struct {
				Value string `json:"value"`
			} `json:"weatherDesc"`
			FeelsLikeC string `json:"FeelsLikeC"`
		} `json:"current_condition"`
	}
	if err := json.Unmarshal(data, &weather); err != nil || len(weather.CurrentCondition) == 0 {
		editDeferredEmbed(s, i, createErrorEmbed("Not Found", "Could not find weather data for that location."), nil)
		return
	}

	cc := weather.CurrentCondition[0]
	condition := "Unknown"
	if len(cc.WeatherDesc) > 0 {
		condition = cc.WeatherDesc[0].Value
	}

	desc := fmt.Sprintf("**Condition:** %s\n**Temperature:** %s°C / %s°F\n**Feels Like:** %s°C\n**Humidity:** %s%%\n**Wind:** %s km/h",
		condition, cc.TempC, cc.TempF, cc.FeelsLikeC, cc.Humidity, cc.WindspeedKm)

	embed := createInfoEmbed("🌤️ Weather in "+city, desc)
	editDeferredEmbed(s, i, embed, nil)
}

// ============================================================================
// STICKY MESSAGE
// ============================================================================

func handleSticky(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	message := optionString(opts, "message", "")
	if message == "" {
		respondEmbed(s, i, createErrorEmbed("Missing Message", "Please provide a message for the sticky."))
		return
	}

	user := interactionUser(i)
	authorName := "Unknown"
	if user != nil {
		authorName = user.Username
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📌 Sticky Message",
		Description: message,
		Color:       ColorYellow,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Pinned by " + authorName},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	respondEmbed(s, i, embed)
}
