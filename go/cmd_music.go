package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ── Music session types ─────────────────────────────────────────────────

type musicTrack struct {
	Title    string
	Query    string
	Duration string
	AddedBy  string
}

type musicSession struct {
	GuildID   string
	ChannelID string
	VoiceConn *discordgo.VoiceConnection
	Queue     []musicTrack
	Current   int
	Playing   bool
	Paused    bool
	Volume    int
	LoopMode  string // "off", "song", "queue"
	StartedAt time.Time
}

var (
	musicSessions = map[string]*musicSession{}
	musicMu       sync.Mutex
)

func getOrCreateMusicSession(guildID string) *musicSession {
	if ms, ok := musicSessions[guildID]; ok {
		return ms
	}
	ms := &musicSession{
		GuildID:  guildID,
		Volume:   50,
		LoopMode: "off",
	}
	musicSessions[guildID] = ms
	return ms
}

// findUserVoiceChannel returns the voice channel ID the user is in, or "".
func findUserVoiceChannel(s *discordgo.Session, guildID, userID string) string {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		// Fallback: fetch from API
		guild, err = s.Guild(guildID)
		if err != nil {
			return ""
		}
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}
	return ""
}

// ── Command handler ─────────────────────────────────────────────────────

func handleMusicCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		respondEmbed(s, i, createErrorEmbed("Music", "No subcommand provided."))
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "play":
		handleMusicPlay(s, i, sub.Options)
	case "pause":
		handleMusicPause(s, i)
	case "resume":
		handleMusicResume(s, i)
	case "skip":
		handleMusicSkip(s, i)
	case "stop":
		handleMusicStop(s, i)
	case "queue":
		handleMusicQueue(s, i)
	case "nowplaying":
		handleMusicNowPlaying(s, i)
	case "volume":
		handleMusicVolume(s, i, sub.Options)
	case "shuffle":
		handleMusicShuffle(s, i)
	case "loop":
		handleMusicLoop(s, i, sub.Options)
	default:
		respondEmbed(s, i, createErrorEmbed("Music", "Unknown subcommand."))
	}
}

// ── /music play ─────────────────────────────────────────────────────────

func handleMusicPlay(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	query := optionString(opts, "query", "")
	if query == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "Please provide a song name or URL."))
		return
	}

	user := interactionUser(i)
	if user == nil {
		respondEmbed(s, i, createErrorEmbed("Music", "Could not identify user."))
		return
	}

	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	voiceChannelID := findUserVoiceChannel(s, guildID, user.ID)
	if voiceChannelID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "You must be in a voice channel to use this command."))
		return
	}

	musicMu.Lock()
	ms := getOrCreateMusicSession(guildID)

	// Join voice channel if not already connected
	if ms.VoiceConn == nil {
		vc, err := s.ChannelVoiceJoin(guildID, voiceChannelID, false, true)
		if err != nil {
			musicMu.Unlock()
			respondEmbed(s, i, createErrorEmbed("Music", fmt.Sprintf("Failed to join voice channel: %v", err)))
			return
		}
		ms.VoiceConn = vc
		ms.ChannelID = voiceChannelID
	}

	track := musicTrack{
		Title:    query, // TODO: Resolve actual title from YouTube/Spotify API
		Query:    query,
		Duration: "3:30", // TODO: Fetch real duration
		AddedBy:  user.Username,
	}
	ms.Queue = append(ms.Queue, track)

	wasPlaying := ms.Playing
	if !wasPlaying {
		ms.Current = len(ms.Queue) - 1
		ms.Playing = true
		ms.Paused = false
		ms.StartedAt = time.Now()
	}
	musicMu.Unlock()

	// TODO: Start actual audio streaming (ffmpeg/opus/DCA pipeline)

	if wasPlaying {
		embed := createMusicEmbed("🎵 Added to Queue", fmt.Sprintf(
			"**%s**\nDuration: `%s` • Requested by %s\nPosition in queue: #%d",
			track.Title, track.Duration, track.AddedBy, len(ms.Queue),
		))
		respondEmbed(s, i, embed)
	} else {
		embed := createMusicEmbed("🎶 Now Playing", fmt.Sprintf(
			"**%s**\nDuration: `%s` • Requested by %s\n🔊 Volume: %d%%",
			track.Title, track.Duration, track.AddedBy, ms.Volume,
		))
		respondEmbed(s, i, embed)
	}
}

// ── /music pause ────────────────────────────────────────────────────────

