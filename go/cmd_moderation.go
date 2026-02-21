package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// hasPermission checks if a user has a specific permission in a guild
func hasPermission(s *discordgo.Session, guildID, userID string, perm int64) bool {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	// Administrator bypasses all permission checks
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	// Check specific permission
	return member.Permissions&perm != 0
}

// canModerate checks if moderator can moderate target (hierarchy check)
func canModerate(s *discordgo.Session, guildID string, moderator, target *discordgo.Member) bool {
	// Can't moderate yourself
	if moderator.User.ID == target.User.ID {
		return false
	}

	// Get guild to check owner
	guild, err := s.Guild(guildID)
	if err != nil {
		return false
	}

	// Owner can moderate anyone
	if moderator.User.ID == guild.OwnerID {
		return true
	}

	// Can't moderate server owner
	if target.User.ID == guild.OwnerID {
		return false
	}

	// Get highest roles for both users
	modHighest := getHighestRole(s, guildID, moderator)
	targetHighest := getHighestRole(s, guildID, target)

	// Compare role positions
	return modHighest.Position > targetHighest.Position
}

// getHighestRole returns the highest role for a member
func getHighestRole(s *discordgo.Session, guildID string, member *discordgo.Member) *discordgo.Role {
	guild, err := s.Guild(guildID)
	if err != nil || len(member.Roles) == 0 {
		// Return @everyone role
		for _, role := range guild.Roles {
			if role.ID == guildID {
				return role
			}
		}
		return &discordgo.Role{Position: 0}
	}

	var highest *discordgo.Role
	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				if highest == nil || role.Position > highest.Position {
					highest = role
				}
			}
		}
	}

	if highest == nil {
		highest = &discordgo.Role{Position: 0}
	}
	return highest
}

// notifyUser sends a DM to a user about moderation action
func notifyUser(s *discordgo.Session, userID, action, reason, guildName string) error {
	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return err // User has DMs disabled
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("You were %s from %s", action, guildName),
		Description: fmt.Sprintf("**Reason:** %s", reason),
		Color:       ColorRed,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err = s.ChannelMessageSendEmbed(ch.ID, embed)
	return err
}

// logModAction logs a moderation action to the mod log channel and database
func logModAction(s *discordgo.Session, guildID string, action modAction) {
	// Save to database
	actions := []modAction{}
	_ = readData("mod-actions.json", &actions)
	actions = append(actions, action)
	_ = writeData("mod-actions.json", actions)

	// Send to mod log channel if configured
	configs := map[string]guildConfig{}
	_ = readData("guild-config.json", &configs)

	config, exists := configs[guildID]
	if !exists || config.ModLogChannel == "" {
		return
	}

	// Create embed for mod log
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🛡️ Moderation: %s", strings.ToUpper(action.Type)),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "User", Value: fmt.Sprintf("<@%s> (%s)", action.UserID, action.Username), Inline: true},
			{Name: "Moderator", Value: fmt.Sprintf("<@%s>", action.ModeratorID), Inline: true},
			{Name: "Reason", Value: action.Reason, Inline: false},
		},
		Color:     ColorYellow,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("ID: %s", action.ID)},
	}

	// Add duration for timeouts
	if action.Duration > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  formatDuration(action.Duration),
			Inline: true,
		})
	}

	// Add message count for purge
	if action.MessageCount > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Messages Deleted",
			Value:  fmt.Sprintf("%d", action.MessageCount),
			Inline: true,
		})
	}

	_, _ = s.ChannelMessageSendEmbed(config.ModLogChannel, embed)
}

// formatDuration converts seconds to human readable duration
func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d minutes", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%d hours", seconds/3600)
	}
	return fmt.Sprintf("%d days", seconds/86400)
}

// parseDuration converts duration string like "5m", "1h", "7d" to seconds
func parseDuration(duration string) int64 {
	if len(duration) < 2 {
		return 0
	}

	unit := duration[len(duration)-1]
	value := duration[:len(duration)-1]

	var multiplier int64
	switch unit {
	case 'm':
		multiplier = 60
	case 'h':
		multiplier = 3600
	case 'd':
		multiplier = 86400
	default:
		return 0
	}

	// Simple integer parsing
	var num int64
	fmt.Sscanf(value, "%d", &num)
	return num * multiplier
}

// generateID generates a unique ID for actions/warnings
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ============================================================================
// KICK COMMAND
// ============================================================================

