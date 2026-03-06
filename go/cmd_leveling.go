package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ─── Leveling Data Types ────────────────────────────────────────────────────────

type levelingUserData struct {
	Username      string `json:"username"`
	XP            int    `json:"xp"`
	Level         int    `json:"level"`
	TotalMessages int    `json:"totalMessages"`
	LastXPTime    int64  `json:"lastXPTime"`
}

type levelingGuildData struct {
	Enabled    bool                        `json:"enabled"`
	ChannelID  string                      `json:"channelId"`
	Multiplier float64                     `json:"multiplier"`
	Rewards    map[string]string           `json:"rewards"`
	Users      map[string]levelingUserData `json:"users"`
}

type levelingData struct {
	Guilds map[string]levelingGuildData `json:"guilds"`
}

const levelingFile = "leveling-data.json"

// xpForLevel returns XP needed to reach level N: 100 * N^1.5
func xpForLevel(level int) int {
	if level <= 0 {
		return 0
	}
	return int(100 * math.Pow(float64(level), 1.5))
}

func loadLevelingData() levelingData {
	var data levelingData
	if err := readData(levelingFile, &data); err != nil || data.Guilds == nil {
		data.Guilds = make(map[string]levelingGuildData)
	}
	return data
}

func saveLevelingData(data levelingData) {
	_ = writeData(levelingFile, data)
}

func ensureGuild(data *levelingData, guildID string) {
	if _, ok := data.Guilds[guildID]; !ok {
		data.Guilds[guildID] = levelingGuildData{
			Enabled:    true,
			Multiplier: 1.0,
			Rewards:    make(map[string]string),
			Users:      make(map[string]levelingUserData),
		}
	}
}

// ─── Command Router ─────────────────────────────────────────────────────────────

func handleLevelCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	sub := opts[0]
	subOpts := sub.Options

	switch sub.Name {
	case "view":
		handleLevelView(s, i, subOpts)
	case "leaderboard":
		handleLevelLeaderboard(s, i)
	case "rewards":
		handleLevelRewards(s, i)
	case "set-reward":
		handleLevelSetRewards(s, i, subOpts)
	case "set-channel":
		handleLevelSetChannel(s, i, subOpts)
	case "toggle":
		handleLevelToggle(s, i)
	case "reset":
		handleLevelReset(s, i, subOpts)
	case "set-multiplier":
		handleLevelSetMultiplier(s, i, subOpts)
	}
}

// ─── /level view ────────────────────────────────────────────────────────────────

func handleLevelView(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.GuildID == "" {
		respondEmbed(s, i, createErrorEmbed("Error", "This command can only be used in a server."))
		return
	}

	target := optionUser(opts, "user")
	caller := interactionUser(i)
	if target == nil {
		target = caller
	}
	if target == nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Unable to identify user."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	guild := data.Guilds[i.GuildID]

	if !guild.Enabled {
		respondEmbed(s, i, createErrorEmbed("Leveling Disabled", "The leveling system is not enabled in this server."))
		return
	}

	userData, ok := guild.Users[target.ID]
	if !ok {
		userData = levelingUserData{Username: target.Username}
	}

	currentLevelXP := xpForLevel(userData.Level)
	nextLevelXP := xpForLevel(userData.Level + 1)
	xpProgress := userData.XP - currentLevelXP
	xpNeeded := nextLevelXP - currentLevelXP
	if xpNeeded <= 0 {
		xpNeeded = 1
	}
	if xpProgress < 0 {
		xpProgress = 0
	}

	// Calculate rank in guild
	type rankedUser struct {
		ID    string
		Level int
		XP    int
	}
	var ranked []rankedUser
	for uid, u := range guild.Users {
		ranked = append(ranked, rankedUser{ID: uid, Level: u.Level, XP: u.XP})
	}
	sort.Slice(ranked, func(a, b int) bool {
		if ranked[a].Level != ranked[b].Level {
			return ranked[a].Level > ranked[b].Level
		}
		return ranked[a].XP > ranked[b].XP
	})
	rank := 0
	for idx, r := range ranked {
		if r.ID == target.ID {
			rank = idx + 1
			break
		}
	}
	if rank == 0 {
		rank = len(ranked) + 1
	}

	xpBar := renderXPBar(xpProgress, xpNeeded)

	desc := fmt.Sprintf("**%s**\n\n", target.Username) +
		fmt.Sprintf("⭐ **Level:** %d\n", userData.Level) +
		fmt.Sprintf("🏆 **Rank:** #%d\n", rank) +
		fmt.Sprintf("✨ **Total XP:** %d\n", userData.XP) +
		fmt.Sprintf("💬 **Messages:** %d\n\n", userData.TotalMessages) +
		fmt.Sprintf("**Progress to Level %d:**\n%s", userData.Level+1, xpBar)

	embed := richEmbed(richEmbedOpts{
		Title:        "📊 Level Card",
		Description:  desc,
		Color:        ColorLevel,
		Category:     "⭐ Leveling",
		ThumbnailURL: userAvatar(target),
		AuthorName:   target.Username,
		AuthorIcon:   userAvatar(target),
	})
	respondEmbed(s, i, embed)
}

