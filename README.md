# Discorbo 🤖

A comprehensive Discord bot with 32+ slash commands for fun, games, and utility. Built with Node.js and discord.js v14. Visit [the website](https://tinyurl.com/discorbo).

## Features

### 🎮 Fun & Games (16 commands)
- `/8ball` - Magic 8-ball predictions
- `/coinflip` - Heads or tails
- `/dice` - Roll dice (customizable sides)
- `/rps` - Rock, Paper, Scissors game
- `/joke` - Random jokes from JokeAPI
- `/meme` - Fetch memes from Reddit
- `/trivia` - Trivia questions with scoring
- `/trivia-leaderboard` - Top trivia players
- `/would-you-rather` - Random "would you rather" questions
- `/random-number` - Generate random number
- `/random-choice` - Pick from options
- `/flip-text` - Upside-down text
- `/mock-text` - aLtErNaTiNg CaPs
- `/reverse-text` - Reverse text
- `/rate` - Rate something out of 10
- `/ship` - Compatibility calculator

### 🛠️ Utility (17 commands)
- `/help` - Command list and details
- `/ping` - Bot latency check
- `/avatar` - Display user avatar
- `/userinfo` - User details
- `/serverinfo` - Server statistics
- `/poll` - Create reaction polls
- `/remind` - Set reminders (30s, 2h, 1d)
- `/reminders` - List/manage reminders
- `/calc` - Math calculator
- `/timer` - Countdown timer
- `/afk` - Set AFK status with auto-response
- `/translate` - Language translation
- `/stats` - Bot statistics
- `/role-info` - Role details
- `/channel-info` - Channel details
- `/clear-my-data` - GDPR compliance data removal

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
