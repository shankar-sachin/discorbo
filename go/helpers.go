package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

// Math constants
const (
	mathPi  = 3.141592653589793
	mathE   = 2.718281828459045
	mathTau = 6.283185307179586
	mathPhi = 1.618033988749895
)

// Embed color constants
const (
	ColorPurple = 0xEB459E // Fun/primary color
	ColorBlue   = 0x5865F2 // Info/utility
	ColorGreen  = 0x57F287 // Success
	ColorYellow = 0xFEE75C // Warning
	ColorRed    = 0xED4245 // Error
	ColorGray   = 0x99AAB5 // Neutral
)

// Discord interaction helpers
func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	// Convert to info embed for consistency
	embed := createInfoEmbed("", text)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, e *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{e}},
	})
}

// Enhanced embed creators
func createSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✅ " + title,
		Description: description,
		Color:       ColorGreen,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
	}
}

func createInfoEmbed(title, description string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Description: description,
		Color:       ColorBlue,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
	}
	if title != "" {
		embed.Title = title
	}
	return embed
}

func createErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "❌ " + title,
		Description: description,
		Color:       ColorRed,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
	}
}

func createWarningEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⚠️ " + title,
		Description: description,
		Color:       ColorYellow,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
	}
}

func createFunEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       ColorPurple,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
	}
}

// userAvatar returns the avatar URL for a user (256px), or "" if none.
func userAvatar(u *discordgo.User) string {
	if u == nil {
		return ""
	}
	return u.AvatarURL("256")
}

// createGameEmbed builds a fun embed with an optional thumbnail.
func createGameEmbed(title, description, thumbnailURL string) *discordgo.MessageEmbed {
	e := createFunEmbed(title, description)
	if thumbnailURL != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: thumbnailURL}
	}
	return e
}

func deferReply(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func editDeferredText(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &text})
}

func editDeferredEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, e *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	embeds := []*discordgo.MessageEmbed{e}
	components := comps
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
}

func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i == nil {
		return nil
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	if i.User != nil {
		return i.User
	}
	if i.Interaction != nil {
		if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			return i.Interaction.Member.User
		}
		if i.Interaction.User != nil {
			return i.Interaction.User
		}
	}
	return nil
}

// Option extraction helpers
func optionString(opts []*discordgo.ApplicationCommandInteractionDataOption, name, def string) string {
	for _, o := range opts {
		if o.Name == name {
			return o.StringValue()
		}
	}
	return def
}

func optionInt(opts []*discordgo.ApplicationCommandInteractionDataOption, name string, def int64) int64 {
	for _, o := range opts {
		if o.Name == name {
			return o.IntValue()
		}
	}
	return def
}

func optionBool(opts []*discordgo.ApplicationCommandInteractionDataOption, name string, def bool) bool {
	for _, o := range opts {
		if o.Name == name {
			return o.BoolValue()
		}
	}
	return def
}