// ─── /level leaderboard ─────────────────────────────────────────────────────────

func handleLevelLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		respondEmbed(s, i, createErrorEmbed("Error", "This command can only be used in a server."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	guild := data.Guilds[i.GuildID]

	if !guild.Enabled {
		respondEmbed(s, i, createErrorEmbed("Leveling Disabled", "The leveling system is not enabled in this server."))
		return
	}

	type entry struct {
		Username string
		Level    int
		XP       int
	}
	var entries []entry
	for _, u := range guild.Users {
		entries = append(entries, entry{Username: u.Username, Level: u.Level, XP: u.XP})
	}
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].Level != entries[b].Level {
			return entries[a].Level > entries[b].Level
		}
		return entries[a].XP > entries[b].XP
	})

	if len(entries) == 0 {
		respondEmbed(s, i, createLevelEmbed("🏆 Leaderboard", "No users have earned XP yet."))
		return
	}

	medals := []string{"🥇", "🥈", "🥉"}
	desc := ""
	limit := 10
	if len(entries) < limit {
		limit = len(entries)
	}
	for idx := 0; idx < limit; idx++ {
		e := entries[idx]
		prefix := fmt.Sprintf("`#%d`", idx+1)
		if idx < 3 {
			prefix = medals[idx]
		}
		desc += fmt.Sprintf("%s **%s** — Level %d (%d XP)\n", prefix, e.Username, e.Level, e.XP)
	}

	respondEmbed(s, i, createLevelEmbed("🏆 XP Leaderboard", desc))
}

// ─── /level rewards ─────────────────────────────────────────────────────────────

