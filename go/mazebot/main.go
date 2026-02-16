package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type levelDef struct {
	Name       string
	Difficulty string
	Maze       []string
	StartX     int
	StartY     int
	TimeLimit  int
}

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

type gameState struct {
	UserID    string     `json:"userId"`
	Level     int        `json:"level"`
	PlayerX   int        `json:"playerX"`
	PlayerY   int        `json:"playerY"`
	Coins     int        `json:"coins"`
	Lives     int        `json:"lives"`
	Moves     int        `json:"moves"`
	StartTime int64      `json:"startTime"`
	Board     [][]string `json:"board"`
	GameOver  bool       `json:"gameOver"`
	Won       bool       `json:"won"`
}

type mazeSession struct {
	State     gameState
	ChannelID string
	MessageID string
	UserID    string
	Level     levelDef
}

type completion struct {
	Level     int   `json:"level"`
	Time      int   `json:"time"`
	Coins     int   `json:"coins"`
	Timestamp int64 `json:"timestamp"`
}

type userMazeData struct {
	Username    string       `json:"username"`
	Completions []completion `json:"completions"`
	TotalCoins  int          `json:"totalCoins"`
}

type dailyUser struct {
	Username    string `json:"username"`
	Coins       int    `json:"coins"`
	LastClaim   int64  `json:"lastClaim"`
	Streak      int    `json:"streak"`
	TotalClaims int    `json:"totalClaims"`
	BossKills   int    `json:"bossKills"`
}

type raidBoss struct {
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji"`
	MaxHP       int      `json:"maxHP"`
	CurrentHP   int      `json:"currentHP"`
	Description string   `json:"description"`
	Loot        []string `json:"loot"`
	SpawnedAt   int64    `json:"spawnedAt"`
}

type raidParticipant struct {
	Username   string `json:"username"`
	Damage     int    `json:"damage"`
	LastAttack int64  `json:"lastAttack"`
	Attacks    int    `json:"attacks"`
}

type raidGuild struct {
	Boss         *raidBoss                   `json:"boss"`
	Participants map[string]*raidParticipant `json:"participants"`
	SpawnedAt    int64                       `json:"spawnedAt"`
	TotalDamage  int                         `json:"totalDamage"`
}

type quoteGuild struct {
	Quotes []quoteItem `json:"quotes"`
}

type quoteItem struct {
	ID        int         `json:"id"`
	Text      string      `json:"text"`
	Author    quoteAuthor `json:"author"`
	AddedBy   quoteAdder  `json:"addedBy"`
	Timestamp int64       `json:"timestamp"`
	GuildID   string      `json:"guildId"`
}

type quoteAuthor struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type quoteAdder struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type questEntry struct {
	Task       string `json:"task"`
	Difficulty string `json:"difficulty"`
	XP         int    `json:"xp"`
	AssignedAt int64  `json:"assignedAt,omitempty"`
}

type questUser struct {
	Username        string      `json:"username"`
	TotalXP         int         `json:"totalXP"`
	CompletedQuests int         `json:"completedQuests"`
	CurrentQuest    *questEntry `json:"currentQuest"`
}

type lootItem struct {
	Item       string `json:"item"`
	Rarity     string `json:"rarity"`
	ObtainedAt int64  `json:"obtainedAt"`
}

type lootStats struct {
	Common    int `json:"common"`
	Uncommon  int `json:"uncommon"`
	Rare      int `json:"rare"`
	Epic      int `json:"epic"`
	Legendary int `json:"legendary"`
	Cosmic    int `json:"cosmic"`
	Cursed    int `json:"cursed"`
}

type lootUser struct {
	Username   string     `json:"username"`
	Inventory  []lootItem `json:"inventory"`
	Stats      lootStats  `json:"stats"`
	TotalLoots int        `json:"totalLoots"`
}

type battleStats struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Streak   int    `json:"streak"`
}

type triviaScore struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
	Correct  int    `json:"correct"`
	Total    int    `json:"total"`
}

type triviaSession struct {
	UserID        string
	CorrectIndex  int
	CorrectAnswer string
	Points        int
	ExpiresAt     int64
}

var (
	sessionMu      sync.Mutex
	sessions       = map[string]*mazeSession{}
	dataMu         sync.Mutex
	triviaMu       sync.Mutex
	triviaSessions = map[string]triviaSession{}
)

