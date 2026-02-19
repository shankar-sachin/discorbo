package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Maze game types
type levelDef struct {
	Name       string
	Difficulty string
	Maze       []string
	StartX     int
	StartY     int
	TimeLimit  int
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

// Daily rewards types
type dailyUser struct {
	Username    string `json:"username"`
	Coins       int    `json:"coins"`
	LastClaim   int64  `json:"lastClaim"`
	Streak      int    `json:"streak"`
	TotalClaims int    `json:"totalClaims"`
	BossKills   int    `json:"bossKills"`
}

// Boss raid types
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

// Quote types
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

// Quest types
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

// Loot types
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

type lootTable struct {
	Name  string
	Emoji string
	Items []string
}

// Battle types
type battleStats struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Streak   int    `json:"streak"`
}

// Trivia types
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

// AFK types
type afkStatus struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// Reminder types
type reminderEntry struct {
	UserID    string `json:"userId"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"createdAt"`
	DueTime   int64  `json:"dueTime"`
	GuildID   string `json:"guildId,omitempty"`
}

// Economy types
type shopItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       int     `json:"price"`
	Emoji       string  `json:"emoji"`
	Category    string  `json:"category"` // boost, collectible, consumable
	MaxOwned    int     `json:"maxOwned"` // -1 unlimited
	BoostType   string  `json:"boostType,omitempty"`
	BoostValue  float64 `json:"boostValue,omitempty"`
}

type economyUser struct {
	Username     string          `json:"username"`
	Coins        int             `json:"coins"`
	Inventory    []inventoryItem `json:"inventory"`
	ActiveBoosts []activeBoost   `json:"activeBoosts"`
	TradeHistory []tradeRecord   `json:"tradeHistory"`
	TotalSpent   int             `json:"totalSpent"`
	TotalEarned  int             `json:"totalEarned"`
}

type inventoryItem struct {
	ItemID      string `json:"itemId"`
	Quantity    int    `json:"quantity"`
	PurchasedAt int64  `json:"purchasedAt"`
}

type activeBoost struct {
	BoostType  string  `json:"boostType"`
	Multiplier float64 `json:"multiplier"`
	ExpiresAt  int64   `json:"expiresAt"`
}

type tradeRecord struct {
	TradeID   string   `json:"tradeId"`
	OtherUser string   `json:"otherUser"`
	GaveCoins int      `json:"gaveCoins"`
	GaveItems []string `json:"gaveItems"`
	GotCoins  int      `json:"gotCoins"`
	GotItems  []string `json:"gotItems"`
	Timestamp int64    `json:"timestamp"`
}

// In-memory trade sessions
type tradeSession struct {
	TradeID         string
	InitiatorID     string
	TargetID        string
	InitOffer       tradeOffer
	TargetOffer     tradeOffer
	InitConfirmed   bool
	TargetConfirmed bool
	ExpiresAt       int64
	MessageID       string
	ChannelID       string
}

type tradeOffer struct {
	Coins   int
	ItemIDs []string
}