func handleLevelRewards(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		respondEmbed(s, i, createErrorEmbed("Error", "This command can only be used in a server."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	guild := data.Guilds[i.GuildID]

	if len(guild.Rewards) == 0 {
		respondEmbed(s, i, createLevelEmbed("🎁 Level Rewards", "No role rewards have been configured yet.\nAdmins can set them with `/level set-reward`."))
		return
	}

	// Sort reward levels numerically
	type reward struct {
		Level  int
		RoleID string
	}
	var rewards []reward
	for lvlStr, roleID := range guild.Rewards {
		lvl, err := strconv.Atoi(lvlStr)
		if err != nil {
			continue
		}
		rewards = append(rewards, reward{Level: lvl, RoleID: roleID})
	}
	sort.Slice(rewards, func(a, b int) bool { return rewards[a].Level < rewards[b].Level })

	desc := ""
	for _, r := range rewards {
		desc += fmt.Sprintf("⭐ **Level %d** → <@&%s>\n", r.Level, r.RoleID)
	}

	respondEmbed(s, i, createLevelEmbed("🎁 Level Rewards", desc))
}

// ─── /level set-reward (Admin) ──────────────────────────────────────────────────

func handleLevelSetRewards(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil || !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageServer) {
		respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Server` permission to use this command."))
		return
	}

	level := int(optionInt(opts, "level", 0))
	role := optionRole(opts, "role")
	if level <= 0 || role == nil {
		respondEmbed(s, i, createErrorEmbed("Invalid Input", "Please provide a valid level (> 0) and role."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	g := data.Guilds[i.GuildID]
	if g.Rewards == nil {
		g.Rewards = make(map[string]string)
	}
	g.Rewards[strconv.Itoa(level)] = role.ID
	data.Guilds[i.GuildID] = g
	saveLevelingData(data)

	respondEmbed(s, i, createSuccessEmbed("Reward Set", fmt.Sprintf("Users reaching **Level %d** will receive <@&%s>.", level, role.ID)))
}

// ─── /level set-channel (Admin) ─────────────────────────────────────────────────

func handleLevelSetChannel(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil || !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageServer) {
		respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Server` permission to use this command."))
		return
	}

	ch := optionChannel(opts, "channel")
	if ch == nil {
		respondEmbed(s, i, createErrorEmbed("Invalid Input", "Please provide a valid channel."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	g := data.Guilds[i.GuildID]
	g.ChannelID = ch.ID
	data.Guilds[i.GuildID] = g
	saveLevelingData(data)

	respondEmbed(s, i, createSuccessEmbed("Channel Set", fmt.Sprintf("Level-up announcements will be sent to <#%s>.", ch.ID)))
}

// ─── /level toggle (Admin) ──────────────────────────────────────────────────────

func handleLevelToggle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil || !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageServer) {
		respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Server` permission to use this command."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	g := data.Guilds[i.GuildID]
	g.Enabled = !g.Enabled
	data.Guilds[i.GuildID] = g
	saveLevelingData(data)

	status := "disabled"
	if g.Enabled {
		status = "enabled"
	}
	respondEmbed(s, i, createSuccessEmbed("Leveling Toggled", fmt.Sprintf("The leveling system is now **%s**.", status)))
}

// ─── /level reset (Admin) ───────────────────────────────────────────────────────

func handleLevelReset(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil || !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageServer) {
		respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Server` permission to use this command."))
		return
	}

	target := optionUser(opts, "user")
	if target == nil {
		respondEmbed(s, i, createErrorEmbed("Invalid Input", "Please specify a user to reset."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	g := data.Guilds[i.GuildID]
	delete(g.Users, target.ID)
	data.Guilds[i.GuildID] = g
	saveLevelingData(data)

	respondEmbed(s, i, createSuccessEmbed("XP Reset", fmt.Sprintf("All XP and level data for **%s** has been reset.", target.Username)))
}

// ─── /level set-multiplier (Admin) ──────────────────────────────────────────────

func handleLevelSetMultiplier(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil || !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageServer) {
		respondEmbed(s, i, createErrorEmbed("Missing Permissions", "You need the `Manage Server` permission to use this command."))
		return
	}

	// Extract multiplier from Number option
	var multiplier float64
	for _, o := range opts {
		if o.Name == "multiplier" {
			multiplier = o.FloatValue()
			break
		}
	}
	if multiplier < 0.5 || multiplier > 5.0 {
		respondEmbed(s, i, createErrorEmbed("Invalid Value", "Multiplier must be between **0.5** and **5.0**."))
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, i.GuildID)
	g := data.Guilds[i.GuildID]
	g.Multiplier = multiplier
	data.Guilds[i.GuildID] = g
	saveLevelingData(data)

	respondEmbed(s, i, createSuccessEmbed("Multiplier Set", fmt.Sprintf("XP multiplier is now **%.1fx**.", multiplier)))
}

// ─── awardXP (called from messageCreate) ────────────────────────────────────────

func awardXP(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.GuildID == "" || m.Author == nil || m.Author.Bot {
		return
	}

	data := loadLevelingData()
	ensureGuild(&data, m.GuildID)
	guild := data.Guilds[m.GuildID]

	if !guild.Enabled {
		return
	}

	if guild.Users == nil {
		guild.Users = make(map[string]levelingUserData)
	}

	userData := guild.Users[m.Author.ID]
	userData.Username = m.Author.Username

	// 60-second cooldown per user per guild
	now := time.Now().UnixMilli()
	if now-userData.LastXPTime < 60000 {
		// Still count message even if on cooldown
		userData.TotalMessages++
		guild.Users[m.Author.ID] = userData
		data.Guilds[m.GuildID] = guild
		saveLevelingData(data)
		return
	}

	// Award 15-25 XP with multiplier
	baseXP := 15 + rand.Intn(11) // 15-25
	xpGain := int(float64(baseXP) * guild.Multiplier)

	userData.XP += xpGain
	userData.TotalMessages++
	userData.LastXPTime = now

	// Check for level up
	oldLevel := userData.Level
	for userData.XP >= xpForLevel(userData.Level+1) {
		userData.Level++
	}

	guild.Users[m.Author.ID] = userData
	data.Guilds[m.GuildID] = guild
	saveLevelingData(data)

	// Handle level up
	if userData.Level > oldLevel {
		announceLevelUp(s, m, guild, userData, oldLevel)
		awardLevelRoles(s, m.GuildID, m.Author.ID, userData.Level, guild.Rewards)
	}
}

func announceLevelUp(s *discordgo.Session, m *discordgo.MessageCreate, guild levelingGuildData, userData levelingUserData, oldLevel int) {
	desc := fmt.Sprintf("🎉 <@%s> reached **Level %d**!", m.Author.ID, userData.Level)

	// Check if a reward role is given at this level
	for lvl := oldLevel + 1; lvl <= userData.Level; lvl++ {
		if roleID, ok := guild.Rewards[strconv.Itoa(lvl)]; ok {
			desc += fmt.Sprintf("\n🎁 Unlocked role: <@&%s>", roleID)
		}
	}

	embed := createLevelEmbed("🎊 Level Up!", desc)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: m.Author.AvatarURL("128")}

	channelID := guild.ChannelID
	if channelID == "" {
		channelID = m.ChannelID
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func awardLevelRoles(s *discordgo.Session, guildID, userID string, currentLevel int, rewards map[string]string) {
	for lvlStr, roleID := range rewards {
		lvl, err := strconv.Atoi(lvlStr)
		if err != nil {
			continue
		}
		if currentLevel >= lvl {
			_ = s.GuildMemberRoleAdd(guildID, userID, roleID)
		}
	}
}
