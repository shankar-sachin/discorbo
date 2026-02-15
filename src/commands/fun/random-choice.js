/**
 * Random Choice Command
 * Pick a random option from user-provided choices
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('random-choice')
    .setDescription('Pick a random option from a list')
    .addStringOption(option =>
      option.setName('options')
        .setDescription('Options separated by commas (e.g., "pizza, burger, sushi")')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const optionsString = interaction.options.getString('options');

    // Split by comma and trim whitespace
    const options = optionsString.split(',').map(opt => opt.trim()).filter(opt => opt.length > 0);

    if (options.length < 2) {
      const embed = errorEmbed(
        'Not Enough Options',
        'Please provide at least 2 options separated by commas.\n**Example:** `pizza, burger, sushi`'
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    const choice = options[Math.floor(Math.random() * options.length)];

    const embed = funEmbed(
      '🎯 Random Choice',
      `Out of **${options.length}** options, I choose:\n\n**${choice}**`
    )
      .addFields({ name: 'Options', value: options.join(', '), inline: false });

    await interaction.reply({ embeds: [embed] });
  }
};
