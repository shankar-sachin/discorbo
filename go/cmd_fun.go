package main

import (
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Simple fun commands (no API, no persistent state)
func handleSimpleFun(s *discordgo.Session, i *discordgo.InteractionCreate, cmd string, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	switch cmd {
	case "coinflip":
		result := "Heads"
		if rand.Intn(2) != 0 {
			result = "Tails"
		}
		embed := createFunEmbed("🪙 Coin Flip", fmt.Sprintf("**Result:** %s", result))
		respondEmbed(s, i, embed)
	case "dice":
		sides := int64(6)
		count := int64(1)
		for _, o := range opts {
			if o.Name == "sides" {
				sides = o.IntValue()
			}
			if o.Name == "count" {
				count = o.IntValue()
			}
		}
		if sides < 2 {
			sides = 6
		}
		if count < 1 {
			count = 1
		}
		if count > 20 {
			count = 20
		}
		rolls := make([]string, 0, count)
		total := 0
		for j := int64(0); j < count; j++ {
			r := rand.Intn(int(sides)) + 1
			total += r
			rolls = append(rolls, strconv.Itoa(r))
		}
		desc := fmt.Sprintf("**Rolls:** [%s]\n**Total:** %d", strings.Join(rolls, ", "), total)
		embed := createFunEmbed(fmt.Sprintf("🎲 Rolled %dd%d", count, sides), desc)
		respondEmbed(s, i, embed)
	case "8ball":
		answers := []string{"Yes", "No", "Maybe", "Definitely", "Unclear", "Ask later"}
		question := optionString(opts, "question", "")
		answer := answers[rand.Intn(len(answers))]
		embed := createFunEmbed("🔮 Magic 8-Ball", fmt.Sprintf("**Question:** %s\n**Answer:** %s", question, answer))
		respondEmbed(s, i, embed)
	case "random-number":
		min := int64(1)
		max := int64(100)
		for _, o := range opts {
			if o.Name == "min" {
				min = o.IntValue()
			}
			if o.Name == "max" {
				max = o.IntValue()
			}
		}
		if max < min {
			min, max = max, min
		}
		var result int64
		if max == min {
			result = min
		} else {
			result = rand.Int63n(max-min+1) + min
		}
		embed := createFunEmbed("🔢 Random Number", fmt.Sprintf("**Range:** %d - %d\n**Result:** %d", min, max, result))
		respondEmbed(s, i, embed)
	case "random-choice":
		raw := optionString(opts, "options", "")
		items := []string{}
		for _, p := range strings.Split(raw, ",") {
			t := strings.TrimSpace(p)
			if t != "" {
				items = append(items, t)
			}
		}
		if len(items) == 0 {
			embed := createErrorEmbed("Error", "No options provided.")
			respondEmbed(s, i, embed)
		} else {
			choice := items[rand.Intn(len(items))]
			embed := createFunEmbed("🎯 Random Choice", fmt.Sprintf("**Options:** %s\n**Picked:** %s", strings.Join(items, ", "), choice))
			respondEmbed(s, i, embed)
		}
	case "rate":
		thing := optionString(opts, "thing", "that")
		rating := rand.Intn(10) + 1
		embed := createFunEmbed("⭐ Rating", fmt.Sprintf("I rate **%s** %d/10", thing, rating))
		respondEmbed(s, i, embed)
	case "reverse-text":
		src := optionString(opts, "text", "")
		embed := createFunEmbed("🔄 Reversed Text", reverse(src))
		respondEmbed(s, i, embed)
	case "mock-text":
		src := optionString(opts, "text", "")
		embed := createFunEmbed("🙃 Mocking Text", mockCase(src))
		respondEmbed(s, i, embed)
	case "flip-text":
		src := optionString(opts, "text", "")
		embed := createFunEmbed("↕️ Flipped Text", reverse(src))
		respondEmbed(s, i, embed)
	case "rps":
		user := strings.ToLower(optionString(opts, "choice", "rock"))
		picks := []string{"rock", "paper", "scissors"}
		bot := picks[rand.Intn(3)]
		var result string
		if user == bot {
			result = "🤝 Tie!"
		} else if (user == "rock" && bot == "scissors") || (user == "paper" && bot == "rock") || (user == "scissors" && bot == "paper") {
			result = "🎉 You Win!"
		} else {
			result = "😔 I Win!"
		}
		embed := createFunEmbed("✊✋✌️ Rock Paper Scissors", fmt.Sprintf("**You:** %s\n**Me:** %s\n\n%s", user, bot, result))
		respondEmbed(s, i, embed)
	case "roll":
		expr := optionString(opts, "dice", "1d20")
		handleRoll(s, i, expr)
	}
}

// Interactive fun commands (user interaction, no API)
func handleInteractiveFun(s *discordgo.Session, i *discordgo.InteractionCreate, cmd string, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	switch cmd {
	case "ship":
		u1 := optionUser(opts, "user1")
		u2 := optionUser(opts, "user2")
		if u1 == nil || u2 == nil {
			embed := createErrorEmbed("Error", "Both users are required.")
			respondEmbed(s, i, embed)
			return
		}
		combined := u1.ID + u2.ID
		hash := 0
		for _, ch := range combined {
			hash = ((hash << 5) - hash) + int(ch)
		}
		comp := hash
		if comp < 0 {
			comp = -comp
		}
		comp = comp % 101
		hearts := "💔"
		if comp > 75 {
			hearts = "💖💖💖"
		} else if comp > 50 {
			hearts = "💕💕"
		} else if comp > 25 {
			hearts = "💕"
		}
		embed := createFunEmbed("💘 Love Calculator", fmt.Sprintf("**%s** + **%s**\n\n%s **%d%%** compatibility %s", u1.Username, u2.Username, hearts, comp, hearts))
		respondEmbed(s, i, embed)
	case "summon":
		u := optionUser(opts, "user")
		if u == nil {
			embed := createErrorEmbed("Error", "User is required.")
			respondEmbed(s, i, embed)
			return
		}
		reason := optionString(opts, "reason", "")
		msg := fmt.Sprintf(summonMessages[rand.Intn(len(summonMessages))], "<@"+u.ID+">")
		if reason != "" {
			msg += "\n\n**Reason:** " + reason
		}
		embed := createFunEmbed("✨ Summoning Ritual", msg)
		respondEmbed(s, i, embed)
	case "vibecheck":
		target := interactionUser(i)
		if target == nil {
			embed := createErrorEmbed("Error", "Unable to identify user for vibe check.")
			respondEmbed(s, i, embed)
			return
		}
		if u := optionUser(opts, "user"); u != nil {
			target = u
		}
		seed := int64(0)
		for _, ch := range target.ID + time.Now().UTC().Format("2006-01-02") {
			seed += int64(ch)
		}
		vibes := []string{"Immaculate", "Main Character", "Suspicious", "Chaotic Neutral", "Legendary", "Touch Grass"}
		idx := int(seed % int64(len(vibes)))
		pct := (idx + 1) * 100 / len(vibes)
		embed := createFunEmbed("🫨 Vibe Check", fmt.Sprintf("**%s's vibe:** %s (%d%%)", target.Username, vibes[idx], pct))
		respondEmbed(s, i, embed)
	case "would-you-rather":
		q := wyrQuestions[rand.Intn(len(wyrQuestions))]
		embed := createFunEmbed("🤔 Would You Rather", fmt.Sprintf("**Option 1:** %s\n\n**Option 2:** %s", q[0], q[1]))
		respondEmbed(s, i, embed)
	case "hotseat":
		members, err := s.GuildMembers(i.GuildID, "", 1000)
		if err != nil || len(members) == 0 {
			embed := createErrorEmbed("Error", "Failed to fetch server members.")
			respondEmbed(s, i, embed)
			return
		}
		candidates := make([]*discordgo.Member, 0, len(members))
		for _, m := range members {
			if m.User != nil && !m.User.Bot {
				candidates = append(candidates, m)
			}
		}
		if len(candidates) == 0 {
			embed := createErrorEmbed("Error", "No human members found.")
			respondEmbed(s, i, embed)
			return
		}
		m := candidates[rand.Intn(len(candidates))]
		q := hotseatQuestions[rand.Intn(len(hotseatQuestions))]
		embed := createFunEmbed("🔥 Hotseat", fmt.Sprintf("<@%s>\n\n**Question:** %s", m.User.ID, q))
		respondEmbed(s, i, embed)
	}
}

// API-based fun commands
func handleAPIFun(s *discordgo.Session, i *discordgo.InteractionCreate, cmd string, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	switch cmd {
	case "joke":
		deferReply(s, i)
		category := optionString(opts, "category", "Any")
		url := fmt.Sprintf("https://v2.jokeapi.dev/joke/%s?blacklistFlags=nsfw,religious,political,racist,sexist,explicit", category)
		body, err := httpGet(url)
		if err != nil {
			editDeferredText(s, i, "Failed to fetch joke.")
			return
		}
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			editDeferredText(s, i, "Failed to parse joke.")
			return
		}
		typ, _ := data["type"].(string)
		if typ == "single" {
			editDeferredText(s, i, fmt.Sprintf("%v", data["joke"]))
			return
		}
		editDeferredText(s, i, fmt.Sprintf("%v\n\n||%v||", data["setup"], data["delivery"]))
	case "meme":
		deferReply(s, i)
		subreddits := []string{"memes", "dankmemes", "wholesomememes", "me_irl", "AdviceAnimals"}
		sub := subreddits[rand.Intn(len(subreddits))]
		url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=100", sub)
		body, err := httpGet(url)
		if err != nil {
			editDeferredText(s, i, "Failed to fetch meme.")
			return
		}
		var res struct {
			Data struct {
				Children []struct {
					Data struct {
						Title       string `json:"title"`
						URL         string `json:"url"`
						Permalink   string `json:"permalink"`
						IsVideo     bool   `json:"is_video"`
						Stickied    bool   `json:"stickied"`
						Ups         int    `json:"ups"`
						NumComments int    `json:"num_comments"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &res); err != nil {
			editDeferredText(s, i, "Failed to parse meme response.")
			return
		}
		posts := make([]struct {
			Title       string
			URL         string
			Permalink   string
			Ups         int
			NumComments int
		}, 0)
		for _, c := range res.Data.Children {
			p := c.Data
			if p.Stickied || p.IsVideo {
				continue
			}
			u := strings.ToLower(p.URL)
			if strings.HasSuffix(u, ".jpg") || strings.HasSuffix(u, ".png") || strings.HasSuffix(u, ".gif") {
				posts = append(posts, struct {
					Title       string
					URL         string
					Permalink   string
					Ups         int
					NumComments int
				}{Title: p.Title, URL: p.URL, Permalink: p.Permalink, Ups: p.Ups, NumComments: p.NumComments})
			}
		}
		if len(posts) == 0 {
			editDeferredText(s, i, "No meme found right now.")
			return
		}
		p := posts[rand.Intn(len(posts))]
		embed := &discordgo.MessageEmbed{
			Title:       p.Title,
			Description: fmt.Sprintf("\U0001F44D %d upvotes | \U0001F4AC %d comments", p.Ups, p.NumComments),
			Color:       0xEB459E,
			URL:         "https://reddit.com" + p.Permalink,
			Image:       &discordgo.MessageEmbedImage{URL: p.URL},
			Footer:      &discordgo.MessageEmbedFooter{Text: "r/" + sub + " • Discorbo"},
		}
		editDeferredEmbed(s, i, embed, nil)
	case "trivia":
		deferReply(s, i)
		user := interactionUser(i)
		if user == nil {
			editDeferredText(s, i, "Unable to identify user for trivia.")
			return
		}
		category := optionString(opts, "category", "")
		difficulty := optionString(opts, "difficulty", "")
		url := "https://opentdb.com/api.php?amount=1"
		if category != "" {
			url += "&category=" + category
		}
		if difficulty != "" {
			url += "&difficulty=" + difficulty
		}
		body, err := httpGet(url)
		if err != nil {
			editDeferredText(s, i, "Trivia API is unavailable.")
			return
		}
		var res struct {
			ResponseCode int `json:"response_code"`
			Results      []struct {
				Category         string   `json:"category"`
				Type             string   `json:"type"`
				Difficulty       string   `json:"difficulty"`
				Question         string   `json:"question"`
				CorrectAnswer    string   `json:"correct_answer"`
				IncorrectAnswers []string `json:"incorrect_answers"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &res); err != nil || res.ResponseCode != 0 || len(res.Results) == 0 {
			editDeferredText(s, i, "No trivia question available.")
			return
		}
		q := res.Results[0]
		answers := append([]string{html.UnescapeString(q.CorrectAnswer)}, htmlUnescapeSlice(q.IncorrectAnswers)...)
		rand.Shuffle(len(answers), func(i1, i2 int) { answers[i1], answers[i2] = answers[i2], answers[i1] })
		correct := 0
		for idx, a := range answers {
			if a == html.UnescapeString(q.CorrectAnswer) {
				correct = idx
				break
			}
		}
		points := 15
		switch strings.ToLower(q.Difficulty) {
		case "easy":
			points = 10
		case "medium":
			points = 20
		case "hard":
			points = 30
		}
		token := fmt.Sprintf("%s_%d", i.ID, rand.Intn(100000))
		triviaMu.Lock()
		triviaSessions[token] = triviaSession{
			UserID:        user.ID,
			CorrectIndex:  correct,
			CorrectAnswer: html.UnescapeString(q.CorrectAnswer),
			Points:        points,
			ExpiresAt:     time.Now().Add(30 * time.Second).UnixMilli(),
		}
		triviaMu.Unlock()

		rows := []discordgo.MessageComponent{}
		curRow := discordgo.ActionsRow{}
		for idx, ans := range answers {
			label := ans
			if len(label) > 80 {
				label = label[:80]
			}
			btn := discordgo.Button{
				Label:    label,
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("trivia_%s_%d", token, idx),
			}
			curRow.Components = append(curRow.Components, btn)
			if len(curRow.Components) == 2 || idx == len(answers)-1 {
				rows = append(rows, curRow)
				curRow = discordgo.ActionsRow{}
			}
		}
		embed := &discordgo.MessageEmbed{
			Title:       "\U0001F9E0 Trivia - " + q.Category,
			Description: fmt.Sprintf("**%s**\n\n\U0001F3AF Difficulty: %s\n\U0001F4AF Points: %d", html.UnescapeString(q.Question), strings.ToUpper(q.Difficulty), points),
			Color:       0xEB459E,
		}
		editDeferredEmbed(s, i, embed, rows)
	case "trivia-leaderboard":
		scores := map[string]triviaScore{}
		_ = readData("trivia-scores.json", &scores)
		if len(scores) == 0 {
			respondText(s, i, "No scores yet. Use /trivia to start playing.")
			return
		}
		type row struct {
			UserID string
			triviaScore
		}
		rows := make([]row, 0, len(scores))
		for uid, sc := range scores {
			rows = append(rows, row{UserID: uid, triviaScore: sc})
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].Score > rows[b].Score })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := make([]string, 0, len(rows))
		for idx, r := range rows {
			acc := 0.0
			if r.Total > 0 {
				acc = float64(r.Correct) * 100 / float64(r.Total)
			}
			lines = append(lines, fmt.Sprintf("%d. %s - %d pts (%.1f%%)", idx+1, r.Username, r.Score, acc))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	}
}

// handleFunCmd is the top-level /fun group router.
func handleFunCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	subOpts := sub.Options

	switch sub.Name {
	// Simple / interactive / API subcommands
	case "8ball", "coinflip", "dice", "random-number", "random-choice", "rate",
		"reverse-text", "mock-text", "flip-text", "rps", "roll":
		handleSimpleFun(s, i, sub.Name, subOpts)
	case "ship", "summon", "vibecheck", "would-you-rather", "hotseat":
		handleInteractiveFun(s, i, sub.Name, subOpts)
	case "joke", "meme":
		handleAPIFun(s, i, sub.Name, subOpts)
	// trivia is now a SubcommandGroup (play/leaderboard)
	case "trivia":
		if len(subOpts) == 0 {
			return
		}
		trivSub := subOpts[0]
		switch trivSub.Name {
		case "play":
			handleAPIFun(s, i, "trivia", trivSub.Options)
		case "leaderboard":
			handleAPIFun(s, i, "trivia-leaderboard", nil)
		}
	// Game subcommands
	case "battle":
		handleBattle(s, i, subOpts)
	// SubcommandGroups — pass the inner subcommand slice
	case "bossraid":
		handleBossRaid(s, i, subOpts)
	case "daily":
		handleDaily(s, i, subOpts)
	case "loot":
		handleLoot(s, i, subOpts)
	case "quest":
		handleQuest(s, i, subOpts)
	case "quote":
		handleQuote(s, i, subOpts)
	}
}
