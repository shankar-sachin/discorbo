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
	// Migrated from dailyUser (v1 → v2)
	LastClaim   int64 `json:"lastClaim,omitempty"`
	Streak      int   `json:"streak,omitempty"`
	TotalClaims int   `json:"totalClaims,omitempty"`
	BossKills   int   `json:"bossKills,omitempty"`
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

// Moderation types
type warningEntry struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	ModeratorID string `json:"moderatorId"`
	Moderator   string `json:"moderator"`
	Reason      string `json:"reason"`
	Timestamp   int64  `json:"timestamp"`
	GuildID     string `json:"guildId"`
}

type modAction struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // kick, ban, warn, timeout, purge, lock, unlock, slowmode
	UserID       string `json:"userId"`
	Username     string `json:"username"`
	ModeratorID  string `json:"moderatorId"`
	Moderator    string `json:"moderator"`
	Reason       string `json:"reason"`
	Duration     int64  `json:"duration,omitempty"`     // for timeouts (in seconds)
	MessageCount int    `json:"messageCount,omitempty"` // for purge
	Timestamp    int64  `json:"timestamp"`
	GuildID      string `json:"guildId"`
}

type modReport struct {
	ID         string `json:"id"`
	ReporterID string `json:"reporterId"`
	Reporter   string `json:"reporter"`
	TargetID   string `json:"targetId"`
	Target     string `json:"target"`
	Reason     string `json:"reason"`
	Timestamp  int64  `json:"timestamp"`
	GuildID    string `json:"guildId"`
}

type modNote struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	ModeratorID string `json:"moderatorId"`
	Moderator   string `json:"moderator"`
	Note        string `json:"note"`
	Timestamp   int64  `json:"timestamp"`
	GuildID     string `json:"guildId"`
}

type guildConfig struct {
	GuildID       string        `json:"guildId"`
	ModLogChannel string        `json:"modLogChannel"`
	AutoMod       autoModConfig `json:"autoMod"`
}

type autoModConfig struct {
	Enabled          bool     `json:"enabled"`
	SpamEnabled      bool     `json:"spamEnabled"`
	SpamMessageLimit int      `json:"spamMessageLimit"` // messages per interval
	SpamInterval     int      `json:"spamInterval"`     // seconds
	LinksEnabled     bool     `json:"linksEnabled"`
	InvitesEnabled   bool     `json:"invitesEnabled"`
	AllCapsEnabled   bool     `json:"allCapsEnabled"`
	AllCapsPercent   int      `json:"allCapsPercent"` // 70 = 70% caps = trigger
	ForbiddenWords   []string `json:"forbiddenWords"`
	EmojiSpamEnabled bool     `json:"emojiSpamEnabled"`
	EmojiSpamLimit   int      `json:"emojiSpamLimit"` // max emojis per message
}

// For spam detection tracking
type spamTracker struct {
	MessageTimes []int64
	LastMessage  string
	DuplicateCount int
}

// Casino/card game session types
type bjSession struct {
	UserID     string
	Username   string
	Bet        int
	PlayerHand []string
	DealerHand []string
	Deck       []string
	Done       bool
}

type pokerSession struct {
	UserID   string
	Username string
	Bet      int
	Hand     []string
	Held     [5]bool
	Deck     []string
	Drawn    bool
}

type fishSession struct {
	UserID      string
	PlayerHand  []string
	BotHand     []string
	Deck        []string
	PlayerBooks int
	BotBooks    int
}

type snapSession struct {
	UserID   string
	Username string
	Bet      int
	Deck     []string
	LastTwo  []string
}

type g2048Session struct {
	UserID string
	Grid   [4][4]int
	Score  int
}

type hlSession struct {
	UserID      string
	Username    string
	Bet         int
	CurrentCard string
	Deck        []string
	Streak      int
}

// Poll voting session
type pollSession struct {
	Question  string
	Options   []string
	Votes     []int
	Voters    map[string]int // userID → option index voted
	CreatorID string
	ExpiresAt int64
}

// Tic-Tac-Toe session
type tttSession struct {
	Player1ID   string
	Player1Name string
	Player2ID   string
	Player2Name string
	Board       [3][3]int // 0=empty, 1=P1(X), 2=P2(O)
	CurrentTurn int       // 1 or 2
	MessageID   string
	ChannelID   string
}

