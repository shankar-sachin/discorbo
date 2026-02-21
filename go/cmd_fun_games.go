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
		respondEmbed(s, i, createErrorEmbed("Battle Error", "Unable to identify user."))
		return
	}
	opponent := optionUser(opts, "opponent")
	if opponent == nil {
		respondEmbed(s, i, createErrorEmbed("Battle Error", "You must specify an opponent."))
		return
	}
	if opponent.Bot {
		respondEmbed(s, i, createErrorEmbed("Battle Error", "You cannot battle bots."))
		return
	}
	if opponent.ID == user.ID {
		respondEmbed(s, i, createErrorEmbed("Battle Error", "You cannot battle yourself."))
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
	embed := createFunEmbed("⚔️ Battle Result", fmt.Sprintf("```%s```", last))
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "🏆 Winner", Value: winner.Username, Inline: true},
		{Name: "❤️ Final HP", Value: fmt.Sprintf("%d", winner.HP), Inline: true},
		{Name: "📊 Streak", Value: fmt.Sprintf("%d wins", stats[winner.ID].Streak), Inline: true},
	}
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(interactionUser(i))}
	respondEmbed(s, i, embed)
}

func handleDaily(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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
			embed := createInfoEmbed("⏰ Already Claimed", fmt.Sprintf("You've already claimed your daily reward.\nCome back in **%dh %dm**!", hours, mins))
			respondEmbed(s, i, embed)
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
		doubled := false
		if rand.Float64() < 0.1 {
			reward *= 2
			doubled = true
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

		nextClaim := fmt.Sprintf("<t:%d:R>", (now+day)/1000)
		desc := ""
		if doubled {
			desc = "🎉 **Lucky! You got double coins!**\n"
		}
		if multiplier > 1.0 {
			desc += fmt.Sprintf("🔥 Boost active (%.1fx)\n", multiplier)
		}
		embed := createSuccessEmbed("Daily Reward Claimed!", desc)
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "💰 Coins Earned", Value: fmt.Sprintf("**%d**", reward), Inline: true},
			{Name: "🔥 Streak", Value: fmt.Sprintf("**%d** days", u.Streak), Inline: true},
			{Name: "💳 Total Coins", Value: fmt.Sprintf("**%d**", u.Coins), Inline: true},
			{Name: "📅 Next Claim", Value: nextClaim, Inline: false},
		}
		respondEmbed(s, i, embed)
	case "stats":
		next := u.LastClaim + day
		canClaim := now >= next
		nextText := "✅ Available now!"
		if !canClaim {
			nextText = fmt.Sprintf("<t:%d:R>", next/1000)
		}
		level := (u.TotalClaims / 5) + 1
		embed := createInfoEmbed(fmt.Sprintf("📊 %s's Daily Stats", user.Username), "")
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "💰 Total Coins", Value: fmt.Sprintf("%d", u.Coins), Inline: true},
			{Name: "🔥 Streak", Value: fmt.Sprintf("%d days", u.Streak), Inline: true},
			{Name: "📅 Claims", Value: fmt.Sprintf("%d", u.TotalClaims), Inline: true},
			{Name: "👾 Boss Kills", Value: fmt.Sprintf("%d", u.BossKills), Inline: true},
			{Name: "⭐ Level", Value: fmt.Sprintf("%d", level), Inline: true},
			{Name: "⏰ Next Claim", Value: nextText, Inline: true},
		}
		respondEmbed(s, i, embed)
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
			respondEmbed(s, i, createInfoEmbed("💰 Daily Leaderboard", "No rewards claimed yet."))
			return
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].Coins > rows[b].Coins })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		medals := []string{"🥇", "🥈", "🥉"}
		for idx, r := range rows {
			prefix := fmt.Sprintf("%d.", idx+1)
			if idx < len(medals) {
				prefix = medals[idx]
			}
			lines = append(lines, fmt.Sprintf("%s **%s** — %d coins (🔥 %d)", prefix, r.Name, r.Coins, r.Streak))
		}
		embed := &discordgo.MessageEmbed{
			Title:       "💰 Daily Leaderboard",
			Description: strings.Join(lines, "\n"),
			Color:       ColorGold,
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
		}
		respondEmbed(s, i, embed)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: claim, stats, leaderboard"))
	}
}