func main() {
	rand.Seed(time.Now().UnixNano())
	// Load repo-root .env first (go/mazebot/.env usually does not exist).
	if err := godotenv.Load(filepath.Join("..", "..", ".env")); err != nil {
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
	s.Identify.Intents = discordgo.IntentsGuilds

	s.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		log.Printf("go bot ready as %s", r.User.Username)
	})
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handleCommand(s, i)
		case discordgo.InteractionMessageComponent:
			handleComponent(s, i)
		}
	})

	if err := s.Open(); err != nil {
		log.Fatalf("open session: %v", err)
	}
	defer s.Close()

	if err := registerFunCommands(s, clientID, guildID); err != nil {
		log.Fatalf("register commands: %v", err)
	}

	log.Println("go runtime active (fun commands)")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

func registerFunCommands(s *discordgo.Session, appID, guildID string) error {
	cmds := funCommands()
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

func funCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{Name: "8ball", Description: "Ask the magic 8-ball a question", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "Your question", Required: true}}},
		{Name: "battle", Description: "Challenge someone to battle!", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "The user to battle", Required: true}}},
		{Name: "bossraid", Description: "Participate in the server-wide boss raid!", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "status", Description: "Check the current boss status"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "attack", Description: "Attack the boss!"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "spawn", Description: "Spawn a new boss (Admin only)"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View raid damage leaderboard"},
		}},
		{Name: "coinflip", Description: "Flip a coin"},
		{Name: "daily", Description: "Claim your daily rewards and check stats", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "claim", Description: "Claim your daily reward"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your reward stats"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the coins leaderboard"},
		}},
		{Name: "dice", Description: "Roll a dice", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "sides", Description: "Dice sides", Required: false}, {Type: discordgo.ApplicationCommandOptionInteger, Name: "count", Description: "How many dice", Required: false}}},
		{Name: "flip-text", Description: "Flip text upside down", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text to flip", Required: true}}},
		{Name: "hotseat", Description: "Put a random server member in the hotseat with a spicy question"},
		{Name: "joke", Description: "Get a random joke", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Joke category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Any", Value: "Any"}, {Name: "Programming", Value: "Programming"}, {Name: "Miscellaneous", Value: "Misc"}, {Name: "Pun", Value: "Pun"}, {Name: "Spooky", Value: "Spooky"}, {Name: "Christmas", Value: "Christmas"}}}}},
		{Name: "loot", Description: "Open a loot chest and see what you get!", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "open", Description: "Open a loot chest"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "inventory", Description: "View your inventory"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your loot statistics"},
		}},
		{Name: "maze", Description: "Play an interactive maze game with directional controls!", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Choose difficulty level", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Easy - Beginner's Path", Value: 0}, {Name: "Medium - Spike Corridor", Value: 1}, {Name: "Hard - Monster Maze", Value: 2}}}}},
		{Name: "maze-leaderboard", Description: "View maze game leaderboards", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Filter by level", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "All Levels", Value: -1}, {Name: "Easy - Beginner's Path", Value: 0}, {Name: "Medium - Spike Corridor", Value: 1}, {Name: "Hard - Monster Maze", Value: 2}}}}},
		{Name: "meme", Description: "Get a random meme from Reddit"},
		{Name: "mock-text", Description: "CoNvErT tExT tO aLtErNaTiNg CaPs", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text", Required: true}}},
		{Name: "quest", Description: "Get a random quest to complete!", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "get", Description: "Get a new quest"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "complete", Description: "Mark your current quest as complete"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the quest leaderboard"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your quest statistics"},
		}},
		{Name: "quote", Description: "Manage iconic server quotes", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "add", Description: "Add a quote to the collection", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "The quote text", Required: true},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "author", Description: "Who said it", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "random", Description: "Get a random quote"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "List recent quotes"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remove", Description: "Remove a quote by ID (moderators only)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "id", Description: "Quote ID to remove", Required: true},
			}},
		}},
		{Name: "random-choice", Description: "Pick a random option from a list", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "options", Description: "Comma separated options", Required: true}}},
		{Name: "random-number", Description: "Generate a random number", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "min", Description: "Minimum", Required: false}, {Type: discordgo.ApplicationCommandOptionInteger, Name: "max", Description: "Maximum", Required: false}}},
		{Name: "rate", Description: "Rate something out of 10", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "thing", Description: "Thing to rate", Required: true}}},
		{Name: "reverse-text", Description: "Reverse text backwards", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text", Required: true}}},
		{Name: "roll", Description: "Roll custom dice with dramatic flair", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "dice", Description: "Example: 2d6", Required: true}}},
		{Name: "rps", Description: "Play rock, paper, scissors", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "choice", Description: "rock, paper, or scissors", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "rock", Value: "rock"}, {Name: "paper", Value: "paper"}, {Name: "scissors", Value: "scissors"}}}}},
		{Name: "ship", Description: "Calculate compatibility between two users", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user1", Description: "First user", Required: true},
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user2", Description: "Second user", Required: true},
		}},
		{Name: "summon", Description: "Dramatically summon someone to the conversation", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The person to summon", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Why are you summoning them?", Required: false},
		}},
		{Name: "trivia", Description: "Answer trivia questions and compete for high scores!", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Question category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "General Knowledge", Value: "9"},
				{Name: "Science & Nature", Value: "17"},
				{Name: "Computers", Value: "18"},
				{Name: "Mathematics", Value: "19"},
				{Name: "Sports", Value: "21"},
				{Name: "Geography", Value: "22"},
				{Name: "History", Value: "23"},
				{Name: "Animals", Value: "27"},
			}},
			{Type: discordgo.ApplicationCommandOptionString, Name: "difficulty", Description: "Question difficulty", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "Easy", Value: "easy"},
				{Name: "Medium", Value: "medium"},
				{Name: "Hard", Value: "hard"},
			}},
		}},
		{Name: "trivia-leaderboard", Description: "View the trivia leaderboard"},
		{Name: "vibecheck", Description: "Check your current vibe rating", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Check someone else's vibe", Required: false}}},
		{Name: "would-you-rather", Description: "Get a would you rather question"},
	}
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	switch name {
	case "maze":
		handleMazeStart(s, i)
	case "maze-leaderboard":
		handleMazeLeaderboard(s, i)
	default:
		handleSimpleFun(s, i)
	}
}

