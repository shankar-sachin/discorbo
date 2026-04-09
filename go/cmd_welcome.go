package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// welcomeConfig is the per-guild welcome/leave configuration.
type welcomeConfig struct {
	Enabled          bool   `json:"enabled"`
	WelcomeChannelID string `json:"welcomeChannelId"`
	WelcomeMessage   string `json:"welcomeMessage"`
	LeaveChannelID   string `json:"leaveChannelId"`
	LeaveMessage     string `json:"leaveMessage"`
	AutoRoleID       string `json:"autoRoleId"`
	BannerURL        string `json:"bannerUrl"`
}

const welcomeFile = "welcome-config.json"

func defaultWelcomeConfig() welcomeConfig {
	return welcomeConfig{
		Enabled:        true,
		WelcomeMessage: "Welcome to {server}, {user}! You are member #{membercount}!",
		LeaveMessage:   "{username} has left the server. We now have {membercount} members.",
	}
}

func loadWelcomeConfigs() map[string]welcomeConfig {
	data := map[string]welcomeConfig{}
	_ = readData(welcomeFile, &data)
	return data
}

func saveWelcomeConfigs(data map[string]welcomeConfig) {
	_ = writeData(welcomeFile, data)
}

func getWelcomeConfig(guildID string) welcomeConfig {
	configs := loadWelcomeConfigs()
	if cfg, ok := configs[guildID]; ok {
		return cfg
	}
	return defaultWelcomeConfig()
}

func setWelcomeConfig(guildID string, cfg welcomeConfig) {
	configs := loadWelcomeConfigs()
	configs[guildID] = cfg
	saveWelcomeConfigs(configs)
}

// replacePlaceholders substitutes message placeholders.
func replacePlaceholders(msg string, user *discordgo.User, guild *discordgo.Guild) string {
	memberCount := 0
	if guild != nil {
		memberCount = guild.MemberCount
	}
	r := strings.NewReplacer(
		"{user}", "<@"+user.ID+">",
		"{username}", user.Username,
		"{server}", guild.Name,
		"{membercount}", fmt.Sprintf("%d", memberCount),
	)
	return r.Replace(msg)
}

// ── Slash command router ─────────────────────────────────────────────────────

func handleWelcomeCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	sub, opts := getSubcommand(i.ApplicationCommandData().Options)
	switch sub {
	case "setup":
		handleWelcomeSetup(s, i, opts)
	case "set-leave":
		handleWelcomeSetLeave(s, i, opts)
	case "set-role":
		handleWelcomeSetRole(s, i, opts)
	case "toggle":
		handleWelcomeToggle(s, i)
	case "test":
		handleWelcomeTest(s, i)
	case "set-image":
		handleWelcomeSetImage(s, i, opts)
	default:
		respondEmbed(s, i, createErrorEmbed("Unknown Subcommand", "Use `/welcome setup`, `set-leave`, `set-role`, `toggle`, `test`, or `set-image`."))
	}
}

// ── Subcommand handlers ──────────────────────────────────────────────────────

func handleWelcomeSetup(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionManageServer == 0 {
		respondEmbed(s, i, createErrorEmbed("Permission Denied", "You need **Manage Server** permission to use this command."))
		return
	}

	ch := optionChannel(opts, "channel")
	if ch == nil {
		respondEmbed(s, i, createErrorEmbed("Missing Channel", "Please specify a channel for welcome messages."))
		return
	}

	cfg := getWelcomeConfig(i.GuildID)
	cfg.WelcomeChannelID = ch.ID
	cfg.Enabled = true

	if msg := optionString(opts, "message", ""); msg != "" {
		cfg.WelcomeMessage = msg
	}

	setWelcomeConfig(i.GuildID, cfg)
	respondEmbed(s, i, createSuccessEmbed("Welcome Setup Complete",
		fmt.Sprintf("Welcome messages will be sent to <#%s>.\n**Message:** %s", ch.ID, cfg.WelcomeMessage)))
}

func handleWelcomeSetLeave(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionManageServer == 0 {
		respondEmbed(s, i, createErrorEmbed("Permission Denied", "You need **Manage Server** permission to use this command."))
		return
	}

	ch := optionChannel(opts, "channel")
	if ch == nil {
		respondEmbed(s, i, createErrorEmbed("Missing Channel", "Please specify a channel for leave messages."))
		return
	}

	cfg := getWelcomeConfig(i.GuildID)
	cfg.LeaveChannelID = ch.ID

	if msg := optionString(opts, "message", ""); msg != "" {
		cfg.LeaveMessage = msg
	}

	setWelcomeConfig(i.GuildID, cfg)
	respondEmbed(s, i, createSuccessEmbed("Leave Setup Complete",
		fmt.Sprintf("Leave messages will be sent to <#%s>.\n**Message:** %s", ch.ID, cfg.LeaveMessage)))
}

