/**
 * Centralized bot configuration
 * Loads environment variables and provides configuration constants
 */

require('dotenv').config();

module.exports = {
  // Discord credentials
  token: process.env.DISCORD_TOKEN,
  clientId: process.env.CLIENT_ID,
  guildId: process.env.GUILD_ID,
  ownerId: process.env.OWNER_ID,

  // Bot settings
  presence: {
    status: 'online',
    activities: [{
      name: '/help for commands',
      type: 0 // PLAYING
    }]
  },

  // Colors for embeds (Discord standard colors)
  colors: {
    success: 0x57F287,
    error: 0xED4245,
    info: 0x5865F2,
    warning: 0xFEE75C,
    fun: 0xEB459E
  },

  // Default cooldowns (in seconds)
  cooldowns: {
    default: 3,
    api: 10,
    spam: 5
  },

  // API endpoints
  apis: {
    jokes: 'https://v2.jokeapi.dev/joke',
    trivia: 'https://opentdb.com/api.php',
    memes: 'https://www.reddit.com/r/memes.json',
    translate: 'https://libretranslate.com/translate'
  },

  // Feature flags
  features: {
    reminders: true,
    afk: true,
    trivia: true,
    translation: true
  },

  // Limits
  limits: {
    pollOptions: 10,
    reminderMax: 50, // per user
    messageLength: 2000,
    embedFields: 25
  }
};
