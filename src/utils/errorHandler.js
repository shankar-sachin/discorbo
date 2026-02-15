/**
 * Centralized error handling utility
 * Provides consistent error responses and logging
 */

const logger = require('./logger');
const { errorEmbed } = require('./embedBuilder');

/**
 * Handle command execution errors
 * @param {Error} error - The error object
 * @param {Interaction} interaction - Discord interaction
 * @param {string} commandName - Name of the command that errored
 */
async function handleCommandError(error, interaction, commandName) {
  logger.error(`Error in command ${commandName}:`, error);

  const embed = errorEmbed(
    'Something went wrong!',
    'An error occurred while executing this command. Please try again later.'
  );

  try {
    // Try to reply or follow up depending on interaction state
    if (interaction.replied || interaction.deferred) {
      await interaction.followUp({ embeds: [embed], ephemeral: true });
    } else {
      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  } catch (replyError) {
    logger.error('Failed to send error message:', replyError);
  }
}

/**
 * Handle API errors with user-friendly messages
 * @param {Error} error - The API error
 * @param {Interaction} interaction - Discord interaction
 * @param {string} apiName - Name of the API that failed
 */
async function handleApiError(error, interaction, apiName) {
  logger.error(`${apiName} API error:`, error.message);

  const embed = errorEmbed(
    `${apiName} Unavailable`,
    `Sorry, ${apiName} is currently unavailable. Please try again later.`
  );

  try {
    if (interaction.replied || interaction.deferred) {
      await interaction.followUp({ embeds: [embed], ephemeral: true });
    } else {
      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  } catch (replyError) {
    logger.error('Failed to send API error message:', replyError);
  }
}

/**
 * Handle permission errors
 * @param {Interaction} interaction - Discord interaction
 * @param {string} permission - Missing permission
 */
async function handlePermissionError(interaction, permission) {
  const embed = errorEmbed(
    'Missing Permissions',
    `I don't have permission to ${permission}. Please contact a server administrator.`
  );

  try {
    await interaction.reply({ embeds: [embed], ephemeral: true });
  } catch (error) {
    logger.error('Failed to send permission error:', error);
  }
}

/**
 * Handle validation errors
 * @param {Interaction} interaction - Discord interaction
 * @param {string} message - Validation error message
 */
async function handleValidationError(interaction, message) {
  const embed = errorEmbed(
    'Invalid Input',
    message
  );

  try {
    if (interaction.replied || interaction.deferred) {
      await interaction.followUp({ embeds: [embed], ephemeral: true });
    } else {
      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  } catch (error) {
    logger.error('Failed to send validation error:', error);
  }
}

/**
 * Log uncaught errors
 */
function setupGlobalErrorHandlers() {
  process.on('unhandledRejection', (error) => {
    logger.error('Unhandled promise rejection:', error);
  });

  process.on('uncaughtException', (error) => {
    logger.error('Uncaught exception:', error);
    // Don't exit in production, just log
    if (process.env.NODE_ENV === 'development') {
      process.exit(1);
    }
  });
}

module.exports = {
  handleCommandError,
  handleApiError,
  handlePermissionError,
  handleValidationError,
  setupGlobalErrorHandlers
};
