# Discorbo 🤖

A comprehensive Discord bot with 105+ slash commands for moderation, RPG, casino, economy, music, leveling, and community features. Built with Go and discordgo. Visit [the website](https://tinyurl.com/discorbo).

## Features

### 🎮 Fun & Games (30+ commands)
- **Creative:** `/fun creative ascii-art`, `countdown`, `emoji-mix`, `fake-tweet`, `fortune`
- **Social:** `/fun social roast`, `compliment`
- **Party:** `/fun party truth-or-dare`, `this-or-that`, `would-you-rather`, `hotseat`, `vibecheck`
- **Classic:** `/fun coinflip`, `dice`, `8ball`, `rps`, `joke`, `meme`, `trivia`, `roll`, `ship`, `summon`

### 🎲 Games (14 commands)
- `/games 2048` - Sliding tile puzzle with Unicode board
- `/games tictactoe` - Multiplayer tic-tac-toe
- `/games connect4` - Connect Four with emoji board
- `/games wordle` - 6-guess word game
- `/games maze` - Navigate interactive mazes
- `/games highlow` - Card guessing game
- `/games war`, `snap`, `go-fish`, `tag`

### 🎰 Casino (5 commands)
- `/casino blackjack` - Classic 21 with compact card display
- `/casino poker` - 5-card video poker
- `/casino slots` - Weighted slot machine
- `/casino roulette` - European roulette
- `/casino russian-roulette` - Risk/reward game

### 💰 Economy (16 commands)
- `/economy balance`, `shop`, `inventory`, `trade`
- `/economy daily` - Daily rewards with streaks
- `/economy rob`, `gift`, `leaderboard`, `work`, `lottery`
- `/economy admin` - Admin tools (grant, take, transactions)

### 🛡️ Moderation (23 commands)
- `/mod kick`, `ban`, `unban`, `timeout`, `warn`, `warnings`
- `/mod purge`, `lock`, `unlock`, `slowmode`
- `/mod automod` - Auto-moderation setup
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
- `/level` - View your level card with XP bar
- `/level leaderboard`, `rewards`
- Admin: `set-rewards`, `set-channel`, `toggle`, `reset`, `set-multiplier`

### 👋 Welcome (6 commands)
- `/welcome setup`, `set-leave`, `set-role`
- `/welcome toggle`, `test`, `set-image`

# Note: Do not publish Discorbo as your app, just invite the current one to your server [Discorbo](https://discord.com/oauth2/authorize?client_id=1472679226330059023&permissions=277025770560&scope=bot%20applications.commands)

## Prerequisites
(I don't recommend this unless you want to copy my work..., just head to [my website](

- Node.js v16.9.0 or higher
- A Discord bot token ([Create one here](https://discord.com/developers/applications))
- npm or yarn package manager

## Installation

1. **Clone the repository**
   ```bash
   git clone <your-repo-url>
   cd Discorbo
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Configure environment variables**
   - Copy `.env.example` to `.env`
   - Fill in your bot credentials:
   ```env
   DISCORD_TOKEN=your_bot_token_here
   CLIENT_ID=your_application_id_here
   GUILD_ID=your_dev_server_id_here  # Optional, for faster command updates
   OWNER_ID=your_discord_user_id_here  # Optional
   ```

4. **Get your Discord Bot Token**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create a new application or select existing
   - Go to "Bot" section and click "Add Bot"
   - Copy the token and paste it in `.env`
   - Enable these intents in the Bot settings:
     - Server Members Intent
     - Message Content Intent

5. **Invite the bot to your server**
   - In Developer Portal, go to "OAuth2" → "URL Generator"
   - Select scopes: `bot`, `applications.commands`
   - Select permissions: At minimum, these permissions:
     - Send Messages
     - Embed Links
     - Add Reactions
     - Read Message History
     - Use Slash Commands
   - Copy the generated URL and open it in browser
   - Select your server and authorize

## Usage

1. **Deploy slash commands to Discord**
   ```bash
   npm run deploy
   ```
   This registers your commands with Discord. Run this whenever you add or modify commands.

2. **Start the bot**
   ```bash
   npm start
   ```

3. **Development mode** (auto-restart on file changes)
   ```bash
   npm run dev
   ```

## Project Structure

```
Discorbo/
├── src/
│   ├── index.js              # Bot entry point
│   ├── config.js             # Configuration
│   ├── deploy-commands.js    # Command registration
│   │
│   ├── commands/             # Slash commands
│   │   ├── fun/              # Fun & game commands
│   │   └── utility/          # Utility commands
│   │
│   ├── events/               # Event handlers
│   │   ├── ready.js          # Bot startup
│   │   ├── interactionCreate.js  # Command handler
│   │   └── messageCreate.js  # AFK system
│   │
│   ├── utils/                # Utilities
│   │   ├── logger.js         # Logging
│   │   ├── embedBuilder.js   # Embed helpers
│   │   ├── errorHandler.js   # Error management
│   │   ├── cooldownManager.js # Anti-spam
│   │   ├── dataManager.js    # JSON storage
│   │   └── reminderChecker.js # Reminder system
│   │
│   └── data/                 # JSON data files
│       ├── trivia-scores.json
│       ├── reminders.json
│       ├── afk-users.json
│       └── command-usage.json
│
├── .env                      # Environment config (gitignored)
├── .env.example              # Environment template
├── package.json              # Dependencies
└── README.md                 # This file
```

## Data Persistence

The bot uses JSON files for data storage:
- **trivia-scores.json** - Trivia game scores and statistics
- **reminders.json** - Active user reminders
- **afk-users.json** - AFK status for users
- **command-usage.json** - Command usage statistics

These files are created automatically in `src/data/` when needed.

## Troubleshooting

### Bot doesn't come online
- Verify your `DISCORD_TOKEN` is correct in `.env`
- Check that required intents are enabled in Discord Developer Portal
- Ensure Node.js version is 16.9.0 or higher

### Commands don't appear
- Run `npm run deploy` to register commands
- If using `GUILD_ID`, commands update instantly (recommended for development)
- Without `GUILD_ID`, global commands can take up to 1 hour to update

### Reminders not working
- Check that the bot has permission to DM users
- Verify the bot is online (reminder checker runs every 30 seconds)

### API commands failing (joke, meme, trivia, translate)
- Check your internet connection
- Some APIs may have rate limits or be temporarily down
- The bot will display user-friendly error messages

## Configuration

Edit `src/config.js` to customize:
- Bot presence/status
- Default cooldowns
- API endpoints
- Feature flags
- Limits (poll options, reminders, etc.)

## Adding New Commands

1. Create a new file in `src/commands/fun/` or `src/commands/utility/`
2. Use this template:

```javascript
const { SlashCommandBuilder } = require('discord.js');
const { infoEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('commandname')
    .setDescription('Command description'),

  category: 'fun', // or 'utility'
  cooldown: 5, // seconds

  async execute(interaction) {
    const embed = infoEmbed('Title', 'Description');
    await interaction.reply({ embeds: [embed] });
  }
};
```

3. Run `npm run deploy` to register the command
4. Restart the bot

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - feel free to use this bot for your own projects!

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review bot console logs for error messages
3. Ensure all prerequisites are met
4. Check Discord API status at https://discordstatus.com

## Acknowledgments

- Built with [discord.js](https://discord.js.org/)
- Jokes from [JokeAPI](https://jokeapi.dev/)
- Memes from Reddit
- Trivia from [Open Trivia DB](https://opentdb.com/)
- Translation from [LibreTranslate](https://libretranslate.com/)

---

Made with ❤️ using discord.js
