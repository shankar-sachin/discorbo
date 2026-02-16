# Discorbo - Go Runtime

This is the Go implementation of the Discorbo Discord bot, providing all bot commands in a single runtime.

## Structure

The Go codebase is organized into focused, single-purpose files:

### Core Files
- **main.go** - Application entry point, bot initialization, and command routing
- **types.go** - All type definitions and command declarations
- **handlers.go** - Event handlers for components and messages

### Command Files
- **cmd_fun.go** - Fun commands (simple, interactive, API-based, and game commands)
- **cmd_fun_games.go** - Game commands with persistent state (battle, daily, bossraid, quest, loot, quote)
- **cmd_maze.go** - Maze game commands and logic
- **cmd_utility.go** - Utility commands (info, tools, and data management)

### Helper Files
- **helpers.go** - Shared helper functions (Discord interactions, string manipulation, dice rolling, math parsing)
- **data.go** - Data persistence layer (JSON file management, reminders system)

## Environment Variables

Uses the same `.env` values as the JS bot:
- `DISCORD_TOKEN` - Your Discord bot token
- `CLIENT_ID` - Your Discord application client ID
- `GUILD_ID` - (Optional) Guild ID for faster command updates during development

## Running the Bot

### Prerequisites
- Go 1.22 or higher
- Discord bot token configured in `.env`

### Install Dependencies
```bash
cd go
go mod download
```

### Run the Bot
```bash
# From the go directory
go run .

# Or build and run
go build -o discorbo
./discorbo  # or discorbo.exe on Windows
```

## Commands Implemented

All Discorbo commands are implemented in the Go runtime:

### Fun & Games
- 8ball, battle, bossraid, coinflip, daily, dice, flip-text, hotseat
- joke, loot, maze, maze-leaderboard, meme, mock-text, quest, quote
- random-choice, random-number, rate, reverse-text, roll, rps, ship
- summon, trivia, trivia-leaderboard, vibecheck, would-you-rather

### Utility
- afk, avatar, calc, channel-info, clear-my-data, help, ping, poll
- remind, reminders, role-info, serverinfo, stats, timer, translate, userinfo

## Data Storage

- Data files are stored in `../src/data/` (shared with JS bot)
- JSON-based persistence for simplicity
- Automatic file creation and schema initialization
- Thread-safe operations with mutexes

## Development Notes

- **Organized structure**: Each file has a clear, single responsibility
- **Type safety**: Strong typing throughout the codebase
- **Concurrency**: Safe concurrent access to shared resources
- **Error handling**: Graceful error handling with user-friendly messages
- **Hot reload**: Supports guild-scoped commands for fast development iterations

## Migration Status

✅ All commands migrated to Go
✅ Shared data storage with JS bot
✅ Complete feature parity

---

**Last Updated:** 2026-02-16
**Go Version:** 1.22+
**Discord.js Version (for reference):** 14.14.1
