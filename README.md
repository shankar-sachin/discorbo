# Discorbo 🤖

A comprehensive Discord bot with **105+ slash commands** for moderation, RPG, casino, economy, music, leveling, and community features. Built with **Go** and [discordgo](https://github.com/bwmarrin/discordgo).

[![Version](https://img.shields.io/badge/version-2.2.0-blue)]()
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)]()
[![License](https://img.shields.io/badge/license-MIT-green)]()

🌐 **[Website](https://tinyurl.com/discorbo)** · **[Invite Discorbo](https://discord.com/oauth2/authorize?client_id=1472679226330059023&permissions=277025770560&scope=bot%20applications.commands)**

> **Note:** Please don't publish Discorbo as your own app — just invite it to your server using the link above!

---

## Features

### 🎮 Fun & Games (30+ commands)
- **Creative:** `/fun creative ascii-art`, `countdown`, `emoji-mix`, `fake-tweet`, `fortune`
- **Social:** `/fun social roast`, `compliment`
- **Party:** `/fun party truth-or-dare`, `this-or-that`, `would-you-rather`, `hotseat`, `vibecheck`
- **Classic:** `/fun coinflip`, `dice`, `8ball`, `rps`, `joke`, `meme`, `trivia`, `roll`, `ship`, `summon`

### 🎲 Games (14 commands)
- `/games 2048` — Sliding tile puzzle with Unicode board
- `/games tictactoe` — Multiplayer tic-tac-toe
- `/games connect4` — Connect Four with emoji board
- `/games wordle` — 6-guess word game
- `/games maze` — Navigate interactive mazes
- `/games highlow` — Card guessing game
- `/games war`, `snap`, `go-fish`, `tag`

### 🎰 Casino (5 commands)
- `/casino blackjack` — Classic 21 with compact card display
- `/casino poker` — 5-card video poker
- `/casino slots` — Weighted slot machine
- `/casino roulette` — European roulette
- `/casino russian-roulette` — Risk/reward game

### 💰 Economy (16 commands)
- `/economy balance`, `shop`, `inventory`, `trade`
- `/economy daily` — Daily rewards with streaks
- `/economy rob`, `gift`, `leaderboard`, `work`, `lottery`
- `/economy admin` — Admin tools (grant, take, transactions)

### 🛡️ Moderation (23 commands)
- `/mod kick`, `ban`, `unban`, `timeout`, `warn`, `warnings`
- `/mod purge`, `lock`, `unlock`, `slowmode`
- `/mod automod` — Auto-moderation setup
- `/mod softban`, `massban`, `history`, `case`, `reason`
- `/mod report`, `reports`, `modnote`, `modlog`

### 🛠️ Utility (23 commands)
- `/util help`, `ping`, `avatar`, `userinfo`, `serverinfo`
- `/util poll`, `remind`, `translate`, `calc`, `timer`
- `/util color`, `define`, `github`, `snipe`, `weather`
- `/util embed-builder`, `giveaway`, `sticky`

### 🎵 Music (10 commands)
- `/music play`, `pause`, `resume`, `skip`, `stop`
- `/music queue`, `nowplaying`, `volume`, `shuffle`, `loop`

### ⭐ Leveling (8 commands)
- `/level` — View your level card with XP bar
- `/level leaderboard`, `rewards`
- Admin: `set-rewards`, `set-channel`, `toggle`, `reset`, `set-multiplier`

### 👋 Welcome (6 commands)
- `/welcome setup`, `set-leave`, `set-role`
- `/welcome toggle`, `test`, `set-image`

---

## Prerequisites

- **Go 1.21+** ([Download](https://go.dev/dl/))
- A Discord bot token ([Create one](https://discord.com/developers/applications))

## Installation

1. **Clone the repository**
   ```bash
   git clone <your-repo-url>
   cd Discorbo
   ```

2. **Configure environment variables**
   - Copy `.env.example` to `.env`
   - Fill in your bot credentials:
   ```env
   DISCORD_TOKEN=your_bot_token_here
   CLIENT_ID=your_application_id_here
   GUILD_ID=your_dev_server_id_here  # Optional, for faster command updates
   OWNER_ID=your_discord_user_id_here  # Optional
   ```

3. **Get your Discord Bot Token**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create a new application or select existing
   - Go to "Bot" section and click "Add Bot"
   - Copy the token and paste it in `.env`
   - Enable these intents in the Bot settings:
     - Server Members Intent
     - Message Content Intent

4. **Invite the bot to your server**
   - In Developer Portal, go to "OAuth2" → "URL Generator"
   - Select scopes: `bot`, `applications.commands`
   - Select permissions: Send Messages, Embed Links, Add Reactions, Read Message History, Use Slash Commands
   - Copy the generated URL, open it in browser, and authorize

## Build & Run

```bash
# Build the bot
cd go && go build -o discorbo.exe

# Run the bot (auto-registers slash commands on startup)
.\discorbo.exe
```

Or using npm scripts from the project root:
```bash
npm run build   # Builds go/discorbo.exe
npm start       # Runs go/discorbo.exe
```

Slash commands are **automatically registered** when the bot starts — no separate deploy step needed.

## Project Structure

```
Discorbo/
├── go/                         # Go source (runtime)
│   ├── main.go                 # Entry point, command routing, graceful shutdown
│   ├── types.go                # All struct definitions, allCommands()
│   ├── handlers.go             # Button/component interactions, messageCreate
│   ├── helpers.go              # Embed builders, math parser, visual helpers
│   ├── rendering.go            # Unicode card/board rendering
│   ├── data.go                 # Thread-safe JSON I/O with in-memory cache
│   ├── cmd_fun.go              # Core fun commands (~20)
│   ├── cmd_fun_new.go          # Creative/social/party commands (10)
│   ├── cmd_fun_games.go        # Tag, boss raids, quest, loot
│   ├── cmd_games_puzzle.go     # 2048, high-low, tictactoe, connect4, wordle
│   ├── cmd_games_casino.go     # Blackjack, slots, roulette, war
│   ├── cmd_games_cards.go      # Poker, go-fish, snap
│   ├── cmd_economy.go          # Economy (16 commands)
│   ├── cmd_moderation.go       # Moderation (23 commands)
│   ├── cmd_utility.go          # Utility (23 commands)
│   ├── cmd_leveling.go         # Leveling/XP system (8 commands)
│   ├── cmd_welcome.go          # Welcome/leave system (6 commands)
│   ├── cmd_music.go            # Music queue management (10 commands)
│   ├── cmd_maze.go             # Maze generation and navigation
│   ├── go.mod / go.sum         # Go module dependencies
│   └── build.ps1 / run.ps1    # Build and run scripts
│
├── src/data/                   # JSON data files (created automatically)
│   ├── economy-users.json      # All player balances, inventories, streaks
│   ├── trivia-scores.json      # Trivia scores
│   ├── reminders.json          # Active reminders
│   ├── afk-users.json          # AFK statuses
│   ├── command-usage.json      # Usage statistics
│   └── ...                     # Guild configs, mod actions, etc.
│
├── .env                        # Environment config (gitignored)
├── .env.example                # Environment template
├── package.json                # npm scripts for build/start
└── README.md                   # This file
```

## Data Persistence

The bot uses JSON files with an **in-memory cache** and automatic 30-second flush for performance. Data files are stored in `src/data/` and created automatically when needed.

Key data files:
- **economy-users.json** — All player balances, inventories, daily streaks, boosts
- **trivia-scores.json** — Trivia scores and statistics
- **reminders.json** — Active user reminders
- **afk-users.json** — AFK status for users
- **guild-config.json** — Per-server settings
- **mod-actions.json** — Moderation action log

## Troubleshooting

### Bot doesn't come online
- Verify your `DISCORD_TOKEN` is correct in `.env`
- Check that required intents are enabled in Discord Developer Portal
- Ensure Go 1.21+ is installed (`go version`)

### Commands don't appear
- Commands register automatically on startup
- If using `GUILD_ID`, commands update instantly
- Without `GUILD_ID`, global commands can take up to 1 hour to propagate

### API commands failing (joke, meme, trivia, translate)
- Check your internet connection
- Some APIs may have rate limits or be temporarily down
- The bot displays user-friendly error embeds

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License — feel free to use this bot for your own projects!

## Acknowledgments

- Built with [discordgo](https://github.com/bwmarrin/discordgo)
- Jokes from [JokeAPI](https://jokeapi.dev/)
- Memes from Reddit
- Trivia from [Open Trivia DB](https://opentdb.com/)

---

Made with ❤️ in Go
