/**
 * Random Number Command
 * Generate a random number within a range
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed, errorEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('random-number')
    .setDescription('Generate a random number')
    .addIntegerOption(option =>
      option.setName('min')
        .setDescription('Minimum value (default: 1)')
        .setRequired(false))
    .addIntegerOption(option =>
      option.setName('max')
        .setDescription('Maximum value (default: 100)')
        .setRequired(false)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const min = interaction.options.getInteger('min') || 1;
    const max = interaction.options.getInteger('max') || 100;

    if (min >= max) {
      const embed = errorEmbed(
        'Invalid Range',
        'Minimum value must be less than maximum value.'
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    const random = Math.floor(Math.random() * (max - min + 1)) + min;

    const embed = funEmbed(
      '🎲 Random Number',
      `Your random number between **${min}** and **${max}** is:\n\n**${random}**`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