func optionUser(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.User {
	for _, o := range opts {
		if o.Name == name {
			return o.UserValue(nil)
		}
	}
	return nil
}

func optionChannel(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.Channel {
	for _, o := range opts {
		if o.Name == name {
			return o.ChannelValue(nil)
		}
	}
	return nil
}

func optionRole(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.Role {
	for _, o := range opts {
		if o.Name == name {
			return o.RoleValue(nil, "")
		}
	}
	return nil
}

func getSubcommand(opts []*discordgo.ApplicationCommandInteractionDataOption) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
	for _, o := range opts {
		if o.Type == discordgo.ApplicationCommandOptionSubCommand {
			return o.Name, o.Options
		}
	}
	return "", opts
}

// String manipulation helpers
func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func mockCase(s string) string {
	out := make([]rune, 0, len([]rune(s)))
	up := false
	for _, ch := range []rune(s) {
		if up {
			out = append(out, []rune(strings.ToUpper(string(ch)))...)
		} else {
			out = append(out, []rune(strings.ToLower(string(ch)))...)
		}
		if ch != ' ' {
			up = !up
		}
	}
	return string(out)
}

// HTTP helper
func httpGet(url string) ([]byte, error) {
	client := http.Client{Timeout: 7 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "DiscorboBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func htmlUnescapeSlice(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, strings.TrimSpace(v))
	}
	return out
}

// Dice rolling
func handleRoll(s *discordgo.Session, i *discordgo.InteractionCreate, expr string) {
	count, sides, mod, ok := parseDice(expr)
	if !ok {
		embed := createErrorEmbed("Invalid Dice Notation", "Use format like: d20, 3d6, 2d10+5")
		respondEmbed(s, i, embed)
		return
	}
	if count > 20 || sides < 2 || sides > 1000 {
		embed := createErrorEmbed("Invalid Dice Parameters", "Limits: up to 20 dice, 2-1000 sides per die")
		respondEmbed(s, i, embed)
		return
	}
	rolls := make([]int, 0, count)
	total := 0
	hasNat20 := false
	hasNat1 := false

	for j := 0; j < count; j++ {
		r := rand.Intn(sides) + 1
		rolls = append(rolls, r)
		total += r
		if sides == 20 && r == 20 {
			hasNat20 = true
		}
		if r == 1 {
			hasNat1 = true
		}
	}

	final := total + mod
	rollsStr := make([]string, len(rolls))
	for idx, r := range rolls {
		// Highlight nat 20s and nat 1s
		if sides == 20 && r == 20 {
			rollsStr[idx] = fmt.Sprintf("**%d**", r)
		} else if r == 1 {
			rollsStr[idx] = fmt.Sprintf("*%d*", r)
		} else {
			rollsStr[idx] = fmt.Sprintf("%d", r)
		}
	}

	average := float64(total) / float64(count)

	desc := fmt.Sprintf("**Rolls:** [%s]\n**Average:** %.1f", strings.Join(rollsStr, ", "), average)
	if mod != 0 {
		desc += fmt.Sprintf("\n**Modifier:** %+d", mod)
	}

	if hasNat20 {
		desc += "\n🎉 **NATURAL 20!** Critical success!"
	}
	if hasNat1 {
		desc += "\n💀 **NATURAL 1!** Critical failure!"
	}

	embed := createFunEmbed(fmt.Sprintf("🎲 Rolling %s", expr), desc)
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Total", Value: fmt.Sprintf("**%d**", final), Inline: true},
	}

	respondEmbed(s, i, embed)
}

func parseDice(expr string) (int, int, int, bool) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	parts := strings.Split(expr, "d")
	if len(parts) != 2 {
		return 0, 0, 0, false
	}
	count := 1
	if parts[0] != "" {
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, false
		}
		count = v
	}
	sidesPart := parts[1]
	mod := 0
	sides := 0
	if strings.Contains(sidesPart, "+") {
		sub := strings.SplitN(sidesPart, "+", 2)
		v, e1 := strconv.Atoi(sub[0])
		m, e2 := strconv.Atoi(sub[1])
		if e1 != nil || e2 != nil {
			return 0, 0, 0, false
		}
		sides = v
		mod = m
	} else if strings.Contains(sidesPart, "-") {
		sub := strings.SplitN(sidesPart, "-", 2)
		v, e1 := strconv.Atoi(sub[0])
		m, e2 := strconv.Atoi(sub[1])
		if e1 != nil || e2 != nil {
			return 0, 0, 0, false
		}
		sides = v
		mod = -m
	} else {
		v, err := strconv.Atoi(sidesPart)
		if err != nil {
			return 0, 0, 0, false
		}
		sides = v
	}
	return count, sides, mod, true
}

// Math expression evaluator
type mathParser struct {
	input string
	pos   int
}