// Connect4 session
type c4Session struct {
	Player1ID   string
	Player1Name string
	Player2ID   string
	Player2Name string
	Board       [6][7]int // 0=empty, 1=P1(red), 2=P2(yellow)
	CurrentTurn int       // 1 or 2
	MessageID   string
	ChannelID   string
}

// Wordle session
type wordleSession struct {
	UserID    string
	Word      string
	Guesses   []string
	Results   [][]int
	MessageID string
	ChannelID string
}

// Global state
var (
	botStartedAt   = time.Now()
	sessionMu      sync.Mutex
	sessions       = map[string]*mazeSession{}
	triviaMu       sync.Mutex
	triviaSessions = map[string]triviaSession{}
	tradeMu        sync.Mutex
	tradeSessions  = map[string]*tradeSession{}
	tagSessions    = map[string]*tagSession{}
	modMu          sync.Mutex
	spamTrackers   = map[string]*spamTracker{} // key: guildID_userID
	gameMu         sync.Mutex
	bjSessions     = map[string]*bjSession{}
	pokerSessions  = map[string]*pokerSession{}
	fishSessions   = map[string]*fishSession{}
	snapSessions   = map[string]*snapSession{}
	g2048Sessions  = map[string]*g2048Session{}
	hlSessions     = map[string]*hlSession{}
	pollMu         sync.Mutex
	pollSessions   = map[string]*pollSession{}
	tttMu          sync.Mutex
	tttSessions    = map[string]*tttSession{}
	c4Mu           sync.Mutex
	c4Sessions     = map[string]*c4Session{}
	wordleMu       sync.Mutex
	wordleSessions = map[string]*wordleSession{}
	robCooldowns   = map[string]int64{} // userID → unix timestamp
	workCooldowns  = map[string]int64{} // userID → unix timestamp
	sessionAge     = map[string]int64{} // sessionKey → unix creation timestamp
	sessionAgeMu   sync.Mutex
)

