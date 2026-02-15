/**
 * Command cooldown manager
 * Prevents command spam by tracking usage timestamps
 */

const { Collection } = require('discord.js');
const logger = require('./logger');

// Store cooldowns: Map<commandName, Map<userId, timestamp>>
const cooldowns = new Collection();

/**
 * Check if a user is on cooldown for a command
 * @param {string} userId - Discord user ID
 * @param {string} commandName - Command name
 * @param {number} cooldownAmount - Cooldown duration in seconds
 * @returns {Object} { onCooldown: boolean, timeLeft: number }
 */
function checkCooldown(userId, commandName, cooldownAmount) {
  // Get or create cooldown collection for this command
  if (!cooldowns.has(commandName)) {
    cooldowns.set(commandName, new Collection());
  }

  const now = Date.now();
  const timestamps = cooldowns.get(commandName);
  const cooldownDuration = (cooldownAmount || 3) * 1000; // Default 3 seconds

  // Check if user has used this command recently
  if (timestamps.has(userId)) {
    const expirationTime = timestamps.get(userId) + cooldownDuration;

    if (now < expirationTime) {
      const timeLeft = (expirationTime - now) / 1000;
      return {
        onCooldown: true,
        timeLeft: timeLeft.toFixed(1)
      };
    }
  }

  // Set new timestamp
  timestamps.set(userId, now);

  // Auto-cleanup: remove expired cooldowns after duration
  setTimeout(() => timestamps.delete(userId), cooldownDuration);

  return {
    onCooldown: false,
    timeLeft: 0
  };
}

/**
 * Clear all cooldowns for a user (admin function)
 * @param {string} userId - Discord user ID
 */
function clearUserCooldowns(userId) {
  cooldowns.forEach(timestamps => {
    timestamps.delete(userId);
  });
  logger.debug(`Cleared cooldowns for user ${userId}`);
}

/**
 * Clear all cooldowns for a command (admin function)
 * @param {string} commandName - Command name
 */
function clearCommandCooldowns(commandName) {
  cooldowns.delete(commandName);
  logger.debug(`Cleared cooldowns for command ${commandName}`);
}

/**
 * Get cooldown statistics
 * @returns {Object} Cooldown stats
 */
function getCooldownStats() {
  const stats = {
    totalCommands: cooldowns.size,
    activeUsers: 0,
    commands: {}
  };

  cooldowns.forEach((timestamps, commandName) => {
    stats.commands[commandName] = timestamps.size;
    stats.activeUsers += timestamps.size;
  });

  return stats;
}

module.exports = {
  checkCooldown,
  clearUserCooldowns,
  clearCommandCooldowns,
  getCooldownStats
};