func handleSimpleFun(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name
	opts := i.ApplicationCommandData().Options

	switch cmd {
	case "coinflip":
		if rand.Intn(2) == 0 {
			respondText(s, i, "\U0001FA99 Heads")
		} else {
			respondText(s, i, "\U0001FA99 Tails")
		}
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
		respondText(s, i, fmt.Sprintf("\U0001F3B2 Rolled %dd%d: [%s]\n\U0001F4CA Total: %d", count, sides, strings.Join(rolls, ", "), total))
	case "8ball":
		answers := []string{"Yes", "No", "Maybe", "Definitely", "Unclear", "Ask later"}
		question := optionString(opts, "question", "")
		respondText(s, i, fmt.Sprintf("\U0001F52E Question: %s\n\U0001F3B1 Answer: %s", question, answers[rand.Intn(len(answers))]))
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
		if max == min {
			respondText(s, i, fmt.Sprintf("%d", min))
		} else {
			respondText(s, i, fmt.Sprintf("%d", rand.Int63n(max-min+1)+min))
		}
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
			respondText(s, i, "No options provided.")
		} else {
			respondText(s, i, fmt.Sprintf("Picked: %s", items[rand.Intn(len(items))]))
		}
	case "rate":
		thing := optionString(opts, "thing", "that")
		respondText(s, i, fmt.Sprintf("I rate %s %d/10", thing, rand.Intn(10)+1))
	case "reverse-text":
		src := optionString(opts, "text", "")
		respondText(s, i, reverse(src))
	case "mock-text":
		src := optionString(opts, "text", "")
		respondText(s, i, mockCase(src))
	case "flip-text":
		src := optionString(opts, "text", "")
		respondText(s, i, reverse(src))
	case "rps":
		user := strings.ToLower(optionString(opts, "choice", "rock"))
		picks := []string{"rock", "paper", "scissors"}
		bot := picks[rand.Intn(3)]
		respondText(s, i, fmt.Sprintf("You: %s | Me: %s", user, bot))
	case "roll":
		expr := optionString(opts, "dice", "1d20")
		handleRoll(s, i, expr)
	case "ship":
		handleShip(s, i, opts)
	case "summon":
		handleSummon(s, i, opts)
	case "vibecheck":
		handleVibeCheck(s, i, opts)
	case "would-you-rather":
		handleWouldYouRather(s, i)
	case "hotseat":
		handleHotseat(s, i)
	case "joke":
		handleJoke(s, i, opts)
	case "meme":
		handleMeme(s, i)
	case "trivia":
		handleTrivia(s, i, opts)
	case "trivia-leaderboard":
		handleTriviaLeaderboard(s, i)
	case "battle":
		handleBattle(s, i, opts)
	case "daily":
		handleDaily(s, i, opts)
	case "bossraid":
		handleBossRaid(s, i, opts)
	case "quote":
		handleQuote(s, i, opts)
	case "quest":
		handleQuest(s, i, opts)
	case "loot":
		handleLoot(s, i, opts)
	default:
		respondText(s, i, fmt.Sprintf("/%s is now running in Go.", cmd))
	}
}

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: text},
	})
}