// Command definitions — all commands are grouped into 7 top-level slash commands.
func allCommands() []*discordgo.ApplicationCommand {
	minv := func(v float64) *float64 { return &v }

	pollOpts := []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "The poll question", Required: true},
		{Type: discordgo.ApplicationCommandOptionString, Name: "option1", Description: "First option", Required: true},
		{Type: discordgo.ApplicationCommandOptionString, Name: "option2", Description: "Second option", Required: true},
	}
	for idx := 3; idx <= 10; idx++ {
		pollOpts = append(pollOpts, &discordgo.ApplicationCommandOption{
			Type: discordgo.ApplicationCommandOptionString, Name: fmt.Sprintf("option%d", idx),
			Description: fmt.Sprintf("Option %d", idx), Required: false,
		})
	}

	return []*discordgo.ApplicationCommand{
		// ── /games ────────────────────────────────────────────────────────
		{Name: "games", Description: "Play interactive games", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "2048", Description: "Play the 2048 sliding tile puzzle"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "highlow", Description: "Guess higher or lower for coins", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-500)", Required: true, MinValue: minv(1), MaxValue: 500},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "maze", Description: "Navigate interactive maze levels", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "play", Description: "Start a maze", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Choose difficulty level", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Easy - Beginner's Path", Value: 0}, {Name: "Medium - Spike Corridor", Value: 1}, {Name: "Hard - Monster Maze", Value: 2},
					}},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View maze leaderboard", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Filter by level", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "All Levels", Value: -1}, {Name: "Easy - Beginner's Path", Value: 0}, {Name: "Medium - Spike Corridor", Value: 1}, {Name: "Hard - Monster Maze", Value: 2},
					}},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "war", Description: "Flip cards — higher card wins", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-1000)", Required: true, MinValue: minv(1), MaxValue: 1000},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "snap", Description: "Play Snap — press it when cards match!", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-500)", Required: true, MinValue: minv(1), MaxValue: 500},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "go-fish", Description: "Play Go Fish against the bot"},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "tag", Description: "Strategic tag game", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "challenge", Description: "Challenge someone to tag", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "Who to challenge", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View tag leaderboard"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View tag stats", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to check", Required: false},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "tictactoe", Description: "Play Tic-Tac-Toe against another user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "Who to challenge", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "connect4", Description: "Play Connect Four against another user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "Who to challenge", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "wordle", Description: "Play a Wordle word-guessing game"},
		}},
		// ── /casino ───────────────────────────────────────────────────────
		{Name: "casino", Description: "Casino and gambling games", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "blackjack", Description: "Play blackjack vs the dealer", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-10000)", Required: true, MinValue: minv(1), MaxValue: 10000},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "slots", Description: "Spin the slot machine", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-1000)", Required: true, MinValue: minv(1), MaxValue: 1000},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "roulette", Description: "Bet on the roulette wheel", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-5000)", Required: true, MinValue: minv(1), MaxValue: 5000},
				{Type: discordgo.ApplicationCommandOptionString, Name: "choice", Description: "red, black, green, or a number 0-36", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "russian-roulette", Description: "Test your luck in Russian roulette", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "chambers", Description: "Number of chambers (2-6, default 6)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "poker", Description: "Play 5-card video poker", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "bet", Description: "Amount to bet (1-5000)", Required: true, MinValue: minv(1), MaxValue: 5000},
			}},
		}},
		// ── /fun ──────────────────────────────────────────────────────────
		{Name: "fun", Description: "Fun commands and social games", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "8ball", Description: "Ask the magic 8-ball a question", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "Your question", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "battle", Description: "Challenge someone to battle!", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "opponent", Description: "The user to battle", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "bossraid", Description: "Server-wide boss raid", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "status", Description: "Check the current boss status"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "attack", Description: "Attack the boss!"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "spawn", Description: "Spawn a new boss (Admin only)"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View raid damage leaderboard"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "coinflip", Description: "Flip a coin"},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "creative", Description: "Creative and text fun commands", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ascii-art", Description: "Convert text to ASCII block art", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text to convert (max 10 chars, A-Z 0-9)", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "countdown", Description: "Countdown to a date", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "event", Description: "Event name", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "date", Description: "Target date (YYYY-MM-DD)", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "emoji-mix", Description: "Combine two emojis humorously", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "emoji1", Description: "First emoji", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "emoji2", Description: "Second emoji", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "fake-tweet", Description: "Generate a fake tweet card", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Who posted the tweet", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Tweet content", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "fortune", Description: "Get a random fortune cookie message"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "daily", Description: "Daily rewards system", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "claim", Description: "Claim your daily reward"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your reward stats"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the coins leaderboard"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "dice", Description: "Roll a dice", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "sides", Description: "Dice sides", Required: false},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "count", Description: "How many dice", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "flip-text", Description: "Flip text upside down", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text to flip", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "joke", Description: "Get a random joke", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Joke category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Any", Value: "Any"}, {Name: "Programming", Value: "Programming"}, {Name: "Miscellaneous", Value: "Misc"},
					{Name: "Pun", Value: "Pun"}, {Name: "Spooky", Value: "Spooky"}, {Name: "Christmas", Value: "Christmas"},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "loot", Description: "Open loot chests!", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "open", Description: "Open a loot chest"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "inventory", Description: "View your inventory"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your loot statistics"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "meme", Description: "Get a random meme from Reddit"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "mock-text", Description: "CoNvErT tExT tO aLtErNaTiNg CaPs", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "party", Description: "Party games and social prompts", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "hotseat", Description: "Put a random server member in the hotseat"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "this-or-that", Description: "Quick preference poll — pick one!"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "truth-or-dare", Description: "Get a random truth or dare prompt"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "vibecheck", Description: "Check your current vibe rating", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Check someone else's vibe", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "would-you-rather", Description: "Get a would you rather question"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "quest", Description: "Complete quests for XP!", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "get", Description: "Get a new quest"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "complete", Description: "Mark your current quest as complete"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the quest leaderboard"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "View your quest statistics"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "quote", Description: "Manage iconic server quotes", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "add", Description: "Add a quote", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "The quote text", Required: true},
					{Type: discordgo.ApplicationCommandOptionUser, Name: "author", Description: "Who said it", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "random", Description: "Get a random quote"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "List recent quotes"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remove", Description: "Remove a quote by ID (moderators only)", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "id", Description: "Quote ID to remove", Required: true},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "random-choice", Description: "Pick a random option from a list", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "options", Description: "Comma separated options", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "random-number", Description: "Generate a random number", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "min", Description: "Minimum", Required: false},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "max", Description: "Maximum", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "rate", Description: "Rate something out of 10", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "thing", Description: "Thing to rate", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "reverse-text", Description: "Reverse text backwards", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "roll", Description: "Roll custom dice with dramatic flair", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "dice", Description: "Example: 2d6", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "rps", Description: "Play rock, paper, scissors", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "choice", Description: "rock, paper, or scissors", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "rock", Value: "rock"}, {Name: "paper", Value: "paper"}, {Name: "scissors", Value: "scissors"},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ship", Description: "Calculate compatibility between two users", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user1", Description: "First user", Required: true},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user2", Description: "Second user", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "social", Description: "Social fun commands", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "compliment", Description: "Give someone a nice compliment", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Who to compliment", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "roast", Description: "Funny roast generator", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Who to roast", Required: true},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "summon", Description: "Dramatically summon someone", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The person to summon", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Why are you summoning them?", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "trivia", Description: "Trivia questions and leaderboard", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "play", Description: "Answer trivia questions", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Question category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "General Knowledge", Value: "9"}, {Name: "Science & Nature", Value: "17"}, {Name: "Computers", Value: "18"},
						{Name: "Mathematics", Value: "19"}, {Name: "Sports", Value: "21"}, {Name: "Geography", Value: "22"},
						{Name: "History", Value: "23"}, {Name: "Animals", Value: "27"},
					}},
					{Type: discordgo.ApplicationCommandOptionString, Name: "difficulty", Description: "Question difficulty", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Easy", Value: "easy"}, {Name: "Medium", Value: "medium"}, {Name: "Hard", Value: "hard"},
					}},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the trivia leaderboard"},
			}},
		}},
		// ── /math ─────────────────────────────────────────────────────────
		{Name: "math", Description: "Math and unit conversion tools", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "calc", Description: "Perform mathematical calculations", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "expression", Description: "Expression", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "convert", Description: "Convert between units", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "value", Description: "Value to convert", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "from", Description: "Source unit (f, c, k, ft, m, mi, km, lb, kg, etc.)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "to", Description: "Target unit", Required: true},
			}},
		}},
		// ── /util ─────────────────────────────────────────────────────────
		{Name: "util", Description: "Utility and information commands", Options: func() []*discordgo.ApplicationCommandOption {
			opts := []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "afk", Description: "Set your AFK status", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Why are you AFK?", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "avatar", Description: "Display a user's avatar", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "channel-info", Description: "Display detailed information about a channel", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "clear-my-data", Description: "Remove all your data from the bot", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: "confirm", Description: "Confirm deletion", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "color", Description: "Preview a hex color with RGB values", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "hex", Description: "Hex color code (e.g. FF5733 or #FF5733)", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "define", Description: "Look up a word in the dictionary", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "word", Description: "Word to define", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "embed-builder", Description: "Create a custom embed", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "title", Description: "Embed title", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "description", Description: "Embed description", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "color", Description: "Hex color (default: 5865F2)", Required: false},
					{Type: discordgo.ApplicationCommandOptionString, Name: "footer", Description: "Footer text", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "github", Description: "Show GitHub repo info", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "repo", Description: "Repository in owner/repo format", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "giveaway", Description: "Create a giveaway with reactions", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "prize", Description: "Prize to give away", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "duration", Description: "Duration in minutes (1-1440)", Required: true},
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "winners", Description: "Number of winners (default: 1)", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "help", Description: "Display all available commands", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Filter category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Games", Value: "games"}, {Name: "Casino", Value: "casino"}, {Name: "Fun", Value: "fun"},
						{Name: "Math", Value: "math"}, {Name: "Utility", Value: "util"}, {Name: "Economy", Value: "economy"},
						{Name: "Moderation", Value: "mod"}, {Name: "All Commands", Value: "all"},
					}},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ping", Description: "Check bot latency and response time"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "poll", Description: "Create a poll with up to 10 options", Options: pollOpts},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remind", Description: "Set a reminder", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "time", Description: "Time like 30m, 2h", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Reminder message", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "reminders", Description: "Manage your active reminders", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "View all your active reminders"},
					{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "clear", Description: "Delete all your reminders"},
					{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remove", Description: "Remove a specific reminder", Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "number", Description: "Reminder number", Required: true},
					}},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "role-info", Description: "Display detailed information about a role", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "serverinfo", Description: "Display information about this server"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "snipe", Description: "Recover the last deleted message in this channel"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stats", Description: "Display bot statistics and metrics"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "sticky", Description: "Create a sticky message embed", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Message to pin as sticky", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "timer", Description: "Set a countdown timer", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "seconds", Description: "Duration in seconds (max 300)", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "translate", Description: "Translate text to another language", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "text", Description: "Text to translate", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "to", Description: "Target language code (es, fr, de, ...)", Required: true},
					{Type: discordgo.ApplicationCommandOptionString, Name: "from", Description: "Source language (default auto)", Required: false},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "weather", Description: "Look up current weather for a city", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "city", Description: "City name", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "userinfo", Description: "Display detailed information about a user", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: false},
				}},
			}
			return opts
		}()},
		// ── /economy ──────────────────────────────────────────────────────
		{Name: "economy", Description: "Economy, shop, and trading system", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "balance", Description: "View your coin balance and stats"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "rob", Description: "Attempt to steal coins from another user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to rob", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "gift", Description: "Gift coins to another user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to gift coins to", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount of coins to gift", Required: true, MinValue: minv(1)},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the top 10 richest users"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "work", Description: "Work a job to earn coins (30 min cooldown)"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "lottery", Description: "Buy lottery tickets for a chance to win big", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "tickets", Description: "Number of tickets to buy (1-10, 10 coins each)", Required: true, MinValue: minv(1), MaxValue: 10},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "shop", Description: "Browse and buy items from the shop", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "Browse shop items", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Filter by category", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "All", Value: "all"}, {Name: "Boosts", Value: "boost"}, {Name: "Collectibles", Value: "collectible"}, {Name: "Consumables", Value: "consumable"},
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
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "inventory", Description: "View and manage your inventory", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "view", Description: "View your inventory"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "use", Description: "Use/activate an item", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item to use", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "info", Description: "View item details", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "item_id", Description: "Item to inspect", Required: true},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "trade", Description: "Trade coins and items with other users", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "offer", Description: "Start a trade with someone", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to trade with", Required: true},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "admin", Description: "Manage server economy (Admin only)", Options: []*discordgo.ApplicationCommandOption{
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
		}},
		// ── /mod ──────────────────────────────────────────────────────────
		{Name: "mod", Description: "Moderation tools", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "announce", Description: "Send an announcement (requires Manage Messages)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel to announce in", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Announcement message", Required: true},
				{Type: discordgo.ApplicationCommandOptionBoolean, Name: "ping", Description: "Ping @everyone?", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "automod", Description: "Configure auto-moderation", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "setup", Description: "Configure automod settings"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "toggle", Description: "Toggle specific automod rules", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "rule", Description: "Rule to toggle", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Spam Detection", Value: "spam"}, {Name: "Invite Links", Value: "invites"},
						{Name: "All Caps", Value: "caps"}, {Name: "Emoji Spam", Value: "emoji"}, {Name: "External Links", Value: "links"},
					}},
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: "enabled", Description: "Enable or disable", Required: true},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ban", Description: "Ban a member from the server", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to ban", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for ban", Required: false},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "delete_days", Description: "Delete message history (0-7 days)", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "None", Value: 0}, {Name: "1 day", Value: 1}, {Name: "3 days", Value: 3}, {Name: "7 days", Value: 7},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "clearwarnings", Description: "Clear warnings for a member", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to clear warnings", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "warning_id", Description: "Specific warning ID to clear (optional)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "kick", Description: "Kick a member from the server", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to kick", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for kick", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "lock", Description: "Lock a channel", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel to lock (default: current)", Required: false},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for lock", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "modlog", Description: "Manage moderation logs", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "setup", Description: "Configure mod log channel", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel for mod logs", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "view", Description: "View moderation history", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Filter by user", Required: false},
					{Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "Filter by action type", Required: false, Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Kick", Value: "kick"}, {Name: "Ban", Value: "ban"}, {Name: "Warn", Value: "warn"},
						{Name: "Timeout", Value: "timeout"}, {Name: "Purge", Value: "purge"},
					}},
				}},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "nick", Description: "Change a user's nickname", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to rename", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "nickname", Description: "New nickname (leave empty to reset)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "purge", Description: "Bulk delete messages", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Number of messages (2-100)", Required: true},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Only delete from this user (optional)", Required: false},
				{Type: discordgo.ApplicationCommandOptionString, Name: "contains", Description: "Only delete messages containing text (optional)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: "role", Description: "Manage server roles", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "add", Description: "Add a role to a user", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Target user", Required: true},
					{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role to add", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "remove", Description: "Remove a role from a user", Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Target user", Required: true},
					{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role to remove", Required: true},
				}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "List all server roles"},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "slowmode", Description: "Set channel slowmode", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "seconds", Description: "Delay in seconds (0 to disable, max 21600)", Required: true},
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel (default: current)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "timeout", Description: "Timeout a member", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to timeout", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "duration", Description: "Duration", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "5 minutes", Value: "5m"}, {Name: "10 minutes", Value: "10m"}, {Name: "30 minutes", Value: "30m"},
					{Name: "1 hour", Value: "1h"}, {Name: "6 hours", Value: "6h"}, {Name: "12 hours", Value: "12h"},
					{Name: "1 day", Value: "1d"}, {Name: "3 days", Value: "3d"}, {Name: "7 days", Value: "7d"},
				}},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for timeout", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "unban", Description: "Unban a member from the server", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "user_id", Description: "User ID to unban", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for unban", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "unlock", Description: "Unlock a channel", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel to unlock (default: current)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "warn", Description: "Warn a member", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to warn", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for warning", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "warnings", Description: "View warnings for a member", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to check", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "softban", Description: "Ban + immediate unban to purge messages", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to softban", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for softban", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "history", Description: "Full moderation history for a user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to view history for", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "case", Description: "View a specific mod case by ID", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "id", Description: "Case ID to look up", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "reason", Description: "Update the reason for a mod case", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "case_id", Description: "Case ID to update", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "New reason", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "report", Description: "Report a user to the moderators", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to report", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for report", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "reports", Description: "View user reports (Manage Messages)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Filter by reported user (optional)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "massban", Description: "Ban multiple users at once", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user1", Description: "First user to ban", Required: true},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user2", Description: "Second user to ban", Required: true},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user3", Description: "Third user to ban", Required: false},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user4", Description: "Fourth user to ban", Required: false},
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user5", Description: "Fifth user to ban", Required: false},
				{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for mass ban", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "modnote", Description: "Add a private moderation note about a user", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to add note for", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "note", Description: "Note content", Required: true},
			}},
		}},
		// ── /level ───────────────────────────────────────────────────────
		{Name: "level", Description: "Leveling and XP system", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "view", Description: "View your level card", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to view (default: yourself)", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "leaderboard", Description: "View the XP leaderboard"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "rewards", Description: "View role rewards per level"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-reward", Description: "Set a role reward for a level (Admin)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Level to set reward for", Required: true, MinValue: minv(1)},
				{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role to award", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-channel", Description: "Set level-up announcement channel (Admin)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel for level-up announcements", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "toggle", Description: "Enable or disable the leveling system (Admin)"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "reset", Description: "Reset a user's XP (Admin)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User to reset", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-multiplier", Description: "Set XP multiplier (Admin)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionNumber, Name: "multiplier", Description: "XP multiplier (0.5-5.0)", Required: true, MinValue: minv(0.5), MaxValue: 5.0},
			}},
		}},
		// ── /welcome ─────────────────────────────────────────────────────
		{Name: "welcome", Description: "Configure welcome and leave messages", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "setup", Description: "Set the welcome channel and message (Manage Server)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel for welcome messages", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Custom welcome message ({user}, {username}, {server}, {membercount})", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-leave", Description: "Set the leave channel and message (Manage Server)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Channel for leave messages", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Custom leave message ({user}, {username}, {server}, {membercount})", Required: false},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-role", Description: "Set auto-role for new members (Manage Roles)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role to assign on join", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "toggle", Description: "Enable or disable welcome/leave messages (Manage Server)"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "test", Description: "Preview the welcome message for yourself"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "set-image", Description: "Set a custom banner image for welcome messages (Manage Server)", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "url", Description: "Banner image URL", Required: true},
			}},
		}},
		// ── /music ──────────────────────────────────────────────────────────
		{Name: "music", Description: "Music player and queue management", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "play", Description: "Play a song or add it to the queue", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "query", Description: "Song name or URL", Required: true},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "pause", Description: "Pause the current song"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "resume", Description: "Resume playback"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "skip", Description: "Skip to the next song in the queue"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "stop", Description: "Stop playback, clear queue, and disconnect"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "queue", Description: "Show the current music queue"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "nowplaying", Description: "Show the currently playing song"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "volume", Description: "Set the playback volume", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "level", Description: "Volume level (1-100)", Required: true, MinValue: minv(1), MaxValue: 100},
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "shuffle", Description: "Shuffle the upcoming songs in the queue"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "loop", Description: "Set loop mode", Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "mode", Description: "Loop mode", Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Off", Value: "off"},
					{Name: "Song", Value: "song"},
					{Name: "Queue", Value: "queue"},
				}},
			}},
		}},
	}
}

