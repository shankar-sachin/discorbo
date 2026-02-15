/**
 * Timer Command
 * Set a countdown timer with live updates
 */

const { SlashCommandBuilder } = require('discord.js');
const { infoEmbed, successEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('timer')
    .setDescription('Set a countdown timer')
    .addIntegerOption(option =>
      option.setName('seconds')
        .setDescription('Duration in seconds (max 300 / 5 minutes)')
        .setRequired(true)
        .setMinValue(1)
        .setMaxValue(300)),

  cooldown: 10,

  async execute(interaction) {
    const duration = interaction.options.getInteger('seconds');
    const endTime = Math.floor(Date.now() / 1000) + duration;

    // Initial reply
    const embed = infoEmbed(
      'Timer Started',
      `Timer will end <t:${endTime}:R>\n**Duration:** ${duration} seconds`
    );

    await interaction.reply({ embeds: [embed] });

    // Wait for the duration
    setTimeout(async () => {
      try {
        const doneEmbed = successEmbed(
          'Timer Complete!',
          `${interaction.user}, your ${duration} second timer has finished!`
        );

        await interaction.followUp({ content: `${interaction.user}`, embeds: [doneEmbed] });
      } catch (error) {
        console.error('Failed to send timer completion message:', error);
      }
    }, duration * 1000);
  }
};
