package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func handleBattle(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for battle.")
		return
	}
	opponent := optionUser(opts, "opponent")
	if opponent == nil {
		respondText(s, i, "Opponent is required.")
		return
	}
	if opponent.Bot {
		respondText(s, i, "You cannot battle bots.")
		return
	}
	if opponent.ID == user.ID {
		respondText(s, i, "You cannot battle yourself.")
		return
	}
	type fighter struct {
		ID       string
		Username string
		HP       int
	}
	a := fighter{ID: user.ID, Username: user.Username, HP: 100}
	b := fighter{ID: opponent.ID, Username: opponent.Username, HP: 100}
	turnA := true
	logs := []string{}
	attacks := []struct {
		name string
		min  int
		max  int
		crit float64
	}{
		{"Quick Jab", 8, 15, 0.10},
		{"Power Strike", 15, 25, 0.15},
		{"Tactical Shot", 10, 20, 0.20},
		{"Chaos Blast", 5, 35, 0.25},
	}
	for a.HP > 0 && b.HP > 0 {
		attack := attacks[rand.Intn(len(attacks))]
		damage := rand.Intn(attack.max-attack.min+1) + attack.min
		if rand.Float64() < attack.crit {
			damage = int(float64(damage) * 1.5)
		}
		if turnA {
			b.HP = max(0, b.HP-damage)
			logs = append(logs, fmt.Sprintf("%s used %s for %d", a.Username, attack.name, damage))
		} else {
			a.HP = max(0, a.HP-damage)
			logs = append(logs, fmt.Sprintf("%s used %s for %d", b.Username, attack.name, damage))
		}
		turnA = !turnA
	}
	winner := a
	loser := b
	if b.HP > 0 {
		winner = b
		loser = a
	}

	stats := map[string]battleStats{}
	_ = readData("battle-stats.json", &stats)
	ws := stats[winner.ID]
	ls := stats[loser.ID]
	ws.Username = winner.Username
	ls.Username = loser.Username
	ws.Wins++
	ws.Streak++
	ls.Losses++
	ls.Streak = 0
	stats[winner.ID] = ws
	stats[loser.ID] = ls
	_ = writeData("battle-stats.json", stats)

	last := strings.Join(logs[max(0, len(logs)-3):], "\n")
	embed := &discordgo.MessageEmbed{
		Title:       "\u2694\uFE0F Battle Result",
		Description: fmt.Sprintf("**Winner:** %s\n**Final HP:** %d\n\n```%s```", winner.Username, winner.HP, last),
		Color:       0xED4245,
	}
	respondEmbed(s, i, embed)
}