func evalMath(expr string) (string, error) {
	// Step 1: Replace constants
	expr = replaceConstants(expr)

	// Step 2: Parse scientific notation
	expr = parseScientific(expr)

	// Step 3: Evaluate functions
	var err error
	expr, err = evaluateFunctions(expr)
	if err != nil {
		return "", err
	}

	// Step 4: Continue with existing recursive descent parser
	p := &mathParser{input: strings.TrimSpace(expr)}
	if p.input == "" {
		return "", errors.New("empty")
	}
	v, err := p.parseExpr()
	if err != nil {
		return "", err
	}
	p.skipSpaces()
	if p.pos != len(p.input) {
		return "", errors.New("unexpected trailing input")
	}
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10), nil
	}
	return strconv.FormatFloat(v, 'f', -1, 64), nil
}

func replaceConstants(expr string) string {
	expr = strings.ReplaceAll(expr, "pi", fmt.Sprintf("%f", mathPi))
	expr = strings.ReplaceAll(expr, "e", fmt.Sprintf("%f", mathE))
	expr = strings.ReplaceAll(expr, "tau", fmt.Sprintf("%f", mathTau))
	expr = strings.ReplaceAll(expr, "phi", fmt.Sprintf("%f", mathPhi))
	return expr
}

func parseScientific(expr string) string {
	// Replace scientific notation like 1e6 → 1000000
	pattern := regexp.MustCompile(`(\d+(?:\.\d+)?)e([+-]?\d+)`)
	return pattern.ReplaceAllStringFunc(expr, func(match string) string {
		val, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}
		return fmt.Sprintf("%f", val)
	})
}

func evaluateFunctions(expr string) (string, error) {
	// Regex: \w+\([^)]+\)
	funcPattern := regexp.MustCompile(`(\w+)\(([^)]+)\)`)

	for funcPattern.MatchString(expr) {
		matches := funcPattern.FindStringSubmatch(expr)
		if len(matches) < 3 {
			break
		}

		funcName := matches[1]
		argExpr := matches[2]

		// Recursively evaluate argument
		argResult, err := evalMath(argExpr)
		if err != nil {
			return "", err
		}

		argVal, err := strconv.ParseFloat(argResult, 64)
		if err != nil {
			return "", err
		}

		result, err := parseFunction(funcName, argVal)
		if err != nil {
			return "", err
		}

		// Replace function call with result
		expr = strings.Replace(expr, matches[0], fmt.Sprintf("%f", result), 1)
	}

	return expr, nil
}

func parseFunction(name string, arg float64) (float64, error) {
	switch strings.ToLower(name) {
	case "sqrt":
		if arg < 0 {
			return 0, errors.New("sqrt of negative number")
		}
		return math.Sqrt(arg), nil
	case "sin":
		return math.Sin(arg), nil // radians
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "asin":
		return math.Asin(arg), nil
	case "acos":
		return math.Acos(arg), nil
	case "atan":
		return math.Atan(arg), nil
	case "log":
		if arg <= 0 {
			return 0, errors.New("log of non-positive number")
		}
		return math.Log10(arg), nil
	case "ln":
		if arg <= 0 {
			return 0, errors.New("ln of non-positive number")
		}
		return math.Log(arg), nil
	case "abs":
		return math.Abs(arg), nil
	case "ceil":
		return math.Ceil(arg), nil
	case "floor":
		return math.Floor(arg), nil
	case "round":
		return math.Round(arg), nil
	case "exp":
		return math.Exp(arg), nil
	default:
		return 0, errors.New("unknown function: " + name)
	}
}

func (p *mathParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('+') {
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left += right
			continue
		}
		if p.match('-') {
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left -= right
			continue
		}
		break
	}
	return left, nil
}

func (p *mathParser) parseTerm() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('*') {
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			left *= right
			continue
		}
		if p.match('/') {
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, errors.New("division by zero")
			}
			left /= right
			continue
		}
		if p.match('%') {
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, errors.New("mod by zero")
			}
			left = float64(int64(left) % int64(right))
			continue
		}
		break
	}
	return left, nil
}

func (p *mathParser) parsePower() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if !p.match('^') {
			break
		}
		right, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		val := 1.0
		exp := int(right)
		for idx := 0; idx < exp; idx++ {
			val *= left
		}
		left = val
	}
	return left, nil
}

