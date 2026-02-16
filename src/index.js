/**
 * Discorbo - Discord Bot Entry Point
 * A comprehensive bot with 32+ slash commands for fun and utility
 */

const { Client, Collection, GatewayIntentBits } = require('discord.js');
const fs = require('fs');
const path = require('path');
const config = require('./config');
const logger = require('./utils/logger');
const { setupGlobalErrorHandlers } = require('./utils/errorHandler');

// Setup global error handlers
setupGlobalErrorHandlers();

// Prevent multiple bot instances from running concurrently in this workspace.
const lockPath = path.join(__dirname, '.bot.lock');
let lockFd = null;
try {
  lockFd = fs.openSync(lockPath, 'wx');
  fs.writeFileSync(lockFd, String(process.pid), 'utf8');
} catch (error) {
  if (error.code === 'EEXIST') {
    // Recover from stale lock files after an unclean shutdown.
    let staleLock = false;
    try {
      const existingPid = Number.parseInt(fs.readFileSync(lockPath, 'utf8').trim(), 10);
      if (!Number.isInteger(existingPid)) {
        staleLock = true;
      } else {
        try {
          process.kill(existingPid, 0);
        } catch (_) {
          staleLock = true;
        }
      }
    } catch (_) {
      staleLock = true;
    }

    if (staleLock) {
      try { fs.unlinkSync(lockPath); } catch (_) {}
      lockFd = fs.openSync(lockPath, 'wx');
      fs.writeFileSync(lockFd, String(process.pid), 'utf8');
    } else {
      logger.error('Another bot instance is already running (.bot.lock exists). Exiting this process.');
      process.exit(1);
    }
  } else {
    throw error;
  }
}

function releaseLock() {
  try {
    if (lockFd !== null) {
      fs.closeSync(lockFd);
      lockFd = null;
    }
    if (fs.existsSync(lockPath)) {
      fs.unlinkSync(lockPath);
    }
  } catch (_) {
    // Best-effort cleanup; ignore failures.
  }
}

process.on('exit', releaseLock);
process.on('SIGINT', () => { releaseLock(); process.exit(0); });
process.on('SIGTERM', () => { releaseLock(); process.exit(0); });

// Create Discord client with required intents
const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMembers,
    GatewayIntentBits.GuildMessages,
    GatewayIntentBits.MessageContent,
  ],
  presence: config.presence
});

// Initialize command collection
client.commands = new Collection();
const includeJSCommands = process.env.JS_COMMANDS === 'true';

/**
 * Recursively load command files from a directory
 */
function loadCommands(dir) {
  const files = fs.readdirSync(dir);

  for (const file of files) {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);

    if (stat.isDirectory()) {
      if (!includeJSCommands) {
        logger.info('Skipping JS commands (Go runtime is primary).');
        continue;
      }
      // Recursively load commands from subdirectories
      loadCommands(filePath);
    } else if (file.endsWith('.js')) {
      try {
        const command = require(filePath);

        // Validate command structure
        if ('data' in command && 'execute' in command) {
          client.commands.set(command.data.name, command);
          logger.success(`Loaded command: ${command.data.name}`);
        } else {
          logger.warn(`Command at ${filePath} is missing "data" or "execute" property`);
        }
      } catch (error) {
        logger.error(`Error loading command ${file}:`, error.message);
      }
    }
  }
}

/**
 * Load all event handlers
 */
function loadEvents() {
  const eventsPath = path.join(__dirname, 'events');

  if (!fs.existsSync(eventsPath)) {
    logger.warn('Events directory not found');
    return;
  }

  const eventFiles = fs.readdirSync(eventsPath).filter(file => file.endsWith('.js'));

  for (const file of eventFiles) {
    try {
      const filePath = path.join(eventsPath, file);
      const event = require(filePath);

      if (event.once) {
        client.once(event.name, (...args) => event.execute(...args, client));
      } else {
        client.on(event.name, (...args) => event.execute(...args, client));
      }

      logger.success(`Loaded event: ${event.name}`);
    } catch (error) {
      logger.error(`Error loading event ${file}:`, error.message);
    }
  }
}

// Load commands and events
logger.info('Loading commands...');
const commandsPath = path.join(__dirname, 'commands');
if (fs.existsSync(commandsPath)) {
  loadCommands(commandsPath);
} else {
  logger.warn('Commands directory not found');
}

logger.info('Loading events...');
loadEvents();

// Login to Discord
if (!config.token) {
  logger.error('DISCORD_TOKEN not found in environment variables!');
  logger.error('Please create a .env file with your bot token.');
  process.exit(1);
}

client.login(config.token)
  .then(() => {
    logger.success('Login successful!');
  })
  .catch((error) => {
    logger.error('Failed to login:', error.message);
    releaseLock();
    process.exit(1);
  });

// Export client for potential testing
module.exports = client;