func optionString(opts []*discordgo.ApplicationCommandInteractionDataOption, name, def string) string {
	for _, o := range opts {
		if o.Name == name {
			return o.StringValue()
		}
	}
	return def
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func mockCase(s string) string {
	out := make([]rune, 0, len([]rune(s)))
	up := false
	for _, ch := range []rune(s) {
		if up {
			out = append(out, []rune(strings.ToUpper(string(ch)))...)
		} else {
			out = append(out, []rune(strings.ToLower(string(ch)))...)
		}
		if ch != ' ' {
			up = !up
		}
	}
	return string(out)
}

var summonMessages = []string{
	"The council demands your presence, %s.",
	"Through ancient rites, we call upon %s.",
	"By order of the realm, %s is summoned.",
	"From the shadows we call upon %s.",
	"The stars align for %s.",
}

var hotseatQuestions = []string{
	"What's your most embarrassing Discord moment?",
	"If you had to delete one app forever, what is it?",
	"What's something you're supposed to be doing right now?",
	"What's your most unpopular opinion?",
	"What's your current phone battery percentage?",
}

var wyrQuestions = [][2]string{
	{"Have the ability to fly", "Have the ability to be invisible"},
	{"Live without music", "Live without movies"},
	{"Be able to teleport", "Be able to time travel"},
	{"Fight one horse-sized duck", "Fight 100 duck-sized horses"},
}

var questTemplates = []questEntry{
	{Task: "Retrieve the sacred pizza from the Forbidden Fridge", Difficulty: "easy", XP: 50},
	{Task: "Defeat the Lag Monster in the Router Realm", Difficulty: "medium", XP: 100},
	{Task: "Survive a family gathering without checking your phone", Difficulty: "hard", XP: 200},
	{Task: "Complete a group project where everyone contributes", Difficulty: "legendary", XP: 500},
}

var raidBosses = []raidBoss{
	{Name: "The Lag Monster", Emoji: "\U0001F47E", MaxHP: 5000, Description: "A creature that feeds on your WiFi", Loot: []string{"Router of Legends", "Ping Reducer", "5G Crystal"}},
	{Name: "The Monday Overlord", Emoji: "\U0001F4C5", MaxHP: 7000, Description: "The most feared entity", Loot: []string{"Weekend Extension", "Coffee of Awakening", "Skip Monday Pass"}},
	{Name: "Social Anxiety Dragon", Emoji: "\U0001F409", MaxHP: 5500, Description: "Guards social skills treasure", Loot: []string{"Conversation Starter Kit", "Confidence Amulet", "Small Talk Guide"}},
}

type lootTable struct {
	Name  string
	Emoji string
	Items []string
}

var lootTables = map[string]lootTable{
	"common":    {Name: "Common", Emoji: "\u26AA", Items: []string{"Broken Pencil", "Old Receipt", "Bottle Cap"}},
	"uncommon":  {Name: "Uncommon", Emoji: "\U0001F7E2", Items: []string{"Working Charger", "Good Vibes", "Fresh Pizza Slice"}},
	"rare":      {Name: "Rare", Emoji: "\U0001F535", Items: []string{"Productive Day", "Extra Fries in Bag", "Reply from Crush"}},
	"epic":      {Name: "Epic", Emoji: "\U0001F7E3", Items: []string{"WiFi That Actually Works", "Inbox Zero Achievement", "Main Character Moment"}},
	"legendary": {Name: "Legendary", Emoji: "\U0001F7E1", Items: []string{"Extra Day Weekend", "Unlimited Garlic Bread", "Infinite Battery Life"}},
	"cosmic":    {Name: "Cosmic", Emoji: "\U0001F30C", Items: []string{"Pause Time", "World Peace Token", "Respec Button for Life Choices"}},
	"cursed":    {Name: "Cursed", Emoji: "\U0001F480", Items: []string{"Wet Socks (Permanent)", "Eternal Loading Screen", "Unstoppable Hiccups"}},
}

func dataPath(file string) string {
	return filepath.Join("..", "..", "src", "data", file)
}

func readData(file string, out any) error {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) == "" {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, out)
}

func writeData(file string, in any) error {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := dataPath(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func getSubcommand(opts []*discordgo.ApplicationCommandInteractionDataOption) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
	for _, o := range opts {
		if o.Type == discordgo.ApplicationCommandOptionSubCommand {
			return o.Name, o.Options
		}
	}
	return "", opts
}

func optionInt(opts []*discordgo.ApplicationCommandInteractionDataOption, name string, def int64) int64 {
	for _, o := range opts {
		if o.Name == name {
			return o.IntValue()
		}
	}
	return def
}

func optionUser(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.User {
	for _, o := range opts {
		if o.Name == name {
			return o.UserValue(nil)
		}
	}
	return nil
}

func optionUserID(opts []*discordgo.ApplicationCommandInteractionDataOption, name, def string) string {
	for _, o := range opts {
		if o.Name == name {
			if o.Value != nil {
				if id, ok := o.Value.(string); ok {
					return id
				}
			}
		}
	}
	return def
}

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, e *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{e}},
	})
}

