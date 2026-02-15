/**
 * Clear My Data Command
 * GDPR compliance - allows users to remove all their data
 */

const { SlashCommandBuilder } = require('discord.js');
const { successEmbed, warningEmbed } = require('../../utils/embedBuilder');
const { clearAllUserData } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('clear-my-data')
    .setDescription('Remove all your data from the bot (GDPR compliance)')
    .addBooleanOption(option =>
      option.setName('confirm')
        .setDescription('Confirm that you want to delete all your data')
        .setRequired(true)),

  cooldown: 60,

  async execute(interaction) {
    const confirmed = interaction.options.getBoolean('confirm');

    if (!confirmed) {
      const embed = warningEmbed(
        'Data Deletion Cancelled',
        'You must confirm to delete your data. Run the command again with confirm set to `True`.'
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    try {
      // Clear all user data
      clearAllUserData(interaction.user.id);

      const embed = successEmbed(
        'Data Deleted',
        'All your data has been removed from the bot, including:\n• Trivia scores\n• Reminders\n• AFK status\n• Command usage statistics'
      );

      await interaction.reply({ embeds: [embed], ephemeral: true });
    } catch (error) {
      console.error('Error clearing user data:', error);

      const embed = errorEmbed(
        'Error',
        'An error occurred while deleting your data. Please try again later.'
      );

      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  }
};
