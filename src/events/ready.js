/**
 * Ready Event Handler
 * Fires when the bot successfully logs in and is ready to operate
 */

const logger = require('../utils/logger');
const { checkReminders } = require('../utils/reminderChecker');

module.exports = {
  name: 'clientReady',
  once: true,

  async execute(client) {
    logger.success(`Logged in as ${client.user.tag}`);
    logger.info(`Serving ${client.guilds.cache.size} server(s)`);
    logger.info(`Loaded ${client.commands.size} command(s)`);

    // Set bot presence
    client.user.setPresence({
      activities: [{ name: '/help for commands', type: 0 }],
      status: 'online'
    });

    // Start reminder checker (runs every 30 seconds)
    setInterval(() => {
      try {
        checkReminders(client);
      } catch (error) {
        logger.error('Error in reminder checker:', error);
      }
    }, 30000); // 30 seconds

    logger.success('Bot is ready!');
  }
};