type transactionLog struct {
	UserID    string `json:"userId"`
	Type      string `json:"type"` // purchase, sell, admin_grant, admin_take, trade
	ItemID    string `json:"itemId,omitempty"`
	Amount    int    `json:"amount"`
	GrantedBy string `json:"grantedBy,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Tag game types
type tagSession struct {
	SessionID   string
	Player1ID   string
	Player1Name string
	Player2ID   string
	Player2Name string
	CurrentTurn int // 0 = P1, 1 = P2
	Player1X    int
	Player1Y    int
	Player2X    int
	Player2Y    int
	Board       [][]string // 5x5 grid with obstacles
	Moves       int
	MaxMoves    int // 20 moves
	StartTime   int64
	MessageID   string
	ChannelID   string
}

type tagStats struct {
	Username    string `json:"username"`
	Wins        int    `json:"wins"`
	Losses      int    `json:"losses"`
	TotalGames  int    `json:"totalGames"`
	CoinsEarned int    `json:"coinsEarned"`
}

// Global state
var (
	botStartedAt   = time.Now()
	sessionMu      sync.Mutex
	sessions       = map[string]*mazeSession{}
	dataMu         sync.Mutex
	triviaMu       sync.Mutex
	triviaSessions = map[string]triviaSession{}
	tradeMu        sync.Mutex
	tradeSessions  = map[string]*tradeSession{}
	tagSessions    = map[string]*tagSession{}
)

// Command definitions
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
		{Name: "tag", Description: "Play a strategic tag game!", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "challenge", Description: "Challenge someone to tag", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "Who to challenge", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View tag leaderboard"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View tag stats", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to check", Required: false},
			}},
		}},
	}
}

func utilityCommands() []*discordgo.ApplicationCommand {
	pollOptions := []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "The poll question", Required: true},
		{Type: discordgo.ApplicationCommandOptionString, Name: "option1", Description: "First option", Required: true},
		{Type: discordgo.ApplicationCommandOptionString, Name: "option2", Description: "Second option", Required: true},
	}
	for idx := 3; idx <= 10; idx++ {
		pollOptions = append(pollOptions, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        fmt.Sprintf("option%d", idx),
			Description: fmt.Sprintf("Option %d", idx),
			Required:    false,
		})
	}

	return []*discordgo.ApplicationCommand{
		{Name: "afk", Description: "Set your AFK status", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Why are you AFK?", Required: false}}},
		{Name: "avatar", Description: "Display a user's avatar", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: false}}},
		{Name: "calc", Description: "Perform mathematical calculations", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "expression", Description: "Expression", Required: true}}},
		{Name: "channel-info", Description: "Display detailed information about a channel", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel", Required: false}}},
		{Name: "clear-my-data", Description: "Remove all your data from the bot", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionBoolean, Name: "confirm", Description: "Confirm deletion", Required: true}}},
		{Name: "help", Description: "Display all available commands and their descriptions", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Filter category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Fun & Games", Value: "fun"}, {Name: "Utility", Value: "utility"}, {Name: "All Commands", Value: "all"}}}}},
		{Name: "ping", Description: "Check bot latency and response time"},
		{Name: "poll", Description: "Create a poll with up to 10 options", Options: pollOptions},
		{Name: "remind", Description: "Set a reminder", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "time", Description: "Time like 30m, 2h", Required: true}, {Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Reminder message", Required: true}}},
		{Name: "reminders", Description: "Manage your active reminders", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "View all your active reminders"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "clear", Description: "Delete all your reminders"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remove", Description: "Remove a specific reminder", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "number", Description: "Reminder number", Required: true}}},
		}},
		{Name: "role-info", Description: "Display detailed information about a role", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role", Required: true}}},
		{Name: "serverinfo", Description: "Display information about this server"},
		{Name: "stats", Description: "Display bot statistics and metrics"},
		{Name: "timer", Description: "Set a countdown timer", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionInteger, Name: "seconds", Description: "Duration in seconds (max 300)", Required: true}}},
		{Name: "translate", Description: "Translate text to another language", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text to translate", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "to", Description: "Target language code (es, fr, de, ...)", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "from", Description: "Source language (default auto)", Required: false},
		}},
		{Name: "userinfo", Description: "Display detailed information about a user", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: false}}},
		{Name: "shop", Description: "Browse and buy items from the shop", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "Browse shop items", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Filter by category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "All", Value: "all"},
					{Name: "Boosts", Value: "boost"},
					{Name: "Collectibles", Value: "collectible"},
					{Name: "Consumables", Value: "consumable"},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "buy", Description: "Purchase an item", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item ID to buy", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "quantity", Description: "How many to buy", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "sell", Description: "Sell an item for 50% refund", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item ID to sell", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "quantity", Description: "How many to sell", Required: false},
			}},
		}},
		{Name: "inventory", Description: "View and manage your inventory", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "view", Description: "View your inventory"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "use", Description: "Use/activate an item", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item to use", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "info", Description: "View item details", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item to inspect", Required: true},
			}},
		}},
		{Name: "balance", Description: "View your coin balance and stats"},
		{Name: "trade", Description: "Trade coins and items with other users", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "offer", Description: "Start a trade with someone", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to trade with", Required: true},
			}},
		}},
		{Name: "economy-admin", Description: "Manage server economy (Admin only)", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "grant", Description: "Give coins to user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to grant coins", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "coins", Description: "Amount of coins", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for grant", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "take", Description: "Remove coins from user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to take coins from", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "coins", Description: "Amount of coins", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for removal", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "transactions", Description: "View transaction history", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to check", Required: false},
			}},
		}},
		{Name: "convert", Description: "Convert between units", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionNumber, Name: "value", Description: "Value to convert", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "from", Description: "Source unit (f, c, k, ft, m, mi, km, lb, kg, etc.)", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "to", Description: "Target unit", Required: true},
		}},
	}
}

func allCommands() []*discordgo.ApplicationCommand {
	return append(funCommands(), utilityCommands()...)
}