func handleDaily(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for daily rewards.")
		return
	}
	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "claim"
	}
	_ = subOpts
	all := map[string]dailyUser{}
	_ = readData("daily-rewards.json", &all)
	u := all[user.ID]
	if u.Username == "" {
		u = dailyUser{Username: user.Username}
	}
	u.Username = user.Username
	now := time.Now().UnixMilli()
	day := int64(24 * time.Hour / time.Millisecond)

	switch sub {
	case "claim":
		if now-u.LastClaim < day {
			left := day - (now - u.LastClaim)
			hours := left / int64(time.Hour/time.Millisecond)
			mins := (left % int64(time.Hour/time.Millisecond)) / int64(time.Minute/time.Millisecond)
			respondText(s, i, fmt.Sprintf("Already claimed. Next claim in %dh %dm.", hours, mins))
			return
		}
		if now-u.LastClaim < 2*day {
			u.Streak++
		} else {
			u.Streak = 1
		}
		base := 100
		bonus := min(u.Streak*10, 500)
		reward := base + bonus
		if rand.Float64() < 0.1 {
			reward *= 2
		}

		// Apply active boosts
		boosts := getActiveBoosts(user.ID)
		multiplier := 1.0
		for _, b := range boosts {
			if b.BoostType == "daily_multiplier" || b.BoostType == "global_coin_multiplier" {
				multiplier *= b.Multiplier
			}
		}
		if multiplier > 1.0 {
			reward = int(float64(reward) * multiplier)
		}

		u.Coins += reward
		u.LastClaim = now
		u.TotalClaims++
		all[user.ID] = u
		_ = writeData("daily-rewards.json", all)
		respondText(s, i, fmt.Sprintf("Claimed %d coins. Total: %d | Streak: %d", reward, u.Coins, u.Streak))
	case "stats":
		next := u.LastClaim + day
		canClaim := now >= next
		nextText := "available now"
		if !canClaim {
			nextText = fmt.Sprintf("<t:%d:R>", next/1000)
		}
		respondText(s, i, fmt.Sprintf("Coins: %d\nStreak: %d\nClaims: %d\nBoss Kills: %d\nNext Claim: %s", u.Coins, u.Streak, u.TotalClaims, u.BossKills, nextText))
	case "leaderboard":
		type row struct {
			Name   string
			Coins  int
			Streak int
		}
		rows := []row{}
		for _, v := range all {
			rows = append(rows, row{Name: v.Username, Coins: v.Coins, Streak: v.Streak})
		}
		if len(rows) == 0 {
			respondText(s, i, "No rewards claimed yet.")
			return
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].Coins > rows[b].Coins })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		for idx, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. %s - %d coins (streak %d)", idx+1, r.Name, r.Coins, r.Streak))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleBossRaid(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for boss raid.")
		return
	}
	sub, _ := getSubcommand(opts)
	if sub == "" {
		sub = "status"
	}
	all := map[string]raidGuild{}
	_ = readData("bossraid.json", &all)
	g := all[i.GuildID]
	if g.Participants == nil {
		g.Participants = map[string]*raidParticipant{}
	}

	switch sub {
	case "status":
		if g.Boss == nil {
			respondText(s, i, "No active boss. Admin can spawn one with /bossraid spawn.")
			return
		}
		respondText(s, i, fmt.Sprintf("%s %s\nHP: %d/%d\nParticipants: %d\nTotal Damage: %d", g.Boss.Emoji, g.Boss.Name, g.Boss.CurrentHP, g.Boss.MaxHP, len(g.Participants), g.TotalDamage))
	case "spawn":
		if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator) == 0 {
			respondText(s, i, "Only administrators can spawn bosses.")
			return
		}
		if g.Boss != nil && g.Boss.CurrentHP > 0 {
			respondText(s, i, fmt.Sprintf("%s is still alive with %d HP.", g.Boss.Name, g.Boss.CurrentHP))
			return
		}
		template := raidBosses[rand.Intn(len(raidBosses))]
		template.CurrentHP = template.MaxHP
		template.SpawnedAt = time.Now().UnixMilli()
		g.Boss = &template
		g.Participants = map[string]*raidParticipant{}
		g.TotalDamage = 0
		all[i.GuildID] = g
		_ = writeData("bossraid.json", all)
		respondText(s, i, fmt.Sprintf("@everyone\n%s %s appeared with %d HP", g.Boss.Emoji, g.Boss.Name, g.Boss.MaxHP))
	case "attack":
		if g.Boss == nil || g.Boss.CurrentHP <= 0 {
			respondText(s, i, "No active boss to attack.")
			return
		}
		p := g.Participants[user.ID]
		if p == nil {
			p = &raidParticipant{Username: user.Username}
			g.Participants[user.ID] = p
		}
		now := time.Now().UnixMilli()
		if now-p.LastAttack < int64(5*time.Minute/time.Millisecond) {
			left := int((int64(5*time.Minute/time.Millisecond) - (now - p.LastAttack)) / 60000)
			if left < 1 {
				left = 1
			}
			respondText(s, i, fmt.Sprintf("Attack cooldown active. Try again in %d minute(s).", left))
			return
		}
		damage := rand.Intn(401) + 100
		crit := rand.Float64() < 0.15
		if crit {
			damage *= 2
		}

		// Apply active boosts
		boosts := getActiveBoosts(user.ID)
		multiplier := 1.0
		for _, b := range boosts {
			if b.BoostType == "raid_multiplier" || b.BoostType == "global_coin_multiplier" {
				multiplier *= b.Multiplier
			}
		}
		if multiplier > 1.0 {
			damage = int(float64(damage) * multiplier)
		}

		g.Boss.CurrentHP = max(0, g.Boss.CurrentHP-damage)
		p.Damage += damage
		p.Attacks++
		p.LastAttack = now
		p.Username = user.Username
		g.TotalDamage += damage
		defeated := g.Boss.CurrentHP == 0
		all[i.GuildID] = g
		_ = writeData("bossraid.json", all)

		if defeated {
			rewards := map[string]dailyUser{}
			_ = readData("daily-rewards.json", &rewards)
			for uid, part := range g.Participants {
				du := rewards[uid]
				if du.Username == "" {
					du.Username = part.Username
				}
				du.Coins += part.Damage / 10
				du.BossKills++
				du.Username = part.Username
				rewards[uid] = du
			}
			_ = writeData("daily-rewards.json", rewards)
			loot := g.Boss.Loot[rand.Intn(len(g.Boss.Loot))]
			respondText(s, i, fmt.Sprintf("%s defeated! Final blow by %s for %d damage%s\nLoot: %s", g.Boss.Name, user.Username, damage, map[bool]string{true: " (CRIT)", false: ""}[crit], loot))
			return
		}
		respondText(s, i, fmt.Sprintf("Dealt %d damage%s. %s HP: %d/%d", damage, map[bool]string{true: " (CRIT)", false: ""}[crit], g.Boss.Name, g.Boss.CurrentHP, g.Boss.MaxHP))
	case "leaderboard":
		if len(g.Participants) == 0 {
			respondText(s, i, "No attacks yet.")
			return
		}
		type row struct {
			Name    string
			Damage  int
			Attacks int
		}
		rows := []row{}
		for _, p := range g.Participants {
			rows = append(rows, row{Name: p.Username, Damage: p.Damage, Attacks: p.Attacks})
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].Damage > rows[b].Damage })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		for idx, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. %s - %d damage (%d attacks)", idx+1, r.Name, r.Damage, r.Attacks))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleQuote(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for quote command.")
		return
	}
	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "random"
	}
	all := map[string]quoteGuild{}
	_ = readData("quotes.json", &all)
	g := all[i.GuildID]
	switch sub {
	case "add":
		text := optionString(subOpts, "text", "")
		if text == "" {
			respondText(s, i, "Quote text is required.")
			return
		}
		author := user
		if au := optionUser(subOpts, "author"); au != nil {
			author = au
		}
		q := quoteItem{
			ID:   len(g.Quotes) + 1,
			Text: text,
			Author: quoteAuthor{
				ID:       author.ID,
				Username: author.Username,
				Avatar:   author.AvatarURL(""),
			},
			AddedBy:   quoteAdder{ID: user.ID, Username: user.Username},
			Timestamp: time.Now().UnixMilli(),
			GuildID:   i.GuildID,
		}
		g.Quotes = append(g.Quotes, q)
		all[i.GuildID] = g
		_ = writeData("quotes.json", all)
		respondText(s, i, fmt.Sprintf("\U0001F4DD Quote #%d added: \"%s\" - %s", q.ID, q.Text, q.Author.Username))
	case "random":
		if len(g.Quotes) == 0 {
			respondText(s, i, "No quotes yet. Use /quote add.")
			return
		}
		q := g.Quotes[rand.Intn(len(g.Quotes))]
		respondText(s, i, fmt.Sprintf("\"%s\"\n- %s (Quote #%d)", q.Text, q.Author.Username, q.ID))
	case "list":
		if len(g.Quotes) == 0 {
			respondText(s, i, "No quotes yet. Use /quote add.")
			return
		}
		lines := []string{}
		for idx := len(g.Quotes) - 1; idx >= 0 && len(lines) < 10; idx-- {
			q := g.Quotes[idx]
			preview := q.Text
			if len(preview) > 60 {
				preview = preview[:60] + "..."
			}
			lines = append(lines, fmt.Sprintf("#%d \"%s\" - %s", q.ID, preview, q.Author.Username))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	case "remove":
		if i.Member == nil || (i.Member.Permissions&discordgo.PermissionManageMessages) == 0 {
			respondText(s, i, "You need Manage Messages permission.")
			return
		}
		id := int(optionInt(subOpts, "id", -1))
		if id < 0 {
			respondText(s, i, "Quote id is required.")
			return
		}
		found := -1
		for idx, q := range g.Quotes {
			if q.ID == id {
				found = idx
				break
			}
		}
		if found == -1 {
			respondText(s, i, fmt.Sprintf("No quote found with ID %d.", id))
			return
		}
		removed := g.Quotes[found]
		g.Quotes = append(g.Quotes[:found], g.Quotes[found+1:]...)
		all[i.GuildID] = g
		_ = writeData("quotes.json", all)
		respondText(s, i, fmt.Sprintf("Removed quote #%d: \"%s\"", id, removed.Text))
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleQuest(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for quests.")
		return
	}
	sub, _ := getSubcommand(opts)
	if sub == "" {
		sub = "get"
	}
	all := map[string]questUser{}
	_ = readData("quests.json", &all)
	u := all[user.ID]
	if u.Username == "" {
		u = questUser{Username: user.Username}
	}
	u.Username = user.Username

	switch sub {
	case "get":
		if u.CurrentQuest != nil {
			respondText(s, i, fmt.Sprintf("Active quest: %s (%s, %d XP)", u.CurrentQuest.Task, u.CurrentQuest.Difficulty, u.CurrentQuest.XP))
			return
		}
		q := questTemplates[rand.Intn(len(questTemplates))]
		q.AssignedAt = time.Now().UnixMilli()
		u.CurrentQuest = &q
		all[user.ID] = u
		_ = writeData("quests.json", all)
		respondText(s, i, fmt.Sprintf("New quest: %s\nDifficulty: %s\nReward: %d XP", q.Task, q.Difficulty, q.XP))
	case "complete":
		if u.CurrentQuest == nil {
			respondText(s, i, "No active quest. Use /quest get.")
			return
		}
		q := u.CurrentQuest
		u.TotalXP += q.XP
		u.CompletedQuests++
		u.CurrentQuest = nil
		all[user.ID] = u
		_ = writeData("quests.json", all)
		level := (u.TotalXP / 100) + 1
		respondText(s, i, fmt.Sprintf("Quest completed: %s\n+%d XP\nLevel: %d", q.Task, q.XP, level))
	case "leaderboard":
		type row struct {
			Name      string
			TotalXP   int
			Completed int
		}
		rows := []row{}
		for _, v := range all {
			rows = append(rows, row{Name: v.Username, TotalXP: v.TotalXP, Completed: v.CompletedQuests})
		}
		if len(rows) == 0 {
			respondText(s, i, "No quest completions yet.")
			return
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].TotalXP > rows[b].TotalXP })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		for idx, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. %s - Lvl %d (%d XP, %d quests)", idx+1, r.Name, (r.TotalXP/100)+1, r.TotalXP, r.Completed))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	case "stats":
		level := (u.TotalXP / 100) + 1
		xpNext := level*100 - u.TotalXP
		msg := fmt.Sprintf("Level: %d\nTotal XP: %d\nXP to next level: %d\nCompleted quests: %d", level, u.TotalXP, xpNext, u.CompletedQuests)
		if u.CurrentQuest != nil {
			msg += fmt.Sprintf("\nCurrent quest: %s (%s)", u.CurrentQuest.Task, u.CurrentQuest.Difficulty)
		}
		respondText(s, i, msg)
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleLoot(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for loot.")
		return
	}
	sub, _ := getSubcommand(opts)
	if sub == "" {
		sub = "open"
	}
	all := map[string]lootUser{}
	_ = readData("loot.json", &all)
	u := all[user.ID]
	if u.Username == "" {
		u = lootUser{
			Username:  user.Username,
			Inventory: []lootItem{},
		}
	}
	u.Username = user.Username

	chooseRarity := func() string {
		r := rand.Float64() * 100
		switch {
		case r < 35:
			return "common"
		case r < 60:
			return "uncommon"
		case r < 80:
			return "rare"
		case r < 92:
			return "epic"
		case r < 98:
			return "legendary"
		case r < 99.5:
			return "cosmic"
		default:
			return "cursed"
		}
	}

	switch sub {
	case "open":
		rarity := chooseRarity()
		table := lootTables[rarity]
		item := table.Items[rand.Intn(len(table.Items))]
		u.Inventory = append(u.Inventory, lootItem{Item: item, Rarity: rarity, ObtainedAt: time.Now().UnixMilli()})
		switch rarity {
		case "common":
			u.Stats.Common++
		case "uncommon":
			u.Stats.Uncommon++
		case "rare":
			u.Stats.Rare++
		case "epic":
			u.Stats.Epic++
		case "legendary":
			u.Stats.Legendary++
		case "cosmic":
			u.Stats.Cosmic++
		case "cursed":
			u.Stats.Cursed++
		}
		u.TotalLoots++
		all[user.ID] = u
		_ = writeData("loot.json", all)
		respondText(s, i, fmt.Sprintf("Loot opened:\n%s %s\n%s\nTotal loot: %d", table.Emoji, table.Name, item, u.TotalLoots))
	case "inventory":
		if len(u.Inventory) == 0 {
			respondText(s, i, "Inventory is empty. Use /loot open.")
			return
		}
		lines := []string{}
		for idx := len(u.Inventory) - 1; idx >= 0 && len(lines) < 10; idx-- {
			it := u.Inventory[idx]
			table := lootTables[it.Rarity]
			lines = append(lines, fmt.Sprintf("%s %s (%s)", table.Emoji, it.Item, table.Name))
		}
		respondText(s, i, strings.Join(lines, "\n"))
	case "stats":
		msg := fmt.Sprintf("Total Loot: %d\nCommon: %d\nUncommon: %d\nRare: %d\nEpic: %d\nLegendary: %d\nCosmic: %d\nCursed: %d",
			u.TotalLoots, u.Stats.Common, u.Stats.Uncommon, u.Stats.Rare, u.Stats.Epic, u.Stats.Legendary, u.Stats.Cosmic, u.Stats.Cursed)
		respondText(s, i, msg)
	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

func handleTag(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for tag.")
		return
	}

	sub, subOpts := getSubcommand(opts)
	if sub == "" {
		sub = "challenge"
	}

	switch sub {
	case "challenge":
		opponent := optionUser(subOpts, "opponent")
		if opponent == nil {
			respondText(s, i, "You must specify an opponent.")
			return
		}
		if opponent.Bot {
			respondText(s, i, "You cannot challenge bots.")
			return
		}
		if opponent.ID == user.ID {
			respondText(s, i, "You cannot challenge yourself.")
			return
		}

		sessionID := fmt.Sprintf("%s_%s_%d", user.ID, opponent.ID, time.Now().UnixMilli())

		embed := &discordgo.MessageEmbed{
			Title:       "🏃 Tag Challenge!",
			Description: fmt.Sprintf("%s challenges %s to a game of tag!\n\nRules: 5x5 grid, 20 moves max. First to tag the opponent wins!", user.Username, opponent.Username),
			Color:       ColorPurple,
		}

		buttons := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Accept",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("tag_accept_%s", sessionID),
				},
				discordgo.Button{
					Label:    "Decline",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("tag_decline_%s", sessionID),
				},
			},
		}

		resp := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{buttons},
			},
		}

		if err := s.InteractionRespond(i.Interaction, resp); err != nil {
			return
		}

		msg, _ := s.InteractionResponse(i.Interaction)
		if msg != nil {
			sessionMu.Lock()
			tagSessions[msg.ID] = &tagSession{
				SessionID:   sessionID,
				Player1ID:   user.ID,
				Player1Name: user.Username,
				Player2ID:   opponent.ID,
				Player2Name: opponent.Username,
				MessageID:   msg.ID,
				ChannelID:   i.ChannelID,
			}
			sessionMu.Unlock()

			// Auto-cleanup after 5 minutes
			go func(msgID string) {
				time.Sleep(5 * time.Minute)
				sessionMu.Lock()
				delete(tagSessions, msgID)
				sessionMu.Unlock()
			}(msg.ID)
		}

	case "leaderboard":
		tagStatsMap := map[string]tagStats{}
		_ = readData("tag-stats.json", &tagStatsMap)

		if len(tagStatsMap) == 0 {
			respondText(s, i, "No tag games played yet!")
			return
		}

		type row struct {
			Name        string
			Wins        int
			Losses      int
			TotalGames  int
			CoinsEarned int
		}

		rows := []row{}
		for _, s := range tagStatsMap {
			rows = append(rows, row{
				Name:        s.Username,
				Wins:        s.Wins,
				Losses:      s.Losses,
				TotalGames:  s.TotalGames,
				CoinsEarned: s.CoinsEarned,
			})
		}

		sort.Slice(rows, func(a, b int) bool {
			return rows[a].Wins > rows[b].Wins
		})

		if len(rows) > 10 {
			rows = rows[:10]
		}

		lines := []string{}
		for idx, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. **%s** - %d wins, %d losses (%d coins earned)",
				idx+1, r.Name, r.Wins, r.Losses, r.CoinsEarned))
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🏆 Tag Leaderboard",
			Description: strings.Join(lines, "\n"),
			Color:       ColorGold,
		}
		respondEmbed(s, i, embed)

	case "stats":
		targetUser := user
		if u := optionUser(subOpts, "user"); u != nil {
			targetUser = u
		}

		tagStatsMap := map[string]tagStats{}
		_ = readData("tag-stats.json", &tagStatsMap)

		stats := tagStatsMap[targetUser.ID]
		if stats.Username == "" {
			respondText(s, i, fmt.Sprintf("%s hasn't played any tag games yet!", targetUser.Username))
			return
		}

		winRate := 0.0
		if stats.TotalGames > 0 {
			winRate = float64(stats.Wins) / float64(stats.TotalGames) * 100
		}

		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("🏃 %s's Tag Stats", stats.Username),
			Color: ColorBlue,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Wins", Value: fmt.Sprintf("%d", stats.Wins), Inline: true},
				{Name: "Losses", Value: fmt.Sprintf("%d", stats.Losses), Inline: true},
				{Name: "Win Rate", Value: fmt.Sprintf("%.1f%%", winRate), Inline: true},
				{Name: "Total Games", Value: fmt.Sprintf("%d", stats.TotalGames), Inline: true},
				{Name: "Coins Earned", Value: fmt.Sprintf("%d", stats.CoinsEarned), Inline: true},
			},
		}
		respondEmbed(s, i, embed)

	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

const ColorGold = 0xFFD700
