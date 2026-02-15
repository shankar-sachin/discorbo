/**
 * AFK Command
 * Set AFK status - bot will notify people who mention you
 */

const { SlashCommandBuilder } = require('discord.js');
const { successEmbed } = require('../../utils/embedBuilder');
const { updateUserData } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('afk')
    .setDescription('Set your AFK status')
    .addStringOption(option =>
      option.setName('reason')
        .setDescription('Why are you AFK?')
        .setRequired(false)),

  cooldown: 5,

  async execute(interaction) {
    const reason = interaction.options.getString('reason') || 'No reason provided';

    // Save AFK status
    const afkData = {
      reason: reason,
      timestamp: Date.now()
    };

    updateUserData('afk-users.json', interaction.user.id, afkData);

    const embed = successEmbed(
      'AFK Status Set',
      `You are now AFK.\n**Reason:** ${reason}\n\nYou will be marked as active when you send your next message.`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