func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i == nil {
		return nil
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	if i.User != nil {
		return i.User
	}
	if i.Interaction != nil {
		if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			return i.Interaction.Member.User
		}
		if i.Interaction.User != nil {
			return i.Interaction.User
		}
	}
	return nil
}

func handleRoll(s *discordgo.Session, i *discordgo.InteractionCreate, expr string) {
	count, sides, mod, ok := parseDice(expr)
	if !ok {
		respondText(s, i, "Invalid dice notation. Example: d20, 3d6, 2d10+5")
		return
	}
	if count > 20 || sides < 2 || sides > 1000 {
		respondText(s, i, "Dice limits: up to 20 dice, and 2-1000 sides.")
		return
	}
	rolls := make([]int, 0, count)
	total := 0
	for j := 0; j < count; j++ {
		r := rand.Intn(sides) + 1
		rolls = append(rolls, r)
		total += r
	}
	final := total + mod
	respondText(s, i, fmt.Sprintf("Roll %s\nRolls: %v\nTotal: %d", expr, rolls, final))
}

func parseDice(expr string) (int, int, int, bool) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	parts := strings.Split(expr, "d")
	if len(parts) != 2 {
		return 0, 0, 0, false
	}
	count := 1
	if parts[0] != "" {
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, false
		}
		count = v
	}
	sidesPart := parts[1]
	mod := 0
	sides := 0
	if strings.Contains(sidesPart, "+") {
		sub := strings.SplitN(sidesPart, "+", 2)
		v, e1 := strconv.Atoi(sub[0])
		m, e2 := strconv.Atoi(sub[1])
		if e1 != nil || e2 != nil {
			return 0, 0, 0, false
		}
		sides = v
		mod = m
	} else if strings.Contains(sidesPart, "-") {
		sub := strings.SplitN(sidesPart, "-", 2)
		v, e1 := strconv.Atoi(sub[0])
		m, e2 := strconv.Atoi(sub[1])
		if e1 != nil || e2 != nil {
			return 0, 0, 0, false
		}
		sides = v
		mod = -m
	} else {
		v, err := strconv.Atoi(sidesPart)
		if err != nil {
			return 0, 0, 0, false
		}
		sides = v
	}
	return count, sides, mod, true
}

func handleShip(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u1 := optionUser(opts, "user1")
	u2 := optionUser(opts, "user2")
	if u1 == nil || u2 == nil {
		respondText(s, i, "Both users are required.")
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
	respondText(s, i, fmt.Sprintf("%s + %s = %d%% compatibility", u1.Username, u2.Username, comp))
}

func handleSummon(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	u := optionUser(opts, "user")
	if u == nil {
		respondText(s, i, "User is required.")
		return
	}
	reason := optionString(opts, "reason", "")
	msg := fmt.Sprintf(summonMessages[rand.Intn(len(summonMessages))], "<@"+u.ID+">")
	if reason != "" {
		msg += "\nReason: " + reason
	}
	respondText(s, i, msg)
}

func handleVibeCheck(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	target := interactionUser(i)
	if target == nil {
		respondText(s, i, "Unable to identify user for vibe check.")
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
	respondText(s, i, fmt.Sprintf("\U0001FAE8 %s vibe: %s (%d%%)", target.Username, vibes[idx], pct))
}

func handleWouldYouRather(s *discordgo.Session, i *discordgo.InteractionCreate) {
	q := wyrQuestions[rand.Intn(len(wyrQuestions))]
	respondText(s, i, fmt.Sprintf("Would you rather:\n1) %s\n2) %s", q[0], q[1]))
}

func handleHotseat(s *discordgo.Session, i *discordgo.InteractionCreate) {
	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil || len(members) == 0 {
		respondText(s, i, "Failed to fetch server members.")
		return
	}
	candidates := make([]*discordgo.Member, 0, len(members))
	for _, m := range members {
		if m.User != nil && !m.User.Bot {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		respondText(s, i, "No human members found.")
		return
	}
	m := candidates[rand.Intn(len(candidates))]
	q := hotseatQuestions[rand.Intn(len(hotseatQuestions))]
	respondText(s, i, fmt.Sprintf("<@%s> hotseat question:\n%s", m.User.ID, q))
}

func handleJoke(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	category := optionString(opts, "category", "Any")
	url := fmt.Sprintf("https://v2.jokeapi.dev/joke/%s?blacklistFlags=nsfw,religious,political,racist,sexist,explicit", category)
	body, err := httpGet(url)
	if err != nil {
		respondText(s, i, "Failed to fetch joke.")
		return
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		respondText(s, i, "Failed to parse joke.")
		return
	}
	typ, _ := data["type"].(string)
	if typ == "single" {
		respondText(s, i, fmt.Sprintf("%v", data["joke"]))
		return
	}
	respondText(s, i, fmt.Sprintf("%v\n\n||%v||", data["setup"], data["delivery"]))
}

func handleMeme(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subreddits := []string{"memes", "dankmemes", "wholesomememes", "me_irl", "AdviceAnimals"}
	sub := subreddits[rand.Intn(len(subreddits))]
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=100", sub)
	body, err := httpGet(url)
	if err != nil {
		respondText(s, i, "Failed to fetch meme.")
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
		respondText(s, i, "Failed to parse meme response.")
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
		respondText(s, i, "No meme found right now.")
		return
	}
	p := posts[rand.Intn(len(posts))]
	embed := &discordgo.MessageEmbed{
		Title:       p.Title,
		Description: fmt.Sprintf("\U0001F44D %d upvotes | \U0001F4AC %d comments", p.Ups, p.NumComments),
		Color:       0xEB459E,
		URL:         "https://reddit.com" + p.Permalink,
		Image:       &discordgo.MessageEmbedImage{URL: p.URL},
		Footer:      &discordgo.MessageEmbedFooter{Text: "r/" + sub + " ï¿½ Discorbo"},
	}
	respondEmbed(s, i, embed)
}

func httpGet(url string) ([]byte, error) {
	client := http.Client{Timeout: 7 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "DiscorboBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func handleTrivia(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for trivia.")
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
		respondText(s, i, "Trivia API is unavailable.")
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
		respondText(s, i, "No trivia question available.")
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
		Description: fmt.Sprintf("**%s**\\n\\n\U0001F3AF Difficulty: %s\\n\U0001F4AF Points: %d", html.UnescapeString(q.Question), strings.ToUpper(q.Difficulty), points),
		Color:       0xEB459E,
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: rows,
		},
	})
}

