package main

import (
	"fmt"
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
	s.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	s.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Go bot ready as %s", r.User.Username)
	})
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handleCommand(s, i)
		case discordgo.InteractionMessageComponent:
			handleComponent(s, i)
		}
	})
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handleMessageCreate(s, m)
	})

	if err := s.Open(); err != nil {
		log.Fatalf("open session: %v", err)
	}
	defer s.Close()

	if err := registerCommands(s, clientID, guildID); err != nil {
		log.Fatalf("register commands: %v", err)
	}
	startReminderLoop(s)

	log.Println("Go runtime active (all commands)")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
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
	case "maze":
		handleMaze(s, i)
	case "maze-leaderboard":
		handleMazeLeaderboard(s, i)
	case "8ball", "coinflip", "dice", "random-number", "random-choice", "rate",
		"reverse-text", "mock-text", "flip-text", "rps", "roll":
		handleSimpleFun(s, i)
	case "ship", "summon", "vibecheck", "would-you-rather", "hotseat":
		handleInteractiveFun(s, i)
	case "joke", "meme", "trivia", "trivia-leaderboard":
		handleAPIFun(s, i)
	case "battle", "daily", "bossraid", "quote", "quest", "loot":
		handleGameFun(s, i)
	case "ping", "help", "avatar", "userinfo", "serverinfo", "channel-info", "role-info", "stats":
		handleInfoUtility(s, i)
	case "calc", "translate", "poll", "timer":
		handleToolUtility(s, i)
	case "remind", "reminders", "afk", "clear-my-data":
		handleDataUtility(s, i)
	default:
		respondText(s, i, fmt.Sprintf("/%s is implemented in Go.", name))
	}
}