func handleMusicPause(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || !ms.Playing {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "Nothing is currently playing."))
		return
	}

	if ms.Paused {
		musicMu.Unlock()
		respondEmbed(s, i, createMusicEmbed("⏸️ Already Paused", "The player is already paused. Use `/music resume` to continue."))
		return
	}

	ms.Paused = true
	musicMu.Unlock()

	// TODO: Actually pause the audio stream

	respondEmbed(s, i, createMusicEmbed("⏸️ Paused", "Music has been paused. Use `/music resume` to continue."))
}

// ── /music resume ───────────────────────────────────────────────────────

func handleMusicResume(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || !ms.Playing {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "Nothing is currently playing."))
		return
	}

	if !ms.Paused {
		musicMu.Unlock()
		respondEmbed(s, i, createMusicEmbed("▶️ Already Playing", "The player is not paused."))
		return
	}

	ms.Paused = false
	musicMu.Unlock()

	// TODO: Actually resume the audio stream

	current := ms.Queue[ms.Current]
	respondEmbed(s, i, createMusicEmbed("▶️ Resumed", fmt.Sprintf("Now playing: **%s**", current.Title)))
}

// ── /music skip ─────────────────────────────────────────────────────────

func handleMusicSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || !ms.Playing || len(ms.Queue) == 0 {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "Nothing is currently playing."))
		return
	}

	skippedTitle := ms.Queue[ms.Current].Title
	nextIndex := ms.Current + 1

	switch ms.LoopMode {
	case "song":
		// Stay on the same track
		nextIndex = ms.Current
	case "queue":
		if nextIndex >= len(ms.Queue) {
			nextIndex = 0
		}
	default: // "off"
		// nextIndex already set
	}

	if nextIndex >= len(ms.Queue) {
		// No more songs
		ms.Playing = false
		ms.Paused = false
		ms.Current = 0
		musicMu.Unlock()

		// TODO: Stop audio stream

		respondEmbed(s, i, createMusicEmbed("⏭️ Skipped", fmt.Sprintf("Skipped **%s**. Queue is now empty.", skippedTitle)))
		return
	}

	ms.Current = nextIndex
	ms.Paused = false
	ms.StartedAt = time.Now()
	nextTrack := ms.Queue[ms.Current]
	musicMu.Unlock()

	// TODO: Start playing next track

	respondEmbed(s, i, createMusicEmbed("⏭️ Skipped", fmt.Sprintf(
		"Skipped **%s**\nNow playing: **%s**",
		skippedTitle, nextTrack.Title,
	)))
}

// ── /music stop ─────────────────────────────────────────────────────────

func handleMusicStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "No active music session."))
		return
	}

	// Disconnect from voice
	if ms.VoiceConn != nil {
		_ = ms.VoiceConn.Disconnect()
		ms.VoiceConn = nil
	}

	// Clear session
	delete(musicSessions, guildID)
	musicMu.Unlock()

	// TODO: Stop audio stream

	respondEmbed(s, i, createMusicEmbed("⏹️ Stopped", "Cleared the queue and disconnected from voice."))
}

// ── /music queue ────────────────────────────────────────────────────────

func handleMusicQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || len(ms.Queue) == 0 {
		musicMu.Unlock()
		respondEmbed(s, i, createMusicEmbed("📜 Queue", "The queue is empty. Use `/music play` to add songs!"))
		return
	}

	var lines []string

	// Show current track
	if ms.Current < len(ms.Queue) {
		cur := ms.Queue[ms.Current]
		status := "▶️"
		if ms.Paused {
			status = "⏸️"
		}
		lines = append(lines, fmt.Sprintf("%s **Now Playing:** %s [`%s`] — %s\n", status, cur.Title, cur.Duration, cur.AddedBy))
	}

	// Show upcoming tracks (up to 10)
	shown := 0
	for idx := ms.Current + 1; idx < len(ms.Queue) && shown < 10; idx++ {
		t := ms.Queue[idx]
		lines = append(lines, fmt.Sprintf("`%d.` **%s** [`%s`] — %s", idx-ms.Current, t.Title, t.Duration, t.AddedBy))
		shown++
	}

	remaining := len(ms.Queue) - ms.Current - 1 - shown
	if remaining > 0 {
		lines = append(lines, fmt.Sprintf("\n*...and %d more track(s)*", remaining))
	}

	loopIcon := ""
	switch ms.LoopMode {
	case "song":
		loopIcon = " | 🔂 Loop: Song"
	case "queue":
		loopIcon = " | 🔁 Loop: Queue"
	}

	lines = append(lines, fmt.Sprintf("\n**%d track(s) in queue** | 🔊 Volume: %d%%%s", len(ms.Queue)-ms.Current, ms.Volume, loopIcon))
	musicMu.Unlock()

	respondEmbed(s, i, createMusicEmbed("📜 Queue", strings.Join(lines, "\n")))
}