func htmlUnescapeSlice(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, html.UnescapeString(v))
	}
	return out
}

func handleTriviaLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	respondText(s, i, fmt.Sprintf("\u2694\uFE0F Winner: %s\n\u2764\uFE0F Final HP: %d\n%s", winner.Username, winner.HP, last))
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

func handleMazeStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		respondText(s, i, "Unable to identify user for maze.")
		return
	}
	levelIndex := int64(0)
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "level" {
			levelIndex = o.IntValue()
		}
	}
	if levelIndex < 0 || int(levelIndex) >= len(levels) {
		levelIndex = 0
	}
	level := levels[levelIndex]

	state := gameState{
		UserID:    user.ID,
		Level:     int(levelIndex),
		PlayerX:   level.StartX,
		PlayerY:   level.StartY,
		Coins:     0,
		Lives:     3,
		Moves:     0,
		StartTime: time.Now().UnixMilli(),
		Board:     toBoard(level.Maze),
		GameOver:  false,
		Won:       false,
	}

	embed := buildMazeEmbed(state, level, false)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: mazeComponents(),
		},
	})
	if err != nil {
		log.Printf("maze respond error: %v", err)
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Printf("maze fetch response error: %v", err)
		return
	}

	sessionMu.Lock()
	sessions[msg.ID] = &mazeSession{
		State:     state,
		ChannelID: msg.ChannelID,
		MessageID: msg.ID,
		UserID:    user.ID,
		Level:     level,
	}
	sessionMu.Unlock()

	go func(channelID, messageID string, timeoutSec int) {
		<-time.After(time.Duration(timeoutSec) * time.Second)

		sessionMu.Lock()
		sess, ok := sessions[messageID]
		if !ok || sess.State.GameOver {
			sessionMu.Unlock()
			return
		}
		sess.State.GameOver = true
		sess.State.Won = false
		stateCopy := sess.State
		levelCopy := sess.Level
		delete(sessions, messageID)
		sessionMu.Unlock()

		embed := buildMazeEmbed(stateCopy, levelCopy, true)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         messageID,
			Channel:    channelID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &[]discordgo.MessageComponent{},
		})
	}(msg.ChannelID, msg.ID, level.TimeLimit)
}

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	data := i.MessageComponentData()
	if strings.HasPrefix(data.CustomID, "trivia_") {
		handleTriviaComponent(s, i)
		return
	}
	if !strings.HasPrefix(data.CustomID, "maze_") {
		return
	}

	sessionMu.Lock()
	sess, ok := sessions[i.Message.ID]
	if !ok {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	if sess.UserID != user.ID {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "Only the starter can control this maze.", Flags: discordgo.MessageFlagsEphemeral}})
		return
	}
	if sess.State.GameOver {
		sessionMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}

	dir := strings.TrimPrefix(data.CustomID, "maze_")
	_ = step(&sess.State, dir)

	if sess.State.Won {
		completionTime := int((time.Now().UnixMilli() - sess.State.StartTime) / 1000)
		saveMazeCompletion(user.ID, user.Username, sess.State.Level, completionTime, sess.State.Coins)
	}

	gameOver := sess.State.GameOver
	stateCopy := sess.State
	levelCopy := sess.Level
	if gameOver {
		delete(sessions, i.Message.ID)
	}
	sessionMu.Unlock()

	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{buildMazeEmbed(stateCopy, levelCopy, false)}}}
	if gameOver {
		resp.Data.Components = []discordgo.MessageComponent{}
	} else {
		resp.Data.Components = mazeComponents()
	}
	_ = s.InteractionRespond(i.Interaction, resp)
}

func handleTriviaComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := interactionUser(i)
	if user == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) < 4 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	token := parts[1] + "_" + parts[2]
	selected, err := strconv.Atoi(parts[3])
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
		return
	}
	triviaMu.Lock()
	sess, ok := triviaSessions[token]
	if ok && time.Now().UnixMilli() > sess.ExpiresAt {
		delete(triviaSessions, token)
		ok = false
	}
	if !ok {
		triviaMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "This trivia question has expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if user.ID != sess.UserID {
		triviaMu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Only the original player can answer.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	delete(triviaSessions, token)
	triviaMu.Unlock()

	scores := map[string]triviaScore{}
	_ = readData("trivia-scores.json", &scores)
	sc := scores[user.ID]
	sc.Username = user.Username
	sc.Total++
	correct := selected == sess.CorrectIndex
	if correct {
		sc.Correct++
		sc.Score += sess.Points
	}
	scores[user.ID] = sc
	_ = writeData("trivia-scores.json", scores)

	content := fmt.Sprintf("Incorrect. Correct answer: %s\nScore: %d", sess.CorrectAnswer, sc.Score)
	if correct {
		content = fmt.Sprintf("Correct! +%d points\nAnswer: %s\nScore: %d", sess.Points, sess.CorrectAnswer, sc.Score)
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func handleMazeLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	levelFilter := int64(-1)
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "level" {
			levelFilter = o.IntValue()
		}
	}
	data, _ := readMazeScores()
	if len(data) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "No maze completions yet."}})
		return
	}

	if levelFilter == -1 {
		type row struct {
			Name        string
			Coins       int
			Completions int
		}
		rows := make([]row, 0, len(data))
		for _, v := range data {
			rows = append(rows, row{Name: v.Username, Coins: v.TotalCoins, Completions: len(v.Completions)})
		}
		sort.Slice(rows, func(a, b int) bool {
			if rows[a].Coins != rows[b].Coins {
				return rows[a].Coins > rows[b].Coins
			}
			return rows[a].Completions > rows[b].Completions
		})
		if len(rows) > 10 {
			rows = rows[:10]
		}
		lines := []string{}
		for i2, r := range rows {
			lines = append(lines, fmt.Sprintf("%d. %s - Coins:%d Completions:%d", i2+1, r.Name, r.Coins, r.Completions))
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: strings.Join(lines, "\n")}})
		return
	}

	type entry struct {
		Name  string
		Time  int
		Coins int
	}
	rows := []entry{}
	for _, u := range data {
		for _, c := range u.Completions {
			if c.Level == int(levelFilter) {
				rows = append(rows, entry{Name: u.Username, Time: c.Time, Coins: c.Coins})
			}
		}
	}
	if len(rows) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "No completions for this level yet."}})
		return
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].Time != rows[b].Time {
			return rows[a].Time < rows[b].Time
		}
		return rows[a].Coins > rows[b].Coins
	})
	if len(rows) > 10 {
		rows = rows[:10]
	}
	lines := []string{}
	for i2, r := range rows {
		lines = append(lines, fmt.Sprintf("%d. %s - %ds (%d coins)", i2+1, r.Name, r.Time, r.Coins))
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: strings.Join(lines, "\n")}})
}

func step(state *gameState, direction string) error {
	if state.GameOver {
		return nil
	}
	dx, dy, ok := dirDelta(direction)
	if !ok {
		return errors.New("invalid direction")
	}
	newX := state.PlayerX + dx
	newY := state.PlayerY + dy
	if newY < 0 || newY >= len(state.Board) {
		return nil
	}
	if newX < 0 || newX >= len(state.Board[newY]) {
		return nil
	}
	target := state.Board[newY][newX]
	if target == "#" {
		return nil
	}

	state.Board[state.PlayerY][state.PlayerX] = "."
	switch target {
	case "C":
		state.Coins++
	case "S", "E":
		state.Lives--
		if state.Lives <= 0 {
			state.GameOver = true
			state.Won = false
		}
	case "G":
		state.GameOver = true
		state.Won = true
	}
	state.PlayerX = newX
	state.PlayerY = newY
	state.Board[newY][newX] = "P"
	state.Moves++
	return nil
}