func (p *mathParser) parseUnary() (float64, error) {
	p.skipSpaces()
	if p.match('+') {
		return p.parseUnary()
	}
	if p.match('-') {
		v, err := p.parseUnary()
		return -v, err
	}
	return p.parsePrimary()
}

func (p *mathParser) parsePrimary() (float64, error) {
	p.skipSpaces()
	if p.match('(') {
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if !p.match(')') {
			return 0, errors.New("missing )")
		}
		return v, nil
	}
	start := p.pos
	dotSeen := false
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if ch == '.' {
			if dotSeen {
				break
			}
			dotSeen = true
			p.pos++
			continue
		}
		if !unicode.IsDigit(ch) {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return 0, errors.New("expected number")
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

func (p *mathParser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *mathParser) match(ch byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}

// Snowflake timestamp extraction
func snowflakeUnix(id string) int64 {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return time.Now().Unix()
	}
	const discordEpoch = int64(1420070400000)
	ms := (v >> 22) + discordEpoch
	return ms / 1000
}

// Duration parsing for reminders
func parseReminderDuration(input string) (time.Duration, bool) {
	if len(input) < 2 {
		return 0, false
	}
	unit := input[len(input)-1]
	numText := input[:len(input)-1]
	value, err := strconv.Atoi(numText)
	if err != nil || value <= 0 {
		return 0, false
	}
	switch unit {
	case 's':
		return time.Duration(value) * time.Second, true
	case 'm':
		return time.Duration(value) * time.Minute, true
	case 'h':
		return time.Duration(value) * time.Hour, true
	case 'd':
		return time.Duration(value) * 24 * time.Hour, true
	default:
		return 0, false
	}
}

// Game constants
var summonMessages = []string{
	"The council demands your presence, %s.",
	"Through ancient rites, we call upon %s.",
	"By order of the realm, %s is summoned.",
	"From the shadows we call upon %s.",
	"The stars align for %s.",
}

var hotseatQuestions = []string{
	"What's your most embarrassing Discord moment?",
	"If you had to delete one app forever, what is it?",
	"What's something you're supposed to be doing right now?",
	"What's your most unpopular opinion?",
	"What's your current phone battery percentage?",
}

var wyrQuestions = [][2]string{
	{"Have the ability to fly", "Have the ability to be invisible"},
	{"Live without music", "Live without movies"},
	{"Be able to teleport", "Be able to time travel"},
	{"Fight one horse-sized duck", "Fight 100 duck-sized horses"},
}

var questTemplates = []questEntry{
	{Task: "Retrieve the sacred pizza from the Forbidden Fridge", Difficulty: "easy", XP: 50},
	{Task: "Defeat the Lag Monster in the Router Realm", Difficulty: "medium", XP: 100},
	{Task: "Survive a family gathering without checking your phone", Difficulty: "hard", XP: 200},
	{Task: "Complete a group project where everyone contributes", Difficulty: "legendary", XP: 500},
}

var raidBosses = []raidBoss{
	{Name: "The Lag Monster", Emoji: "\U0001F47E", MaxHP: 5000, Description: "A creature that feeds on your WiFi", Loot: []string{"Router of Legends", "Ping Reducer", "5G Crystal"}},
	{Name: "The Monday Overlord", Emoji: "\U0001F4C5", MaxHP: 7000, Description: "The most feared entity", Loot: []string{"Weekend Extension", "Coffee of Awakening", "Skip Monday Pass"}},
	{Name: "Social Anxiety Dragon", Emoji: "\U0001F409", MaxHP: 5500, Description: "Guards social skills treasure", Loot: []string{"Conversation Starter Kit", "Confidence Amulet", "Small Talk Guide"}},
}


var lootTables = map[string]lootTable{
	"common":    {Name: "Common", Emoji: "\u26AA", Items: []string{"Broken Pencil", "Old Receipt", "Bottle Cap"}},
	"uncommon":  {Name: "Uncommon", Emoji: "\U0001F7E2", Items: []string{"Working Charger", "Good Vibes", "Fresh Pizza Slice"}},
	"rare":      {Name: "Rare", Emoji: "\U0001F535", Items: []string{"Productive Day", "Extra Fries in Bag", "Reply from Crush"}},
	"epic":      {Name: "Epic", Emoji: "\U0001F7E3", Items: []string{"WiFi That Actually Works", "Inbox Zero Achievement", "Main Character Moment"}},
	"legendary": {Name: "Legendary", Emoji: "\U0001F7E1", Items: []string{"Extra Day Weekend", "Unlimited Garlic Bread", "Infinite Battery Life"}},
	"cosmic":    {Name: "Cosmic", Emoji: "\U0001F30C", Items: []string{"Pause Time", "World Peace Token", "Respec Button for Life Choices"}},
	"cursed":    {Name: "Cursed", Emoji: "\U0001F480", Items: []string{"Wet Socks (Permanent)", "Eternal Loading Screen", "Unstoppable Hiccups"}},
}

// Tag game helper functions
func generateTagBoard() [][]string {
	board := make([][]string, 5)
	for i := range board {
		board[i] = make([]string, 5)
		for j := range board[i] {
			board[i][j] = "⬜"
		}
	}

	// Place random obstacles (8 total, avoid start positions)
	obstacles := 0
	for obstacles < 8 {
		x, y := rand.Intn(5), rand.Intn(5)
		if (x == 0 && y == 0) || (x == 4 && y == 4) {
			continue // Don't block spawns
		}
		if board[y][x] == "⬜" {
			board[y][x] = "⬛"
			obstacles++
		}
	}

	return board
}

func processTagMove(sess *tagSession, direction string) bool {
	// Get current player's position
	var x, y *int
	if sess.CurrentTurn == 0 {
		x, y = &sess.Player1X, &sess.Player1Y
	} else {
		x, y = &sess.Player2X, &sess.Player2Y
	}

	newX, newY := *x, *y

	switch direction {
	case "up":
		newY--
	case "down":
		newY++
	case "left":
		newX--
	case "right":
		newX++
	case "stay":
		// No movement
		return true
	default:
		return false
	}

	// Validate bounds
	if newX < 0 || newX >= 5 || newY < 0 || newY >= 5 {
		return false
	}

	// Check for wall
	if sess.Board[newY][newX] == "⬛" {
		return false
	}

	// Apply move
	*x = newX
	*y = newY

	return true
}

func buildTagEmbed(sess *tagSession) *discordgo.MessageEmbed {
	// Build board visualization
	boardCopy := make([][]string, 5)
	for i := range sess.Board {
		boardCopy[i] = make([]string, 5)
		copy(boardCopy[i], sess.Board[i])
	}

	// Place players
	boardCopy[sess.Player1Y][sess.Player1X] = "🔵"
	boardCopy[sess.Player2Y][sess.Player2X] = "🔴"

	boardLines := []string{}
	for _, row := range boardCopy {
		boardLines = append(boardLines, strings.Join(row, ""))
	}

	currentPlayer := sess.Player1Name
	if sess.CurrentTurn == 1 {
		currentPlayer = sess.Player2Name
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏃 Tag Game",
		Description: strings.Join(boardLines, "\n"),
		Color:       ColorPurple,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Current Turn", Value: currentPlayer, Inline: true},
			{Name: "Moves", Value: fmt.Sprintf("%d/%d", sess.Moves, sess.MaxMoves), Inline: true},
			{Name: "Players", Value: fmt.Sprintf("🔵 %s\n🔴 %s", sess.Player1Name, sess.Player2Name), Inline: false},
		},
	}

	return embed
}

func buildTagButtons(sessionID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "⬆️", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("tag_up_%s", sessionID)},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "⬅️", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("tag_left_%s", sessionID)},
				discordgo.Button{Label: "Stay", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("tag_stay_%s", sessionID)},
				discordgo.Button{Label: "➡️", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("tag_right_%s", sessionID)},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "⬇️", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("tag_down_%s", sessionID)},
			},
		},
	}
}