func handleWelcomeSetRole(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionManageRoles == 0 {
		respondEmbed(s, i, createErrorEmbed("Permission Denied", "You need **Manage Roles** permission to use this command."))
		return
	}

	role := optionRole(opts, "role")
	if role == nil {
		respondEmbed(s, i, createErrorEmbed("Missing Role", "Please specify a role to assign on join."))
		return
	}

	cfg := getWelcomeConfig(i.GuildID)
	cfg.AutoRoleID = role.ID
	setWelcomeConfig(i.GuildID, cfg)
	respondEmbed(s, i, createSuccessEmbed("Auto-Role Set",
		fmt.Sprintf("New members will automatically receive the **%s** role.", role.Name)))
}

func handleWelcomeToggle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionManageServer == 0 {
		respondEmbed(s, i, createErrorEmbed("Permission Denied", "You need **Manage Server** permission to use this command."))
		return
	}

	cfg := getWelcomeConfig(i.GuildID)
	cfg.Enabled = !cfg.Enabled
	setWelcomeConfig(i.GuildID, cfg)

	status := "disabled"
	if cfg.Enabled {
		status = "enabled"
	}
	respondEmbed(s, i, createSuccessEmbed("Welcome Messages Toggled",
		fmt.Sprintf("Welcome/leave messages are now **%s**.", status)))
}

func handleWelcomeTest(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg := getWelcomeConfig(i.GuildID)
	if cfg.WelcomeChannelID == "" {
		respondEmbed(s, i, createErrorEmbed("Not Configured", "Run `/welcome setup` first to set a welcome channel."))
		return
	}

	guild, err := s.Guild(i.GuildID)
	if err != nil {
		respondEmbed(s, i, createErrorEmbed("Error", "Could not fetch server information."))
		return
	}

	user := interactionUser(i)
	msg := replacePlaceholders(cfg.WelcomeMessage, user, guild)

	embed := createWelcomeEmbed("👋 Welcome!", msg)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("256")}
	if cfg.BannerURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: cfg.BannerURL}
	}

	respondEmbed(s, i, embed)
}

func handleWelcomeSetImage(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionManageServer == 0 {
		respondEmbed(s, i, createErrorEmbed("Permission Denied", "You need **Manage Server** permission to use this command."))
		return
	}

	url := optionString(opts, "url", "")
	if url == "" {
		respondEmbed(s, i, createErrorEmbed("Missing URL", "Please provide a banner image URL."))
		return
	}

	cfg := getWelcomeConfig(i.GuildID)
	cfg.BannerURL = url
	setWelcomeConfig(i.GuildID, cfg)
	respondEmbed(s, i, createSuccessEmbed("Banner Image Set",
		fmt.Sprintf("Welcome banner updated.\n[Preview](%s)", url)))
}

// ── Event handlers ───────────────────────────────────────────────────────────

func handleGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	cfg := getWelcomeConfig(m.GuildID)
	if !cfg.Enabled || cfg.WelcomeChannelID == "" {
		// Still try auto-role even if welcome messages are off
		if cfg.AutoRoleID != "" {
			_ = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, cfg.AutoRoleID)
		}
		return
	}

	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return
	}

	msg := replacePlaceholders(cfg.WelcomeMessage, m.User, guild)
	embed := createWelcomeEmbed("👋 Welcome!", msg)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: m.User.AvatarURL("256")}
	if cfg.BannerURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: cfg.BannerURL}
	}

	_, _ = s.ChannelMessageSendEmbed(cfg.WelcomeChannelID, embed)

	// Assign auto-role
	if cfg.AutoRoleID != "" {
		_ = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, cfg.AutoRoleID)
	}
}

func handleGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	cfg := getWelcomeConfig(m.GuildID)
	if !cfg.Enabled || cfg.LeaveChannelID == "" {
		return
	}

	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return
	}

	msg := replacePlaceholders(cfg.LeaveMessage, m.User, guild)
	embed := createWelcomeEmbed("👋 Goodbye!", msg)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: m.User.AvatarURL("256")}

	_, _ = s.ChannelMessageSendEmbed(cfg.LeaveChannelID, embed)
}
