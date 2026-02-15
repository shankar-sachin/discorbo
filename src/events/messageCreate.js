/**
 * Message Create Event Handler
 * Handles AFK system - detects mentions and user returns
 */

const { readJSON, writeJSON, deleteUserData } = require('../utils/dataManager');
const { infoEmbed } = require('../utils/embedBuilder');
const logger = require('../utils/logger');

module.exports = {
  name: 'messageCreate',
  once: false,

  async execute(message, client) {
    // Ignore bot messages
    if (message.author.bot) return;

    const afkData = readJSON('afk-users.json');

    // Check if user is returning from AFK
    if (afkData[message.author.id]) {
      const { reason, timestamp } = afkData[message.author.id];
      const afkDuration = getTimeDifference(timestamp);

      // Remove AFK status
      deleteUserData('afk-users.json', message.author.id);

      // Send welcome back message
      const embed = infoEmbed(
        'Welcome Back!',
        `You were AFK for ${afkDuration}.\n**Reason:** ${reason}`
      );

      try {
        await message.reply({ embeds: [embed] });
        logger.info(`${message.author.tag} returned from AFK`);
      } catch (error) {
        logger.error('Failed to send AFK return message:', error);
      }
    }

    // Check if any mentioned users are AFK
    if (message.mentions.users.size > 0) {
      message.mentions.users.forEach(async (user) => {
        // Don't check bot users
        if (user.bot) return;

        if (afkData[user.id]) {
          const { reason, timestamp } = afkData[user.id];
          const afkDuration = getTimeDifference(timestamp);

          const embed = infoEmbed(
            `${user.username} is AFK`,
            `**Reason:** ${reason}\n**Duration:** ${afkDuration}`
          );

          try {
            await message.reply({ embeds: [embed] });
          } catch (error) {
            logger.error('Failed to send AFK notification:', error);
          }
        }
      });
    }
  }
};

/**
 * Calculate time difference in human-readable format
 */
function getTimeDifference(timestamp) {
  const now = Date.now();
  const diff = now - timestamp;

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days} day${days > 1 ? 's' : ''}`;
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''}`;
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''}`;
  return `${seconds} second${seconds > 1 ? 's' : ''}`;
}
