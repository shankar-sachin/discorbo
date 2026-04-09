package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ── Music session types ─────────────────────────────────────────────────

type musicTrack struct {
	Title    string
	URL      string // Direct audio URL from yt-dlp
	Query    string
	Duration string
	AddedBy  string
}

type musicSession struct {
	GuildID    string
	ChannelID  string
	TextChanID string
	VoiceConn  *discordgo.VoiceConnection
	Queue      []musicTrack
	Current    int
	Playing    bool
	Paused     bool
	Volume     int
	LoopMode   string // "off", "song", "queue"
	StartedAt  time.Time
	stopCh     chan struct{}
	ffmpeg     *exec.Cmd
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
		stopCh:   make(chan struct{}),
	}
	musicSessions[guildID] = ms
	return ms
}

// findUserVoiceChannel returns the voice channel ID the user is in, or "".
func findUserVoiceChannel(s *discordgo.Session, guildID, userID string) string {
	guild, err := s.State.Guild(guildID)
	if err != nil {
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

// ── Audio pipeline ──────────────────────────────────────────────────────

// resolveTrack uses yt-dlp to get the audio URL and title for a query.
func resolveTrack(query string) (title, audioURL, duration string, err error) {
	args := []string{
		"--no-playlist", "--default-search", "ytsearch",
		"--print", "title", "--print", "url", "--print", "duration_string",
		"-f", "bestaudio[ext=webm]/bestaudio/best",
		"--no-warnings", "--no-check-certificates",
		query,
	}
	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("yt-dlp failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", "", "", fmt.Errorf("yt-dlp returned unexpected output")
	}
	title = strings.TrimSpace(lines[0])
	audioURL = strings.TrimSpace(lines[1])
	duration = "?:??"
	if len(lines) >= 3 {
		duration = strings.TrimSpace(lines[2])
	}
	return title, audioURL, duration, nil
}

// streamTrack streams a single track through the voice connection.
func streamTrack(ms *musicSession) {
	vc := ms.VoiceConn
	if vc == nil || ms.Current >= len(ms.Queue) {
		return
	}
	track := ms.Queue[ms.Current]
	if track.URL == "" {
		fmt.Println("[MUSIC] Track has no URL, skipping")
		return
	}

	// Wait for voice connection to be fully ready
	for i := 0; i < 50; i++ {
		if vc.Ready {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !vc.Ready {
		fmt.Println("[MUSIC] Voice connection not ready after 10s, aborting")
		return
	}

	fmt.Printf("[MUSIC] Starting ffmpeg for: %s\n", track.Title)

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-reconnect", "1", "-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-i", track.URL,
		"-c:a", "libopus",
		"-ar", "48000", "-ac", "2",
		"-b:a", "96k",
		"-application", "audio",
		"-frame_duration", "20",
		"-vbr", "off",
		"-f", "ogg",
		"pipe:1",
	}
	cmd := exec.Command("ffmpeg", args...)

	// Capture stderr for diagnostics
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("[MUSIC] Failed to create stdout pipe: %v\n", err)
		return
	}

	ms.ffmpeg = cmd
	if err := cmd.Start(); err != nil {
		fmt.Printf("[MUSIC] Failed to start ffmpeg: %v\n", err)
		ms.ffmpeg = nil
		return
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
		ms.ffmpeg = nil
		if errStr := stderrBuf.String(); errStr != "" {
			fmt.Printf("[MUSIC] ffmpeg stderr: %s\n", errStr)
		}
	}()

	// Set speaking BEFORE sending any frames
	if vc.Ready {
		if err := vc.Speaking(true); err != nil {
			fmt.Printf("[MUSIC] Speaking(true) failed: %v\n", err)
		}
	}

	reader := bufio.NewReaderSize(stdout, 65536)
	pageNum := 0
	framesSent := 0

	for {
		// Check stop signal
		select {
		case <-ms.stopCh:
			fmt.Printf("[MUSIC] Stop signal received after %d frames\n", framesSent)
			return
		default:
		}

		// Pause loop
		for ms.Paused {
			if vc.Ready {
				vc.Speaking(false)
			}
			time.Sleep(250 * time.Millisecond)
			select {
			case <-ms.stopCh:
				return
			default:
			}
		}
		// Resume speaking after unpause
		if framesSent > 0 && ms.Paused == false && vc.Ready {
			vc.Speaking(true)
		}

		// Bail if voice connection is gone
		if ms.VoiceConn == nil || !vc.Ready {
			fmt.Printf("[MUSIC] Voice connection lost after %d frames\n", framesSent)
			return
		}

		// Read one OGG page
		pageData, segTable, err := readRawOggPage(reader)
		if err != nil {
			if framesSent > 0 {
				fmt.Printf("[MUSIC] Track finished (%d frames sent)\n", framesSent)
			} else {
				fmt.Printf("[MUSIC] OGG read error (0 frames sent): %v\n", err)
			}
			return
		}
		pageNum++

		// Skip first 2 pages unconditionally (OpusHead + OpusTags headers)
		if pageNum <= 2 {
			if pageNum == 1 {
				fmt.Printf("[MUSIC] Skipped OpusHead page (%d bytes)\n", len(pageData))
			} else {
				fmt.Printf("[MUSIC] Skipped OpusTags page (%d bytes)\n", len(pageData))
			}
			continue
		}

		// Extract opus packets from segment data
		packets := extractOggPackets(pageData, segTable)

		for _, pkt := range packets {
			// Validate: opus frames are typically 3-1275 bytes
			if len(pkt) < 1 || len(pkt) > 4000 {
				continue
			}

			if ms.VoiceConn == nil || !vc.Ready || vc.OpusSend == nil {
				fmt.Printf("[MUSIC] Connection lost mid-send after %d frames\n", framesSent)
				return
			}

			select {
			case vc.OpusSend <- pkt:
				framesSent++
				if framesSent == 1 {
					fmt.Printf("[MUSIC] First opus frame sent (%d bytes)\n", len(pkt))
				}
			case <-ms.stopCh:
				return
			case <-time.After(5 * time.Second):
				fmt.Println("[MUSIC] OpusSend blocked for 5s, aborting")
				return
			}
		}
	}
}

// readRawOggPage reads one OGG page and returns the raw segment data and table.
func readRawOggPage(r io.Reader) (data []byte, segTable []byte, err error) {
	// Read "OggS" magic
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, nil, err
	}
	if string(magic) != "OggS" {
		// Try to resync
		if err := resyncOgg(r, magic); err != nil {
			return nil, nil, err
		}
	}

	// Read 23-byte fixed header after "OggS"
	hdr := make([]byte, 23)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, nil, err
	}

	numSegments := int(hdr[22])

	segTable = make([]byte, numSegments)
	if _, err := io.ReadFull(r, segTable); err != nil {
		return nil, nil, err
	}

	totalSize := 0
	for _, s := range segTable {
		totalSize += int(s)
	}
	data = make([]byte, totalSize)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, nil, err
	}

	return data, segTable, nil
}