func handleBossRaid(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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

	// HP bar helper
	hpBar := func(current, max int) string {
		if max == 0 {
			return "░░░░░░░░░░"
		}
		filled := int(float64(current) / float64(max) * 10)
		return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
	}

	switch sub {
	case "status":
		if g.Boss == nil {
			respondEmbed(s, i, createInfoEmbed("⚔️ Boss Raid", "No active boss. An admin can spawn one with `/bossraid spawn`."))
			return
		}
		pct := float64(g.Boss.CurrentHP) / float64(g.Boss.MaxHP) * 100
		embed := createFunEmbed(fmt.Sprintf("%s %s", g.Boss.Emoji, g.Boss.Name), g.Boss.Description)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "❤️ HP", Value: fmt.Sprintf("%s `%d/%d` (%.0f%%)", hpBar(g.Boss.CurrentHP, g.Boss.MaxHP), g.Boss.CurrentHP, g.Boss.MaxHP, pct), Inline: false},
			{Name: "⚔️ Total Damage", Value: fmt.Sprintf("%d", g.TotalDamage), Inline: true},
			{Name: "👥 Participants", Value: fmt.Sprintf("%d", len(g.Participants)), Inline: true},
		}
		respondEmbed(s, i, embed)
	case "spawn":
		if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator) == 0 {
			respondEmbed(s, i, createErrorEmbed("Missing Permissions", "Only administrators can spawn bosses."))
			return
		}
		if g.Boss != nil && g.Boss.CurrentHP > 0 {
			respondEmbed(s, i, createWarningEmbed("Boss Already Active", fmt.Sprintf("%s **%s** is still alive with **%d HP**!", g.Boss.Emoji, g.Boss.Name, g.Boss.CurrentHP)))
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
		embed := createFunEmbed(fmt.Sprintf("💀 %s %s Has Appeared!", g.Boss.Emoji, g.Boss.Name), g.Boss.Description)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "❤️ HP", Value: fmt.Sprintf("%d", g.Boss.MaxHP), Inline: true},
			{Name: "🎁 Possible Loot", Value: strings.Join(g.Boss.Loot, ", "), Inline: false},
		}
		respondEmbed(s, i, embed)
	case "attack":
		if g.Boss == nil || g.Boss.CurrentHP <= 0 {
			respondEmbed(s, i, createErrorEmbed("No Active Boss", "There's no boss to attack right now. Wait for an admin to spawn one."))
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
			respondEmbed(s, i, createInfoEmbed("⏰ Attack Cooldown", fmt.Sprintf("You need to wait **%d more minute(s)** before attacking again.", left)))
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
			critText := ""
			if crit {
				critText = " 💥 **CRITICAL HIT!**"
			}
			embed := createSuccessEmbed(fmt.Sprintf("%s %s Defeated!", g.Boss.Emoji, g.Boss.Name),
				fmt.Sprintf("**%s** landed the final blow for **%d damage**%s!\n🎁 Loot dropped: **%s**\nAll participants earned coins based on damage dealt!", user.Username, damage, critText, loot))
			respondEmbed(s, i, embed)
			return
		}
		critText := ""
		if crit {
			critText = " 💥 CRIT!"
		}
		pct := float64(g.Boss.CurrentHP) / float64(g.Boss.MaxHP) * 100
		embed := createSuccessEmbed("⚔️ Attack!", fmt.Sprintf("You dealt **%d damage**%s to **%s**!", damage, critText, g.Boss.Name))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Boss HP", Value: fmt.Sprintf("%s `%d/%d` (%.0f%%)", hpBar(g.Boss.CurrentHP, g.Boss.MaxHP), g.Boss.CurrentHP, g.Boss.MaxHP, pct), Inline: false},
			{Name: "Your Total Damage", Value: fmt.Sprintf("%d", p.Damage), Inline: true},
			{Name: "Your Attacks", Value: fmt.Sprintf("%d", p.Attacks), Inline: true},
		}
		respondEmbed(s, i, embed)
	case "leaderboard":
		if len(g.Participants) == 0 {
			respondEmbed(s, i, createInfoEmbed("⚔️ Raid Leaderboard", "No attacks yet. Use `/bossraid attack` to participate!"))
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
		medals := []string{"🥇", "🥈", "🥉"}
		for idx, r := range rows {
			prefix := fmt.Sprintf("%d.", idx+1)
			if idx < len(medals) {
				prefix = medals[idx]
			}
			lines = append(lines, fmt.Sprintf("%s **%s** — %d damage (%d attacks)", prefix, r.Name, r.Damage, r.Attacks))
		}
		bossName := "Unknown Boss"
		if g.Boss != nil {
			bossName = g.Boss.Name
		}
		embed := createInfoEmbed(fmt.Sprintf("⚔️ Raid Leaderboard — %s", bossName), strings.Join(lines, "\n"))
		respondEmbed(s, i, embed)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: status, attack, spawn, leaderboard"))
	}
}

