/**
 * Deploy slash commands to Discord
 * Run this script whenever you add or modify commands
 * Usage: npm run deploy
 */

const { REST, Routes } = require('discord.js');
const fs = require('fs');
const path = require('path');
const config = require('./config');
const logger = require('./utils/logger');

const commands = [];

/**
 * Recursively load command files and extract data
 */
function loadCommandData(dir) {
  const files = fs.readdirSync(dir);

  for (const file of files) {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);

    if (stat.isDirectory()) {
      loadCommandData(filePath);
    } else if (file.endsWith('.js')) {
      try {
        const command = require(filePath);
        if ('data' in command && 'execute' in command) {
          commands.push(command.data.toJSON());
          logger.info(`Loaded command data: ${command.data.name}`);
        }
      } catch (error) {
        logger.error(`Error loading ${file}:`, error.message);
      }
    }
  }
}

// Load all commands
const commandsPath = path.join(__dirname, 'commands');
if (fs.existsSync(commandsPath)) {
  loadCommandData(commandsPath);
  logger.success(`Loaded ${commands.length} commands`);
} else {
  logger.error('Commands directory not found!');
  process.exit(1);
}

// Validate environment variables
if (!config.token || !config.clientId) {
  logger.error('Missing DISCORD_TOKEN or CLIENT_ID in .env file!');
  process.exit(1);
}

// Create REST instance
const rest = new REST({ version: '10' }).setToken(config.token);

// Deploy commands
(async () => {
  try {
    logger.info('Deploying slash commands...');

    // Deploy to specific guild (faster, for development) or globally
    if (config.guildId) {
      logger.info(`Deploying to guild: ${config.guildId}`);
      await rest.put(
        Routes.applicationGuildCommands(config.clientId, config.guildId),
        { body: commands }
      );
      logger.success(`Successfully deployed ${commands.length} commands to guild!`);
    } else {
      logger.info('Deploying globally (this may take up to an hour)...');
      await rest.put(
        Routes.applicationCommands(config.clientId),
        { body: commands }
      );
      logger.success(`Successfully deployed ${commands.length} commands globally!`);
    }

    // Display deployed commands
    logger.info('\nDeployed commands:');
    commands.forEach(cmd => {
      logger.info(`  /${cmd.name} - ${cmd.description}`);
    });

  } catch (error) {
    logger.error('Failed to deploy commands:', error);
    process.exit(1);
  }
})();