// ── /music nowplaying ───────────────────────────────────────────────────

func handleMusicNowPlaying(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || !ms.Playing || ms.Current >= len(ms.Queue) {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "Nothing is currently playing."))
		return
	}

	track := ms.Queue[ms.Current]
	elapsed := int(time.Since(ms.StartedAt).Seconds())
	// TODO: Parse actual duration; for now use 210s (3:30)
	totalSeconds := 210
	if elapsed > totalSeconds {
		elapsed = totalSeconds
	}

	progressBar := renderProgressBar(elapsed, totalSeconds, 20)
	elapsedStr := fmt.Sprintf("%d:%02d", elapsed/60, elapsed%60)
	totalStr := track.Duration

	status := "▶️ Playing"
	if ms.Paused {
		status = "⏸️ Paused"
	}

	loopStr := ""
	switch ms.LoopMode {
	case "song":
		loopStr = "🔂 Song"
	case "queue":
		loopStr = "🔁 Queue"
	default:
		loopStr = "Off"
	}
	musicMu.Unlock()

	desc := fmt.Sprintf(
		"**%s**\nRequested by %s\n\n`%s` %s `%s`\n\n%s | 🔊 %d%% | Loop: %s",
		track.Title, track.AddedBy,
		elapsedStr, progressBar, totalStr,
		status, ms.Volume, loopStr,
	)
	respondEmbed(s, i, createMusicEmbed("🎶 Now Playing", desc))
}

// ── /music volume ───────────────────────────────────────────────────────

func handleMusicVolume(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	level := int(optionInt(opts, "level", -1))
	if level < 1 || level > 100 {
		respondEmbed(s, i, createErrorEmbed("Music", "Volume must be between 1 and 100."))
		return
	}

	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms := getOrCreateMusicSession(guildID)
	old := ms.Volume
	ms.Volume = level
	musicMu.Unlock()

	// TODO: Apply volume to audio stream

	icon := "🔊"
	if level <= 30 {
		icon = "🔉"
	} else if level <= 10 {
		icon = "🔈"
	}

	respondEmbed(s, i, createMusicEmbed(fmt.Sprintf("%s Volume", icon), fmt.Sprintf("Volume changed: **%d%%** → **%d%%**", old, level)))
}

// ── /music shuffle ──────────────────────────────────────────────────────

func handleMusicShuffle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms, ok := musicSessions[guildID]
	if !ok || len(ms.Queue) <= 1 {
		musicMu.Unlock()
		respondEmbed(s, i, createErrorEmbed("Music", "Not enough songs in the queue to shuffle."))
		return
	}

	// Only shuffle songs after the current one
	upcoming := ms.Queue[ms.Current+1:]
	rand.Shuffle(len(upcoming), func(i, j int) {
		upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
	})
	musicMu.Unlock()

	respondEmbed(s, i, createMusicEmbed("🔀 Shuffled", fmt.Sprintf("Shuffled **%d** upcoming track(s) in the queue.", len(upcoming))))
}

// ── /music loop ─────────────────────────────────────────────────────────

func handleMusicLoop(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	mode := optionString(opts, "mode", "off")
	if mode != "off" && mode != "song" && mode != "queue" {
		respondEmbed(s, i, createErrorEmbed("Music", "Loop mode must be `off`, `song`, or `queue`."))
		return
	}

	guildID := i.GuildID
	if guildID == "" {
		respondEmbed(s, i, createErrorEmbed("Music", "This command can only be used in a server."))
		return
	}

	musicMu.Lock()
	ms := getOrCreateMusicSession(guildID)
	ms.LoopMode = mode
	musicMu.Unlock()

	icons := map[string]string{"off": "➡️", "song": "🔂", "queue": "🔁"}
	labels := map[string]string{"off": "Off", "song": "Current Song", "queue": "Entire Queue"}

	respondEmbed(s, i, createMusicEmbed(
		fmt.Sprintf("%s Loop Mode", icons[mode]),
		fmt.Sprintf("Loop mode set to: **%s**", labels[mode]),
	))
}