func handleKick(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionKickMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Kick Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	// Bot permission check
	botPerms, err := s.UserChannelPermissions(s.State.User.ID, i.ChannelID)
	if err != nil || botPerms&discordgo.PermissionKickMembers == 0 {
		embed := createErrorEmbed("Bot Missing Permissions", "I don't have permission to kick members.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user to kick.")
		return
	}

	if targetUser.Bot {
		respondText(s, i, "You cannot kick bots.")
		return
	}

	reason := optionString(opts, "reason", "")
	if reason == "" {
		reason = "No reason provided"
	}

	// Get members for hierarchy check
	moderator, err := s.GuildMember(i.GuildID, user.ID)
	if err != nil {
		respondText(s, i, "Failed to fetch moderator information.")
		return
	}

	target, err := s.GuildMember(i.GuildID, targetUser.ID)
	if err != nil {
		respondText(s, i, "Failed to fetch target member information.")
		return
	}

	// Hierarchy check
	if !canModerate(s, i.GuildID, moderator, target) {
		embed := createErrorEmbed("Cannot Moderate", "You cannot kick this user (they have a higher or equal role).")
		respondEmbed(s, i, embed)
		return
	}

	// Get guild name for DM
	guild, _ := s.Guild(i.GuildID)
	guildName := "the server"
	if guild != nil {
		guildName = guild.Name
	}

	// Notify user via DM
	_ = notifyUser(s, targetUser.ID, "kicked", reason, guildName)

	// Perform kick
	err = s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, reason)
	if err != nil {
		embed := createErrorEmbed("Kick Failed", fmt.Sprintf("Failed to kick user: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	action := modAction{
		ID:          generateID("kick"),
		Type:        "kick",
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Member Kicked", fmt.Sprintf("**%s** has been kicked.\n**Reason:** %s", targetUser.Username, reason))
	respondEmbed(s, i, embed)
}

// ============================================================================
// BAN COMMAND
// ============================================================================

func handleBan(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionBanMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Ban Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	// Bot permission check
	botPerms, err := s.UserChannelPermissions(s.State.User.ID, i.ChannelID)
	if err != nil || botPerms&discordgo.PermissionBanMembers == 0 {
		embed := createErrorEmbed("Bot Missing Permissions", "I don't have permission to ban members.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user to ban.")
		return
	}

	if targetUser.Bot {
		respondText(s, i, "You cannot ban bots.")
		return
	}

	reason := optionString(opts, "reason", "")
	if reason == "" {
		reason = "No reason provided"
	}

	deleteDays := int(optionInt(opts, "delete_days", 0))
	if deleteDays < 0 {
		deleteDays = 0
	}
	if deleteDays > 7 {
		deleteDays = 7
	}

	// Get members for hierarchy check
	moderator, err := s.GuildMember(i.GuildID, user.ID)
	if err != nil {
		respondText(s, i, "Failed to fetch moderator information.")
		return
	}

	target, err := s.GuildMember(i.GuildID, targetUser.ID)
	if err != nil {
		// User might not be in server, allow ban anyway for ban by ID
		target = &discordgo.Member{User: targetUser, GuildID: i.GuildID}
	} else {
		// Hierarchy check only if user is in server
		if !canModerate(s, i.GuildID, moderator, target) {
			embed := createErrorEmbed("Cannot Moderate", "You cannot ban this user (they have a higher or equal role).")
			respondEmbed(s, i, embed)
			return
		}
	}

	// Get guild name for DM
	guild, _ := s.Guild(i.GuildID)
	guildName := "the server"
	if guild != nil {
		guildName = guild.Name
	}

	// Notify user via DM
	_ = notifyUser(s, targetUser.ID, "banned", reason, guildName)

	// Perform ban
	err = s.GuildBanCreateWithReason(i.GuildID, targetUser.ID, reason, deleteDays)
	if err != nil {
		embed := createErrorEmbed("Ban Failed", fmt.Sprintf("Failed to ban user: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	action := modAction{
		ID:          generateID("ban"),
		Type:        "ban",
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	deleteMsg := ""
	if deleteDays > 0 {
		deleteMsg = fmt.Sprintf("\n**Messages deleted:** Last %d day(s)", deleteDays)
	}
	embed := createSuccessEmbed("Member Banned", fmt.Sprintf("**%s** has been banned.\n**Reason:** %s%s", targetUser.Username, reason, deleteMsg))
	respondEmbed(s, i, embed)
}

// ============================================================================
// UNBAN COMMAND
// ============================================================================

func handleUnban(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionBanMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Ban Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	userID := optionString(opts, "user_id", "")
	if userID == "" {
		respondText(s, i, "You must specify a user ID to unban.")
		return
	}

	reason := optionString(opts, "reason", "")
	if reason == "" {
		reason = "No reason provided"
	}

	// Perform unban
	err := s.GuildBanDelete(i.GuildID, userID)
	if err != nil {
		embed := createErrorEmbed("Unban Failed", fmt.Sprintf("Failed to unban user: %v\nMake sure the user ID is correct and they are currently banned.", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	action := modAction{
		ID:          generateID("unban"),
		Type:        "unban",
		UserID:      userID,
		Username:    "User " + userID,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Member Unbanned", fmt.Sprintf("User ID **%s** has been unbanned.\n**Reason:** %s", userID, reason))
	respondEmbed(s, i, embed)
}

// ============================================================================
// TIMEOUT COMMAND
// ============================================================================

func handleTimeout(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionModerateMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Moderate Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user to timeout.")
		return
	}

	if targetUser.Bot {
		respondText(s, i, "You cannot timeout bots.")
		return
	}

	duration := optionString(opts, "duration", "")
	if duration == "" {
		respondText(s, i, "You must specify a duration.")
		return
	}

	durationSeconds := parseDuration(duration)
	if durationSeconds == 0 || durationSeconds > 28*86400 {
		respondText(s, i, "Invalid duration. Max is 28 days.")
		return
	}

	reason := optionString(opts, "reason", "")
	if reason == "" {
		reason = "No reason provided"
	}

	// Get members for hierarchy check
	moderator, err := s.GuildMember(i.GuildID, user.ID)
	if err != nil {
		respondText(s, i, "Failed to fetch moderator information.")
		return
	}

	target, err := s.GuildMember(i.GuildID, targetUser.ID)
	if err != nil {
		respondText(s, i, "Failed to fetch target member information.")
		return
	}

	// Hierarchy check
	if !canModerate(s, i.GuildID, moderator, target) {
		embed := createErrorEmbed("Cannot Moderate", "You cannot timeout this user (they have a higher or equal role).")
		respondEmbed(s, i, embed)
		return
	}

	// Get guild name for DM
	guild, _ := s.Guild(i.GuildID)
	guildName := "the server"
	if guild != nil {
		guildName = guild.Name
	}

	// Notify user
	_ = notifyUser(s, targetUser.ID, fmt.Sprintf("timed out for %s", formatDuration(durationSeconds)), reason, guildName)

	// Calculate timeout end time
	timeoutUntil := time.Now().Add(time.Duration(durationSeconds) * time.Second)

	// Apply timeout
	err = s.GuildMemberTimeout(i.GuildID, targetUser.ID, &timeoutUntil)
	if err != nil {
		embed := createErrorEmbed("Timeout Failed", fmt.Sprintf("Failed to timeout user: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	action := modAction{
		ID:          generateID("timeout"),
		Type:        "timeout",
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Duration:    durationSeconds,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Member Timed Out", fmt.Sprintf("**%s** has been timed out for **%s**.\n**Reason:** %s", targetUser.Username, formatDuration(durationSeconds), reason))
	respondEmbed(s, i, embed)
}

// ============================================================================
// WARN COMMAND
// ============================================================================

func handleWarn(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionKickMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Kick Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user to warn.")
		return
	}

	if targetUser.Bot {
		respondText(s, i, "You cannot warn bots.")
		return
	}

	reason := optionString(opts, "reason", "")
	if reason == "" {
		respondText(s, i, "You must provide a reason for the warning.")
		return
	}

	// Create warning entry
	warning := warningEntry{
		ID:          generateID("warn"),
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}

	// Load warnings
	warnings := map[string]map[string][]warningEntry{}
	_ = readData("warnings.json", &warnings)

	// Initialize guild map if needed
	if warnings[i.GuildID] == nil {
		warnings[i.GuildID] = make(map[string][]warningEntry)
	}

	// Add warning
	warnings[i.GuildID][targetUser.ID] = append(warnings[i.GuildID][targetUser.ID], warning)

	// Save warnings
	_ = writeData("warnings.json", warnings)

	// Get total warning count
	totalWarnings := len(warnings[i.GuildID][targetUser.ID])

	// Notify user via DM
	guild, _ := s.Guild(i.GuildID)
	guildName := "the server"
	if guild != nil {
		guildName = guild.Name
	}

	ch, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("⚠️ You received a warning in %s", guildName),
			Description: fmt.Sprintf("**Reason:** %s\n**Total Warnings:** %d", reason, totalWarnings),
			Color:       ColorYellow,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		_, _ = s.ChannelMessageSendEmbed(ch.ID, embed)
	}

	// Log action
	action := modAction{
		ID:          warning.ID,
		Type:        "warn",
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Warning Issued", fmt.Sprintf("**%s** has been warned.\n**Reason:** %s\n**Total Warnings:** %d", targetUser.Username, reason, totalWarnings))
	respondEmbed(s, i, embed)
}

// ============================================================================
// WARNINGS COMMAND
// ============================================================================

func handleWarnings(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionKickMembers) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Kick Members` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user.")
		return
	}

	// Load warnings
	warnings := map[string]map[string][]warningEntry{}
	_ = readData("warnings.json", &warnings)

	// Get warnings for this user in this guild
	userWarnings := []warningEntry{}
	if warnings[i.GuildID] != nil {
		userWarnings = warnings[i.GuildID][targetUser.ID]
	}

	if len(userWarnings) == 0 {
		embed := createInfoEmbed("No Warnings", fmt.Sprintf("**%s** has no warnings.", targetUser.Username))
		respondEmbed(s, i, embed)
		return
	}

	// Build warnings list (show last 10)
	var warningsList string
	start := 0
	if len(userWarnings) > 10 {
		start = len(userWarnings) - 10
	}

	for idx := start; idx < len(userWarnings); idx++ {
		w := userWarnings[idx]
		timestamp := time.Unix(w.Timestamp, 0).Format("2006-01-02")
		warningsList += fmt.Sprintf("**#%d** - %s\n**Moderator:** %s\n**Reason:** %s\n\n", idx+1, timestamp, w.Moderator, w.Reason)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("⚠️ Warnings for %s", targetUser.Username),
		Description: fmt.Sprintf("**Total Warnings:** %d\n\n%s", len(userWarnings), warningsList),
		Color:       ColorYellow,
		Timestamp:   time.Now().Format(time.RFC3339),
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: userAvatar(targetUser)},
	}

	respondEmbed(s, i, embed)
}

// ============================================================================
// CLEARWARNINGS COMMAND
// ============================================================================

func handleClearWarnings(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionAdministrator) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Administrator` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetUser := optionUser(opts, "user")
	if targetUser == nil {
		respondText(s, i, "You must specify a user.")
		return
	}

	warningID := optionString(opts, "warning_id", "")

	// Load warnings
	warnings := map[string]map[string][]warningEntry{}
	_ = readData("warnings.json", &warnings)

	if warnings[i.GuildID] == nil || len(warnings[i.GuildID][targetUser.ID]) == 0 {
		embed := createInfoEmbed("No Warnings", fmt.Sprintf("**%s** has no warnings to clear.", targetUser.Username))
		respondEmbed(s, i, embed)
		return
	}

	var clearedCount int
	if warningID != "" {
		// Clear specific warning
		userWarnings := warnings[i.GuildID][targetUser.ID]
		newWarnings := []warningEntry{}
		for _, w := range userWarnings {
			if w.ID != warningID {
				newWarnings = append(newWarnings, w)
			} else {
				clearedCount++
			}
		}
		warnings[i.GuildID][targetUser.ID] = newWarnings
	} else {
		// Clear all warnings
		clearedCount = len(warnings[i.GuildID][targetUser.ID])
		delete(warnings[i.GuildID], targetUser.ID)
	}

	// Save warnings
	_ = writeData("warnings.json", warnings)

	if clearedCount == 0 {
		embed := createErrorEmbed("Warning Not Found", "Could not find the specified warning ID.")
		respondEmbed(s, i, embed)
		return
	}

	// Success response
	msg := "All warnings"
	if warningID != "" {
		msg = "1 warning"
	}
	embed := createSuccessEmbed("Warnings Cleared", fmt.Sprintf("%s cleared for **%s**.", msg, targetUser.Username))
	respondEmbed(s, i, embed)
}

// ============================================================================
// PURGE COMMAND
// ============================================================================

func handlePurge(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageMessages) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Messages` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	// Bot permission check
	botPerms, err := s.UserChannelPermissions(s.State.User.ID, i.ChannelID)
	if err != nil || botPerms&discordgo.PermissionManageMessages == 0 {
		embed := createErrorEmbed("Bot Missing Permissions", "I don't have permission to manage messages.")
		respondEmbed(s, i, embed)
		return
	}

	amount := int(optionInt(opts, "amount", 0))
	if amount < 2 || amount > 100 {
		respondText(s, i, "Amount must be between 2 and 100.")
		return
	}

	targetUser := optionUser(opts, "user")
	contains := optionString(opts, "contains", "")

	// Fetch messages
	messages, err := s.ChannelMessages(i.ChannelID, amount, "", "", "")
	if err != nil {
		embed := createErrorEmbed("Fetch Failed", fmt.Sprintf("Failed to fetch messages: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Filter messages
	var toDelete []string
	for _, msg := range messages {
		// Skip if filtering by user and message isn't from that user
		if targetUser != nil && msg.Author.ID != targetUser.ID {
			continue
		}

		// Skip if filtering by content and message doesn't contain text
		if contains != "" && !strings.Contains(strings.ToLower(msg.Content), strings.ToLower(contains)) {
			continue
		}

		toDelete = append(toDelete, msg.ID)
	}

	if len(toDelete) == 0 {
		respondText(s, i, "No messages found matching the filters.")
		return
	}

	// Delete messages
	err = s.ChannelMessagesBulkDelete(i.ChannelID, toDelete)
	if err != nil {
		embed := createErrorEmbed("Delete Failed", fmt.Sprintf("Failed to delete messages: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	action := modAction{
		ID:           generateID("purge"),
		Type:         "purge",
		UserID:       user.ID,
		Username:     user.Username,
		ModeratorID:  user.ID,
		Moderator:    user.Username,
		Reason:       fmt.Sprintf("Purged %d messages", len(toDelete)),
		MessageCount: len(toDelete),
		Timestamp:    time.Now().Unix(),
		GuildID:      i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response (ephemeral)
	embed := createSuccessEmbed("Messages Purged", fmt.Sprintf("Successfully deleted **%d** messages.", len(toDelete)))
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// ============================================================================
// LOCK COMMAND
// ============================================================================

func handleLock(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageChannels) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Channels` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetChannel := optionChannel(opts, "channel")
	channelID := i.ChannelID
	if targetChannel != nil {
		channelID = targetChannel.ID
	}

	reason := optionString(opts, "reason", "No reason provided")

	// Get the channel
	channel, err := s.Channel(channelID)
	if err != nil {
		embed := createErrorEmbed("Channel Not Found", "Could not find the specified channel.")
		respondEmbed(s, i, embed)
		return
	}

	// Find @everyone role ID (same as guild ID)
	everyoneRoleID := i.GuildID

	// Update channel permissions to deny SEND_MESSAGES for @everyone
	err = s.ChannelPermissionSet(channelID, everyoneRoleID, discordgo.PermissionOverwriteTypeRole,
		0, discordgo.PermissionSendMessages)
	if err != nil {
		embed := createErrorEmbed("Lock Failed", fmt.Sprintf("Failed to lock channel: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Send lock notification to channel
	lockEmbed := &discordgo.MessageEmbed{
		Title:       "🔒 Channel Locked",
		Description: fmt.Sprintf("**Reason:** %s\n**Moderator:** <@%s>", reason, user.ID),
		Color:       ColorRed,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, lockEmbed)

	// Log action
	action := modAction{
		ID:          generateID("lock"),
		Type:        "lock",
		UserID:      channelID,
		Username:    channel.Name,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Channel Locked", fmt.Sprintf("<#%s> has been locked.", channelID))
	respondEmbed(s, i, embed)
}

// ============================================================================
// UNLOCK COMMAND
// ============================================================================

func handleUnlock(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageChannels) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Channels` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetChannel := optionChannel(opts, "channel")
	channelID := i.ChannelID
	if targetChannel != nil {
		channelID = targetChannel.ID
	}

	// Get the channel
	channel, err := s.Channel(channelID)
	if err != nil {
		embed := createErrorEmbed("Channel Not Found", "Could not find the specified channel.")
		respondEmbed(s, i, embed)
		return
	}

	// Find @everyone role ID (same as guild ID)
	everyoneRoleID := i.GuildID

	// Remove the permission override (or set to neutral)
	err = s.ChannelPermissionDelete(channelID, everyoneRoleID)
	if err != nil {
		// If deletion fails, try setting to neutral (0 deny, 0 allow)
		err = s.ChannelPermissionSet(channelID, everyoneRoleID, discordgo.PermissionOverwriteTypeRole, 0, 0)
		if err != nil {
			embed := createErrorEmbed("Unlock Failed", fmt.Sprintf("Failed to unlock channel: %v", err))
			respondEmbed(s, i, embed)
			return
		}
	}

	// Send unlock notification to channel
	unlockEmbed := &discordgo.MessageEmbed{
		Title:       "🔓 Channel Unlocked",
		Description: fmt.Sprintf("**Moderator:** <@%s>", user.ID),
		Color:       ColorGreen,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, unlockEmbed)

	// Log action
	action := modAction{
		ID:          generateID("unlock"),
		Type:        "unlock",
		UserID:      channelID,
		Username:    channel.Name,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      "Channel unlocked",
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	embed := createSuccessEmbed("Channel Unlocked", fmt.Sprintf("<#%s> has been unlocked.", channelID))
	respondEmbed(s, i, embed)
}

// ============================================================================
// SLOWMODE COMMAND
// ============================================================================

func handleSlowmode(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionManageChannels) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Manage Channels` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	targetChannel := optionChannel(opts, "channel")
	channelID := i.ChannelID
	if targetChannel != nil {
		channelID = targetChannel.ID
	}

	seconds := int(optionInt(opts, "seconds", 0))
	if seconds < 0 || seconds > 21600 {
		respondText(s, i, "Slowmode delay must be between 0 and 21600 seconds (6 hours).")
		return
	}

	// Update channel slowmode
	edit := &discordgo.ChannelEdit{
		RateLimitPerUser: &seconds,
	}
	_, err := s.ChannelEditComplex(channelID, edit)
	if err != nil {
		embed := createErrorEmbed("Slowmode Failed", fmt.Sprintf("Failed to set slowmode: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Log action
	channel, _ := s.Channel(channelID)
	channelName := channelID
	if channel != nil {
		channelName = channel.Name
	}

	action := modAction{
		ID:          generateID("slowmode"),
		Type:        "slowmode",
		UserID:      channelID,
		Username:    channelName,
		ModeratorID: user.ID,
		Moderator:   user.Username,
		Reason:      fmt.Sprintf("Slowmode set to %d seconds", seconds),
		Duration:    int64(seconds),
		Timestamp:   time.Now().Unix(),
		GuildID:     i.GuildID,
	}
	logModAction(s, i.GuildID, action)

	// Success response
	msg := fmt.Sprintf("Slowmode disabled for <#%s>.", channelID)
	if seconds > 0 {
		msg = fmt.Sprintf("Slowmode set to **%d seconds** for <#%s>.", seconds, channelID)
	}
	embed := createSuccessEmbed("Slowmode Updated", msg)
	respondEmbed(s, i, embed)
}

// ============================================================================
// MODLOG COMMAND
// ============================================================================

func handleModlog(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check - requires administrator
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionAdministrator) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Administrator` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	sub, subOpts := getSubcommand(opts)

	switch sub {
	case "setup":
		handleModlogSetup(s, i, subOpts)
	case "view":
		handleModlogView(s, i, subOpts)
	default:
		respondText(s, i, "Invalid subcommand.")
	}
}

func handleModlogSetup(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	channel := optionChannel(opts, "channel")
	if channel == nil {
		respondText(s, i, "You must specify a channel.")
		return
	}

	// Load guild configs
	configs := map[string]guildConfig{}
	_ = readData("guild-config.json", &configs)

	// Update config
	config := configs[i.GuildID]
	config.GuildID = i.GuildID
	config.ModLogChannel = channel.ID
	configs[i.GuildID] = config

	// Save config
	err := writeData("guild-config.json", configs)
	if err != nil {
		embed := createErrorEmbed("Setup Failed", fmt.Sprintf("Failed to save configuration: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Success response
	embed := createSuccessEmbed("Mod Log Configured", fmt.Sprintf("Moderation logs will now be sent to <#%s>.", channel.ID))
	respondEmbed(s, i, embed)
}

func handleModlogView(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	targetUser := optionUser(opts, "user")
	actionType := optionString(opts, "action", "")

	// Load all actions
	actions := []modAction{}
	_ = readData("mod-actions.json", &actions)

	// Filter by guild
	var guildActions []modAction
	for _, action := range actions {
		if action.GuildID == i.GuildID {
			guildActions = append(guildActions, action)
		}
	}

	// Filter by user if specified
	if targetUser != nil {
		var filtered []modAction
		for _, action := range guildActions {
			if action.UserID == targetUser.ID {
				filtered = append(filtered, action)
			}
		}
		guildActions = filtered
	}

	// Filter by action type if specified
	if actionType != "" {
		var filtered []modAction
		for _, action := range guildActions {
			if action.Type == actionType {
				filtered = append(filtered, action)
			}
		}
		guildActions = filtered
	}

	if len(guildActions) == 0 {
		embed := createInfoEmbed("No Actions Found", "No moderation actions match the specified filters.")
		respondEmbed(s, i, embed)
		return
	}

	// Show last 10 actions
	start := 0
	if len(guildActions) > 10 {
		start = len(guildActions) - 10
	}

	var description string
	for idx := start; idx < len(guildActions); idx++ {
		action := guildActions[idx]
		timestamp := time.Unix(action.Timestamp, 0).Format("2006-01-02 15:04")

		actionEmoji := map[string]string{
			"kick":     "👢",
			"ban":      "🔨",
			"unban":    "🔓",
			"timeout":  "⏰",
			"warn":     "⚠️",
			"purge":    "🧹",
			"lock":     "🔒",
			"unlock":   "🔓",
			"slowmode": "🐌",
		}

		emoji := actionEmoji[action.Type]
		if emoji == "" {
			emoji = "📝"
		}

		description += fmt.Sprintf("%s **%s** - <@%s>\n", emoji, strings.ToUpper(action.Type), action.UserID)
		description += fmt.Sprintf("Moderator: <@%s> | %s\n", action.ModeratorID, timestamp)
		description += fmt.Sprintf("Reason: %s\n\n", action.Reason)
	}

	totalCount := len(guildActions)
	title := "📋 Moderation History"
	if targetUser != nil {
		title = fmt.Sprintf("📋 Moderation History - %s", targetUser.Username)
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       ColorBlue,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Showing last %d of %d actions", len(guildActions)-start, totalCount)},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	if targetUser != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: userAvatar(targetUser)}
	}

	respondEmbed(s, i, embed)
}

// ============================================================================
// AUTOMOD COMMAND
// ============================================================================

func handleAutomod(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user.")
		return
	}

	// Permission check - requires administrator
	if !hasPermission(s, i.GuildID, user.ID, discordgo.PermissionAdministrator) {
		embed := createErrorEmbed("Missing Permissions", "You need the `Administrator` permission to use this command.")
		respondEmbed(s, i, embed)
		return
	}

	sub, subOpts := getSubcommand(opts)

	switch sub {
	case "setup":
		handleAutomodSetup(s, i)
	case "toggle":
		handleAutomodToggle(s, i, subOpts)
	default:
		respondText(s, i, "Invalid subcommand.")
	}
}

func handleAutomodSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Load guild configs
	configs := map[string]guildConfig{}
	_ = readData("guild-config.json", &configs)

	// Get or create config
	config := configs[i.GuildID]
	config.GuildID = i.GuildID

	// Initialize automod with default settings if not already set
	if !config.AutoMod.Enabled {
		config.AutoMod = autoModConfig{
			Enabled:          true,
			SpamEnabled:      true,
			SpamMessageLimit: 5,
			SpamInterval:     3,
			LinksEnabled:     false,
			InvitesEnabled:   true,
			AllCapsEnabled:   true,
			AllCapsPercent:   70,
			ForbiddenWords:   []string{},
			EmojiSpamEnabled: true,
			EmojiSpamLimit:   10,
		}
	} else {
		config.AutoMod.Enabled = true
	}

	configs[i.GuildID] = config

	// Save config
	err := writeData("guild-config.json", configs)
	if err != nil {
		embed := createErrorEmbed("Setup Failed", fmt.Sprintf("Failed to save configuration: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Build status embed
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Auto-Moderation Configured",
		Description: "Auto-moderation has been enabled with the following settings:",
		Color:       ColorGreen,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Spam Detection", Value: formatEnabled(config.AutoMod.SpamEnabled), Inline: true},
			{Name: "Invite Links", Value: formatEnabled(config.AutoMod.InvitesEnabled), Inline: true},
			{Name: "All Caps", Value: formatEnabled(config.AutoMod.AllCapsEnabled), Inline: true},
			{Name: "Emoji Spam", Value: formatEnabled(config.AutoMod.EmojiSpamEnabled), Inline: true},
			{Name: "External Links", Value: formatEnabled(config.AutoMod.LinksEnabled), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "Use /automod toggle to enable/disable specific rules"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	respondEmbed(s, i, embed)
}

func handleAutomodToggle(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	rule := optionString(opts, "rule", "")
	enabled := optionBool(opts, "enabled", false)

	if rule == "" {
		respondText(s, i, "You must specify a rule to toggle.")
		return
	}

	// Load guild configs
	configs := map[string]guildConfig{}
	_ = readData("guild-config.json", &configs)

	// Get config
	config := configs[i.GuildID]
	config.GuildID = i.GuildID

	// Toggle the specified rule
	switch rule {
	case "spam":
		config.AutoMod.SpamEnabled = enabled
	case "invites":
		config.AutoMod.InvitesEnabled = enabled
	case "caps":
		config.AutoMod.AllCapsEnabled = enabled
	case "emoji":
		config.AutoMod.EmojiSpamEnabled = enabled
	case "links":
		config.AutoMod.LinksEnabled = enabled
	default:
		respondText(s, i, "Invalid rule specified.")
		return
	}

	configs[i.GuildID] = config

	// Save config
	err := writeData("guild-config.json", configs)
	if err != nil {
		embed := createErrorEmbed("Toggle Failed", fmt.Sprintf("Failed to save configuration: %v", err))
		respondEmbed(s, i, embed)
		return
	}

	// Success response
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	embed := createSuccessEmbed("Rule Updated", fmt.Sprintf("Auto-mod rule **%s** has been **%s**.", rule, status))
	respondEmbed(s, i, embed)
}

func formatEnabled(enabled bool) string {
	if enabled {
		return "✅ Enabled"
	}
	return "❌ Disabled"
}

// ============================================================================
// AUTO-MODERATION MESSAGE CHECKING
// ============================================================================

func CheckAutoMod(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bots
	if m.Author.Bot {
		return
	}

	// Ignore DMs
	if m.GuildID == "" {
		return
	}

	// Load guild config
	configs := map[string]guildConfig{}
	_ = readData("guild-config.json", &configs)

	config, exists := configs[m.GuildID]
	if !exists || !config.AutoMod.Enabled {
		return
	}

	// Check spam
	if config.AutoMod.SpamEnabled {
		if isSpamming(m.GuildID, m.Author.ID, config.AutoMod.SpamMessageLimit, config.AutoMod.SpamInterval) {
			_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
			sendAutomodWarning(s, m, "Spam detected")
			return
		}
	}

	// Check forbidden words
	if len(config.AutoMod.ForbiddenWords) > 0 {
		lower := strings.ToLower(m.Content)
		for _, word := range config.AutoMod.ForbiddenWords {
			if strings.Contains(lower, strings.ToLower(word)) {
				_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
				sendAutomodWarning(s, m, "Used forbidden word")
				return
			}
		}
	}

	// Check all caps
	if config.AutoMod.AllCapsEnabled && len(m.Content) > 10 {
		capsCount := 0
		totalLetters := 0
		for _, r := range m.Content {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				totalLetters++
				if r >= 'A' && r <= 'Z' {
					capsCount++
				}
			}
		}
		if totalLetters > 0 {
			capsPercent := (capsCount * 100) / totalLetters
			if capsPercent >= config.AutoMod.AllCapsPercent {
				_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
				sendAutomodWarning(s, m, "Excessive caps")
				return
			}
		}
	}

	// Check emoji spam
	if config.AutoMod.EmojiSpamEnabled {
		emojiCount := countEmojis(m.Content)
		if emojiCount > config.AutoMod.EmojiSpamLimit {
			_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
			sendAutomodWarning(s, m, "Emoji spam")
			return
		}
	}

	// Check invite links
	if config.AutoMod.InvitesEnabled {
		if strings.Contains(m.Content, "discord.gg/") || strings.Contains(m.Content, "discord.com/invite/") {
			_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
			sendAutomodWarning(s, m, "Posted invite link")
			return
		}
	}

	// Check external links
	if config.AutoMod.LinksEnabled {
		if strings.Contains(m.Content, "http://") || strings.Contains(m.Content, "https://") {
			_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
			sendAutomodWarning(s, m, "Posted external link")
			return
		}
	}
}

func isSpamming(guildID, userID string, limit, interval int) bool {
	key := fmt.Sprintf("%s_%s", guildID, userID)
	now := time.Now().Unix()

	modMu.Lock()
	defer modMu.Unlock()

	tracker := spamTrackers[key]
	if tracker == nil {
		tracker = &spamTracker{
			MessageTimes: []int64{},
		}
		spamTrackers[key] = tracker
	}

	// Remove old messages outside the interval
	var recentMessages []int64
	for _, msgTime := range tracker.MessageTimes {
		if now-msgTime <= int64(interval) {
			recentMessages = append(recentMessages, msgTime)
		}
	}

	// Add current message
	recentMessages = append(recentMessages, now)
	tracker.MessageTimes = recentMessages

	// Check if spam threshold exceeded
	return len(recentMessages) > limit
}

func countEmojis(text string) int {
	count := 0
	// Count Unicode emojis (rough estimation)
	for _, r := range text {
		// Emoji ranges (simplified)
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) { // Dingbats
			count++
		}
	}
	// Count Discord custom emojis <:name:id>
	count += strings.Count(text, "<:") + strings.Count(text, "<a:")
	return count
}

func sendAutomodWarning(s *discordgo.Session, m *discordgo.MessageCreate, reason string) {
	// Send ephemeral warning (via DM)
	ch, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚠️ Auto-Moderation",
		Description: fmt.Sprintf("Your message was removed in the server.\n**Reason:** %s", reason),
		Color:       ColorYellow,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(ch.ID, embed)
}

// ============================================================================
// COMMAND HANDLERS (for routing from main.go)
// ============================================================================

func handleCoreModeration(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	switch i.ApplicationCommandData().Name {
	case "kick":
		handleKick(s, i, opts)
	case "ban":
		handleBan(s, i, opts)
	case "unban":
		handleUnban(s, i, opts)
	case "timeout":
		handleTimeout(s, i, opts)
	case "warn":
		handleWarn(s, i, opts)
	case "warnings":
		handleWarnings(s, i, opts)
	case "clearwarnings":
		handleClearWarnings(s, i, opts)
	}
}

func handleMessageModeration(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	switch i.ApplicationCommandData().Name {
	case "purge":
		handlePurge(s, i, opts)
	case "lock":
		handleLock(s, i, opts)
	case "unlock":
		handleUnlock(s, i, opts)
	case "slowmode":
		handleSlowmode(s, i, opts)
	}
}

func handleModConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	switch i.ApplicationCommandData().Name {
	case "modlog":
		handleModlog(s, i, opts)
	case "automod":
		handleAutomod(s, i, opts)
	}
}