func handleQuote(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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
			respondEmbed(s, i, createErrorEmbed("Quote Error", "Quote text is required."))
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
		embed := createSuccessEmbed("Quote Added", fmt.Sprintf("*\"%s\"*", q.Text))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Author", Value: q.Author.Username, Inline: true},
			{Name: "Added By", Value: user.Username, Inline: true},
			{Name: "Quote ID", Value: fmt.Sprintf("#%d", q.ID), Inline: true},
		}
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: author.AvatarURL("256")}
		respondEmbed(s, i, embed)
	case "random":
		if len(g.Quotes) == 0 {
			respondEmbed(s, i, createInfoEmbed("📝 Quotes", "No quotes yet. Use `/quote add` to add one!"))
			return
		}
		q := g.Quotes[rand.Intn(len(g.Quotes))]
		embed := createFunEmbed("📝 Random Quote", fmt.Sprintf("*\"%s\"*", q.Text))
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Author", Value: q.Author.Username, Inline: true},
			{Name: "Added By", Value: q.AddedBy.Username, Inline: true},
			{Name: "Quote ID", Value: fmt.Sprintf("#%d", q.ID), Inline: true},
		}
		respondEmbed(s, i, embed)
	case "list":
		if len(g.Quotes) == 0 {
			respondEmbed(s, i, createInfoEmbed("📝 Quotes", "No quotes yet. Use `/quote add` to add one!"))
			return
		}
		fields := []*discordgo.MessageEmbedField{}
		for idx := len(g.Quotes) - 1; idx >= 0 && len(fields) < 10; idx-- {
			q := g.Quotes[idx]
			preview := q.Text
			if len(preview) > 80 {
				preview = preview[:77] + "..."
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("#%d — %s", q.ID, q.Author.Username),
				Value:  fmt.Sprintf("*\"%s\"*", preview),
				Inline: false,
			})
		}
		embed := createInfoEmbed(fmt.Sprintf("📝 Server Quotes (%d total)", len(g.Quotes)), "")
		embed.Fields = fields
		respondEmbed(s, i, embed)
	case "remove":
		if i.Member == nil || (i.Member.Permissions&discordgo.PermissionManageMessages) == 0 {
			respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Messages` permission to remove quotes."))
			return
		}
		id := int(optionInt(subOpts, "id", -1))
		if id < 0 {
			respondEmbed(s, i, createErrorEmbed("Quote Error", "Quote ID is required."))
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
			respondEmbed(s, i, createErrorEmbed("Not Found", fmt.Sprintf("No quote found with ID #%d.", id)))
			return
		}
		removed := g.Quotes[found]
		g.Quotes = append(g.Quotes[:found], g.Quotes[found+1:]...)
		all[i.GuildID] = g
		_ = writeData("quotes.json", all)
		embed := createSuccessEmbed("Quote Removed", fmt.Sprintf("Removed **#%d**: *\"%s\"* — %s", id, removed.Text, removed.Author.Username))
		respondEmbed(s, i, embed)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: add, random, list, remove"))
	}
}

func handleQuest(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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

	diffEmoji := map[string]string{"easy": "🟢", "medium": "🟡", "hard": "🔴", "legendary": "⭐"}

	switch sub {
	case "get":
		if u.CurrentQuest != nil {
			emoji := diffEmoji[u.CurrentQuest.Difficulty]
			if emoji == "" {
				emoji = "📜"
			}
			embed := createFunEmbed("📜 Active Quest", u.CurrentQuest.Task)
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Difficulty", Value: fmt.Sprintf("%s %s", emoji, u.CurrentQuest.Difficulty), Inline: true},
				{Name: "XP Reward", Value: fmt.Sprintf("%d XP", u.CurrentQuest.XP), Inline: true},
			}
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
			respondEmbed(s, i, embed)
			return
		}
		q := questTemplates[rand.Intn(len(questTemplates))]
		q.AssignedAt = time.Now().UnixMilli()
		u.CurrentQuest = &q
		all[user.ID] = u
		_ = writeData("quests.json", all)
		emoji := diffEmoji[q.Difficulty]
		if emoji == "" {
			emoji = "📜"
		}
		embed := createFunEmbed("📜 New Quest Assigned!", q.Task)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Difficulty", Value: fmt.Sprintf("%s %s", emoji, q.Difficulty), Inline: true},
			{Name: "XP Reward", Value: fmt.Sprintf("%d XP", q.XP), Inline: true},
		}
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		respondEmbed(s, i, embed)
	case "complete":
		if u.CurrentQuest == nil {
			respondEmbed(s, i, createInfoEmbed("📜 No Active Quest", "You don't have an active quest. Use `/quest get` to get one!"))
			return
		}
		q := u.CurrentQuest
		oldLevel := (u.TotalXP / 100) + 1
		u.TotalXP += q.XP
		u.CompletedQuests++
		u.CurrentQuest = nil
		all[user.ID] = u
		_ = writeData("quests.json", all)
		newLevel := (u.TotalXP / 100) + 1
		desc := fmt.Sprintf("*\"%s\"*", q.Task)
		if newLevel > oldLevel {
			desc += fmt.Sprintf("\n\n🆙 **LEVEL UP!** You are now Level **%d**!", newLevel)
		}
		embed := createSuccessEmbed("Quest Complete!", desc)
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "⭐ XP Earned", Value: fmt.Sprintf("+%d XP", q.XP), Inline: true},
			{Name: "📊 Total XP", Value: fmt.Sprintf("%d XP", u.TotalXP), Inline: true},
			{Name: "🏆 Level", Value: fmt.Sprintf("%d", newLevel), Inline: true},
		}
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		respondEmbed(s, i, embed)
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
			respondEmbed(s, i, createInfoEmbed("🏆 Quest Leaderboard", "No quest completions yet."))
			return
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].TotalXP > rows[b].TotalXP })
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		medals := []string{"🥇", "🥈", "🥉"}
		for idx, r := range rows {
			prefix := fmt.Sprintf("%d.", idx+1)
			if idx < len(medals) {
				prefix = medals[idx]
			}
			lines = append(lines, fmt.Sprintf("%s **%s** — Lvl %d (%d XP, %d quests)", prefix, r.Name, (r.TotalXP/100)+1, r.TotalXP, r.Completed))
		}
		embed := createInfoEmbed("🏆 Quest Leaderboard", strings.Join(lines, "\n"))
		respondEmbed(s, i, embed)
	case "stats":
		level := (u.TotalXP / 100) + 1
		xpNext := level*100 - u.TotalXP
		embed := createInfoEmbed(fmt.Sprintf("📊 %s's Quest Stats", user.Username), "")
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "🏆 Level", Value: fmt.Sprintf("%d", level), Inline: true},
			{Name: "⭐ Total XP", Value: fmt.Sprintf("%d", u.TotalXP), Inline: true},
			{Name: "📈 XP to Next Level", Value: fmt.Sprintf("%d", xpNext), Inline: true},
			{Name: "✅ Quests Completed", Value: fmt.Sprintf("%d", u.CompletedQuests), Inline: true},
		}
		if u.CurrentQuest != nil {
			emoji := diffEmoji[u.CurrentQuest.Difficulty]
			if emoji == "" {
				emoji = "📜"
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "📜 Current Quest",
				Value:  fmt.Sprintf("%s %s (%s)", emoji, u.CurrentQuest.Task, u.CurrentQuest.Difficulty),
				Inline: false,
			})
		}
		respondEmbed(s, i, embed)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: get, complete, leaderboard, stats"))
	}
}

func handleLoot(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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

	rarityColors := map[string]int{
		"common": ColorGray, "uncommon": ColorGreen, "rare": ColorBlue,
		"epic": ColorPurple, "legendary": ColorGold, "cosmic": 0x00FFFF, "cursed": ColorRed,
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
		color := rarityColors[rarity]
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("📦 Loot Chest Opened!"),
			Description: fmt.Sprintf("You received: %s **%s**\n*%s*", table.Emoji, item, table.Name),
			Color:       color,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Rarity", Value: fmt.Sprintf("%s %s", table.Emoji, table.Name), Inline: true},
				{Name: "Total Loots", Value: fmt.Sprintf("%d", u.TotalLoots), Inline: true},
			},
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    &discordgo.MessageEmbedFooter{Text: "Discorbo"},
		}
		respondEmbed(s, i, embed)
	case "inventory":
		if len(u.Inventory) == 0 {
			respondEmbed(s, i, createInfoEmbed("📦 Loot Inventory", "Your inventory is empty. Use `/loot open` to get items!"))
			return
		}
		lines := []string{}
		for idx := len(u.Inventory) - 1; idx >= 0 && len(lines) < 15; idx-- {
			it := u.Inventory[idx]
			table := lootTables[it.Rarity]
			lines = append(lines, fmt.Sprintf("%s **%s** (%s)", table.Emoji, it.Item, table.Name))
		}
		embed := createInfoEmbed(fmt.Sprintf("📦 %s's Loot Inventory", user.Username), strings.Join(lines, "\n"))
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("%d items total | Discorbo", len(u.Inventory))}
		respondEmbed(s, i, embed)
	case "stats":
		total := u.TotalLoots
		pct := func(n int) string {
			if total == 0 {
				return "0%"
			}
			return fmt.Sprintf("%.1f%%", float64(n)/float64(total)*100)
		}
		embed := createInfoEmbed(fmt.Sprintf("📊 %s's Loot Stats", user.Username), fmt.Sprintf("**Total Loots:** %d", total))
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(user)}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "⚪ Common", Value: fmt.Sprintf("%d (%s)", u.Stats.Common, pct(u.Stats.Common)), Inline: true},
			{Name: "🟢 Uncommon", Value: fmt.Sprintf("%d (%s)", u.Stats.Uncommon, pct(u.Stats.Uncommon)), Inline: true},
			{Name: "🔵 Rare", Value: fmt.Sprintf("%d (%s)", u.Stats.Rare, pct(u.Stats.Rare)), Inline: true},
			{Name: "🟣 Epic", Value: fmt.Sprintf("%d (%s)", u.Stats.Epic, pct(u.Stats.Epic)), Inline: true},
			{Name: "🟡 Legendary", Value: fmt.Sprintf("%d (%s)", u.Stats.Legendary, pct(u.Stats.Legendary)), Inline: true},
			{Name: "🌌 Cosmic", Value: fmt.Sprintf("%d (%s)", u.Stats.Cosmic, pct(u.Stats.Cosmic)), Inline: true},
			{Name: "💀 Cursed", Value: fmt.Sprintf("%d (%s)", u.Stats.Cursed, pct(u.Stats.Cursed)), Inline: true},
		}
		respondEmbed(s, i, embed)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Valid options: open, inventory, stats"))
	}
}

func handleTag(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
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
			respondEmbed(s, i, createErrorEmbed("Tag Error", "You must specify an opponent."))
			return
		}
		if opponent.Bot {
			respondEmbed(s, i, createErrorEmbed("Tag Error", "You cannot challenge bots."))
			return
		}
		if opponent.ID == user.ID {
			respondEmbed(s, i, createErrorEmbed("Tag Error", "You cannot challenge yourself."))
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

		// Use deferred response for reliable message tracking
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return
		}

		// Send followup message which we can reliably track
		followup, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{buttons},
		})

		if err != nil || followup == nil {
			return
		}

		// Store session with reliable message ID
		sessionMu.Lock()
		tagSessions[followup.ID] = &tagSession{
			SessionID:   sessionID,
			Player1ID:   user.ID,
			Player1Name: user.Username,
			Player2ID:   opponent.ID,
			Player2Name: opponent.Username,
			MessageID:   followup.ID,
			ChannelID:   i.ChannelID,
		}
		sessionMu.Unlock()

		// Auto-cleanup after 5 minutes
		go func(msgID string) {
			time.Sleep(5 * time.Minute)
			sessionMu.Lock()
			delete(tagSessions, msgID)
			sessionMu.Unlock()
		}(followup.ID)

	case "leaderboard":
		tagStatsMap := map[string]tagStats{}
		_ = readData("tag-stats.json", &tagStatsMap)

		if len(tagStatsMap) == 0 {
			respondEmbed(s, i, createInfoEmbed("🏆 Tag Leaderboard", "No tag games played yet!"))
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
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer:      &discordgo.MessageEmbedFooter{Text: "Discorbo"},
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
			respondEmbed(s, i, createInfoEmbed("🏃 Tag Stats", fmt.Sprintf("**%s** hasn't played any tag games yet!", targetUser.Username)))
			return
		}

		winRate := 0.0
		if stats.TotalGames > 0 {
			winRate = float64(stats.Wins) / float64(stats.TotalGames) * 100
		}

		embed := createInfoEmbed(fmt.Sprintf("🏃 %s's Tag Stats", stats.Username), "")
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(targetUser)}
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "🏆 Wins", Value: fmt.Sprintf("%d", stats.Wins), Inline: true},
			{Name: "💀 Losses", Value: fmt.Sprintf("%d", stats.Losses), Inline: true},
			{Name: "📊 Win Rate", Value: fmt.Sprintf("%.1f%%", winRate), Inline: true},
			{Name: "🎮 Total Games", Value: fmt.Sprintf("%d", stats.TotalGames), Inline: true},
			{Name: "💰 Coins Earned", Value: fmt.Sprintf("%d", stats.CoinsEarned), Inline: true},
		}
		respondEmbed(s, i, embed)

	default:
		respondText(s, i, "Unknown subcommand.")
	}
}

const ColorGold = 0xFFD700
