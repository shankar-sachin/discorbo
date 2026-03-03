package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var levels = []levelDef{
	{
		Name:       "Beginner's Path",
		Difficulty: "Easy",
		Maze: []string{
			"#######",
			"#P.C..#",
			"#.##.##",
			"#C...G#",
			"#######",
		},
		StartX:    1,
		StartY:    1,
		TimeLimit: 120,
	},
	{
		Name:       "Spike Corridor",
		Difficulty: "Medium",
		Maze: []string{
			"#########",
			"#P.C.S.C#",
			"#.##.##.#",
			"#C.S.C..#",
			"#.#.##.G#",
			"#########",
		},
		StartX:    1,
		StartY:    1,
		TimeLimit: 90,
	},
	{
		Name:       "Monster Maze",
		Difficulty: "Hard",
		Maze: []string{
			"###########",
			"#P.C.#C..C#",
			"#.#.##.##.#",
			"#C..E..S..#",
			"###C#.##.##",
			"#C....S..G#",
			"###########",
		},
		StartX:    1,
		StartY:    1,
		TimeLimit: 60,
	},
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Load repo-root .env first
	if err := godotenv.Load(filepath.Join("..", ".env")); err != nil {
		_ = godotenv.Load(".env")
	}

	token := os.Getenv("DISCORD_TOKEN")
	clientID := os.Getenv("CLIENT_ID")
	guildID := os.Getenv("GUILD_ID")
	if token == "" || clientID == "" {
		log.Fatal("DISCORD_TOKEN and CLIENT_ID must be set")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsDirectMessageTyping |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildVoiceStates

	s.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Go bot ready as %s", r.User.Username)
	})
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handleCommand(s, i)
		case discordgo.InteractionMessageComponent:
			handleComponent(s, i)
		case discordgo.InteractionModalSubmit:
			handleModalSubmit(s, i)
		}
	})
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handleMessageCreate(s, m)
	})
	s.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		handleGuildMemberAdd(s, m)
	})
	s.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
		handleGuildMemberRemove(s, m)
	})
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageDelete) {
		HandleMessageDelete(s, m)
	})

	// Connection retry: 3 attempts, 5s delay
	for attempt := 1; attempt <= 3; attempt++ {
		if err := s.Open(); err != nil {
			log.Printf("open session attempt %d/3: %v", attempt, err)
			if attempt == 3 {
				log.Fatalf("failed to open session after 3 attempts: %v", err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	defer s.Close()

	if err := registerCommands(s, clientID, guildID); err != nil {
		log.Fatalf("register commands: %v", err)
	}
	startReminderLoop(s)

	// Migrate v1 data if needed
	migrateV1Data()

	log.Println("Go runtime active (all commands)")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	gracefulShutdown()
}

func gracefulShutdown() {
	log.Println("Shutting down gracefully...")
	// Save any active game sessions
	sessionMu.Lock()
	if len(sessions) > 0 {
		log.Printf("Cleaning up %d active maze session(s)", len(sessions))
	}
	sessionMu.Unlock()
}

func registerCommands(s *discordgo.Session, appID, guildID string) error {
	cmds := allCommands()
	if guildID != "" {
		if _, err := s.ApplicationCommandBulkOverwrite(appID, guildID, cmds); err != nil {
			return err
		}
		// Clear global commands so users don't see duplicates when guild-scoped commands are active.
		_, err := s.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{})
		return err
	}
	_, err := s.ApplicationCommandBulkOverwrite(appID, "", cmds)
	return err
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	switch name {
	case "games":
		handleGamesCmd(s, i)
	case "casino":
		handleCasinoCmd(s, i)
	case "fun":
		handleFunCmd(s, i)
	case "math":
		handleMathCmd(s, i)
	case "util":
		handleUtilCmd(s, i)
	case "economy":
		handleEconomyCmd(s, i)
	case "mod":
		handleModCmd(s, i)
	case "level":
		handleLevelCmd(s, i)
	case "welcome":
		handleWelcomeCmd(s, i)
	case "music":
		handleMusicCmd(s, i)
	default:
		respondText(s, i, "Unknown command: /"+name)
	}
}
