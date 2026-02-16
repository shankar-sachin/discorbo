/**
 * Interaction Create Event Handler
 * Handles all slash command interactions
 */

const logger = require('../utils/logger');
const { checkCooldown } = require('../utils/cooldownManager');
const { handleCommandError, handleValidationError } = require('../utils/errorHandler');
const { errorEmbed, warningEmbed } = require('../utils/embedBuilder');
const { updateUserData, readJSON } = require('../utils/dataManager');

module.exports = {
  name: 'interactionCreate',
  once: false,

  async execute(interaction, client) {
    // Only handle slash commands
    if (!interaction.isChatInputCommand()) return;

    const command = client.commands.get(interaction.commandName);

    // Command not found
    if (!command) {
      logger.warn(`Unknown command: ${interaction.commandName}`);
      return;
    }

    // Allow Go runtime to own specific commands during incremental migration.
    if (process.env.MAZE_RUNTIME === 'go' && ['maze', 'maze-leaderboard'].includes(interaction.commandName)) {
      return;
    }

    // Check cooldown
    const cooldownAmount = command.cooldown || 3;
    const { onCooldown, timeLeft } = checkCooldown(
      interaction.user.id,
      interaction.commandName,
      cooldownAmount
    );

    if (onCooldown) {
      const embed = warningEmbed(
        'Cooldown Active',
        `Please wait ${timeLeft} more second(s) before using \`/${interaction.commandName}\` again.`
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    // Log command usage
    logger.command(interaction.user.tag, interaction.commandName);

    // Track command usage statistics
    try {
      const usage = readJSON('command-usage.json');
      if (!usage[interaction.user.id]) {
        usage[interaction.user.id] = {};
      }
      if (!usage[interaction.user.id][interaction.commandName]) {
        usage[interaction.user.id][interaction.commandName] = 0;
      }
      usage[interaction.user.id][interaction.commandName]++;

      const { writeJSON } = require('../utils/dataManager');
      writeJSON('command-usage.json', usage);
    } catch (error) {
      logger.error('Failed to track command usage:', error);
    }

    // Execute command
    try {
      await command.execute(interaction, client);
    } catch (error) {
      await handleCommandError(error, interaction, interaction.commandName);
    }
  }
};