// resyncOgg scans the stream until it finds the next "OggS" magic.
// initial contains the 4 bytes that were NOT "OggS" so we can check them.
func resyncOgg(r io.Reader, initial []byte) error {
	buf := make([]byte, 1)
	// Seed the window with the bytes we already read (skip first since it wasn't 'O')
	window := [4]byte{initial[0], initial[1], initial[2], initial[3]}
	for i := 0; i < 65536; i++ {
		if window[0] == 'O' && window[1] == 'g' && window[2] == 'g' && window[3] == 'S' {
			return nil
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		window[0], window[1], window[2], window[3] = window[1], window[2], window[3], buf[0]
	}
	return fmt.Errorf("OGG resync failed: no OggS in 64KB")
}

// extractOggPackets splits OGG page data into individual packets.
func extractOggPackets(data []byte, segTable []byte) [][]byte {
	var packets [][]byte
	var pkt []byte
	offset := 0
	for _, segLen := range segTable {
		sl := int(segLen)
		if offset+sl > len(data) {
			break
		}
		pkt = append(pkt, data[offset:offset+sl]...)
		offset += sl
		if segLen < 255 {
			if len(pkt) > 0 {
				out := make([]byte, len(pkt))
				copy(out, pkt)
				packets = append(packets, out)
			}
			pkt = nil
		}
	}
	if len(pkt) > 0 {
		out := make([]byte, len(pkt))
		copy(out, pkt)
		packets = append(packets, out)
	}
	return packets
}

// advanceTrack moves to the next track based on loop mode.
func advanceTrack(ms *musicSession) bool {
	musicMu.Lock()
	defer musicMu.Unlock()

	nextIndex := ms.Current + 1
	switch ms.LoopMode {
	case "song":
		nextIndex = ms.Current
	case "queue":
		if nextIndex >= len(ms.Queue) {
			nextIndex = 0
		}
	default:
		if nextIndex >= len(ms.Queue) {
			ms.Playing = false
			ms.Paused = false
			ms.Current = 0
			return false
		}
	}
	ms.Current = nextIndex
	ms.Paused = false
	ms.StartedAt = time.Now()
	return true
}

// startPlayback is the goroutine that plays through the queue.
func startPlayback(s *discordgo.Session, ms *musicSession) {
	for {
		select {
		case <-ms.stopCh:
			return
		default:
		}

		musicMu.Lock()
		if !ms.Playing || ms.Current >= len(ms.Queue) {
			musicMu.Unlock()
			return
		}
		musicMu.Unlock()

		streamTrack(ms)

		if !advanceTrack(ms) {
			// Only call Speaking(false) if the connection is still alive
			if ms.VoiceConn != nil && ms.VoiceConn.Ready {
				ms.VoiceConn.Speaking(false)
			}
			if ms.TextChanID != "" {
				embed := createMusicEmbed("📭 Queue Ended", "All tracks have been played!")
				s.ChannelMessageSendEmbed(ms.TextChanID, embed)
			}
			return
		}

		musicMu.Lock()
		if ms.Current < len(ms.Queue) && ms.TextChanID != "" {
			next := ms.Queue[ms.Current]
			embed := createMusicEmbed("🎶 Now Playing", fmt.Sprintf(
				"**%s**\nDuration: `%s` • Requested by %s",
				next.Title, next.Duration, next.AddedBy,
			))
			s.ChannelMessageSendEmbed(ms.TextChanID, embed)
		}
		musicMu.Unlock()
	}
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

	// Defer reply since yt-dlp resolution can take a few seconds
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Resolve track info via yt-dlp
	title, audioURL, duration, err := resolveTrack(query)
	if err != nil {
		editDeferredEmbed(s, i, createErrorEmbed("Music", fmt.Sprintf("Could not find track: %v\n\nMake sure `yt-dlp` is installed and in your PATH.", err)), nil)
		return
	}

	musicMu.Lock()
	ms := getOrCreateMusicSession(guildID)
	ms.TextChanID = i.ChannelID

	// Join voice channel if not already connected
	if ms.VoiceConn == nil {
		vc, err := s.ChannelVoiceJoin(guildID, voiceChannelID, false, false)
		if err != nil {
			musicMu.Unlock()
			editDeferredEmbed(s, i, createErrorEmbed("Music", fmt.Sprintf("Failed to join voice channel: %v", err)), nil)
			return
		}
		ms.VoiceConn = vc
		ms.ChannelID = voiceChannelID

		// Wait for voice connection to be ready before proceeding
		musicMu.Unlock()
		ready := false
		for attempts := 0; attempts < 50; attempts++ {
			if vc.Ready {
				ready = true
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !ready {
			vc.Disconnect()
			musicMu.Lock()
			ms.VoiceConn = nil
			musicMu.Unlock()
			editDeferredEmbed(s, i, createErrorEmbed("Music", "Voice connection timed out. Please try again."), nil)
			return
		}
		musicMu.Lock()
	}

	track := musicTrack{
		Title:   title,
		URL:     audioURL,
		Query:   query,
		Duration: duration,
		AddedBy: user.Username,
	}
	ms.Queue = append(ms.Queue, track)

	wasPlaying := ms.Playing
	if !wasPlaying {
		ms.Current = len(ms.Queue) - 1
		ms.Playing = true
		ms.Paused = false
		ms.StartedAt = time.Now()
		ms.stopCh = make(chan struct{})
	}
	musicMu.Unlock()

	if wasPlaying {
		embed := createMusicEmbed("🎵 Added to Queue", fmt.Sprintf(
			"**%s**\nDuration: `%s` • Requested by %s\nPosition in queue: #%d",
			track.Title, track.Duration, track.AddedBy, len(ms.Queue)-ms.Current,
		))
		editDeferredEmbed(s, i, embed, nil)
	} else {
		embed := createMusicEmbed("🎶 Now Playing", fmt.Sprintf(
			"**%s**\nDuration: `%s` • Requested by %s\n🔊 Volume: %d%%",
			track.Title, track.Duration, track.AddedBy, ms.Volume,
		))
		editDeferredEmbed(s, i, embed, nil)
		// Start the playback goroutine
		go startPlayback(s, ms)
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
	if ms.VoiceConn != nil {
		ms.VoiceConn.Speaking(false)
	}
	musicMu.Unlock()

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
	if ms.VoiceConn != nil {
		ms.VoiceConn.Speaking(true)
	}
	musicMu.Unlock()

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

	// Kill ffmpeg to stop current track — startPlayback will advance automatically
	if ms.ffmpeg != nil && ms.ffmpeg.Process != nil {
		ms.ffmpeg.Process.Kill()
	}
	musicMu.Unlock()

	respondEmbed(s, i, createMusicEmbed("⏭️ Skipped", fmt.Sprintf("Skipped **%s**", skippedTitle)))
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

	// Signal playback goroutine to stop
	select {
	case <-ms.stopCh:
	default:
		close(ms.stopCh)
	}

	// Kill ffmpeg first to unblock the reader
	if ms.ffmpeg != nil && ms.ffmpeg.Process != nil {
		ms.ffmpeg.Process.Kill()
	}

	// Give the playback goroutine a moment to exit cleanly
	vc := ms.VoiceConn
	ms.VoiceConn = nil // nil this so streamTrack sees it and bails
	ms.Playing = false
	musicMu.Unlock()

	// Brief wait for goroutine to notice and exit
	time.Sleep(300 * time.Millisecond)

	// Disconnect from voice
	if vc != nil {
		_ = vc.Disconnect()
	}

	// Clean up session
	musicMu.Lock()
	delete(musicSessions, guildID)
	musicMu.Unlock()

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
	// Parse duration string (e.g. "3:30" or "1:02:30")
	totalSeconds := parseDurationStr(track.Duration)
	if totalSeconds <= 0 {
		totalSeconds = 210 // fallback 3:30
	}
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

// parseDurationStr parses a duration string like "3:30" or "1:02:30" into seconds.
func parseDurationStr(d string) int {
	parts := strings.Split(d, ":")
	total := 0
	for _, p := range parts {
		val := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				val = val*10 + int(c-'0')
			}
		}
		total = total*60 + val
	}
	return total
}