func dirDelta(direction string) (int, int, bool) {
	switch direction {
	case "up":
		return 0, -1, true
	case "down":
		return 0, 1, true
	case "left":
		return -1, 0, true
	case "right":
		return 1, 0, true
	default:
		return 0, 0, false
	}
}

func toBoard(rows []string) [][]string {
	board := make([][]string, 0, len(rows))
	for _, row := range rows {
		cur := make([]string, 0, len(row))
		for _, ch := range row {
			cur = append(cur, string(ch))
		}
		board = append(board, cur)
	}
	return board
}

func buildMazeEmbed(state gameState, level levelDef, timeout bool) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(state.Board))
	for _, row := range state.Board {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			cells = append(cells, mazeCellEmoji(cell))
		}
		lines = append(lines, strings.Join(cells, ""))
	}
	board := strings.Join(lines, "\n")
	elapsed := int((time.Now().UnixMilli() - state.StartTime) / 1000)
	timeLeft := level.TimeLimit - elapsed
	if timeLeft < 0 {
		timeLeft = 0
	}

	title := fmt.Sprintf("\U0001F9E9 Maze: %s (%s)", level.Name, level.Difficulty)
	color := 0xEB459E
	result := ""
	if state.GameOver {
		if state.Won {
			title = "\U0001F3C6 Maze: Victory"
			color = 0x57F287
			result = "\U0001F973 You escaped the maze."
		}
		if timeout {
			title = "\u23F0 Maze: Time Up"
			color = 0xFEE75C
			result = "\u23F1\uFE0F Time ran out."
		}
		if !state.Won && !timeout {
			title = "\u2620\uFE0F Maze: Game Over"
			color = 0xED4245
			result = "\U0001F494 You lost all lives."
		}
	}

	desc := fmt.Sprintf("```text\n%s\n```\n", board)
	if result != "" {
		desc += result + "\n\n"
	}
	desc += fmt.Sprintf("\u2764\uFE0F Lives: %d/3\n\U0001FA99 Coins: %d\n\u23F3 Time: %ds\n\U0001F3C3 Moves: %d\n\nKey: \U0001F7E6=You, \u2B1C=Path, \U0001F7EB=Wall, \U0001F3C1=Goal, \U0001FA99=Coin, \U0001F525=Spike, \U0001F47E=Enemy", state.Lives, state.Coins, timeLeft, state.Moves)

	return &discordgo.MessageEmbed{Title: title, Description: desc, Color: color}
}

func mazeCellEmoji(cell string) string {
	switch cell {
	case "P":
		return "\U0001F7E6"
	case ".":
		return "\u2B1C"
	case "#":
		return "\U0001F7EB"
	case "G":
		return "\U0001F3C1"
	case "C":
		return "\U0001FA99"
	case "S":
		return "\U0001F525"
	case "E":
		return "\U0001F47E"
	default:
		return cell
	}
}

func mazeComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "\u2B06\uFE0F Up", Style: discordgo.PrimaryButton, CustomID: "maze_up"}}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "\u2B05\uFE0F Left", Style: discordgo.PrimaryButton, CustomID: "maze_left"},
			discordgo.Button{Label: "\u2B07\uFE0F Down", Style: discordgo.PrimaryButton, CustomID: "maze_down"},
			discordgo.Button{Label: "\u27A1\uFE0F Right", Style: discordgo.PrimaryButton, CustomID: "maze_right"},
		}},
	}
}

func mazeScoresPath() string { return filepath.Join("..", "..", "src", "data", "maze-scores.json") }

func readMazeScores() (map[string]userMazeData, error) {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := mazeScoresPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(path, []byte("{}"), 0o644)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]userMazeData{}
	if strings.TrimSpace(string(raw)) == "" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]userMazeData{}, nil
	}
	return out, nil
}

func saveMazeCompletion(userID, username string, level, sec, coins int) {
	dataMu.Lock()
	defer dataMu.Unlock()
	path := mazeScoresPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	raw, _ := os.ReadFile(path)
	all := map[string]userMazeData{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		_ = json.Unmarshal(raw, &all)
	}
	u := all[userID]
	if u.Username == "" {
		u.Username = username
	}
	u.Username = username
	u.Completions = append(u.Completions, completion{Level: level, Time: sec, Coins: coins, Timestamp: time.Now().UnixMilli()})
	u.TotalCoins += coins
	all[userID] = u
	out, _ := json.MarshalIndent(all, "", "  ")
	_ = os.WriteFile(path, out, 0o644)
}
