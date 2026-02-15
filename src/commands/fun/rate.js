/**
 * Rate Command
 * Rate something out of 10
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('rate')
    .setDescription('Rate something out of 10')
    .addStringOption(option =>
      option.setName('thing')
        .setDescription('What to rate')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const thing = interaction.options.getString('thing');

    // Generate a deterministic rating based on the input
    // This way the same thing always gets the same rating
    let hash = 0;
    for (let i = 0; i < thing.length; i++) {
      hash = ((hash << 5) - hash) + thing.charCodeAt(i);
      hash = hash & hash; // Convert to 32-bit integer
    }

    const rating = Math.abs(hash % 11); // 0-10

    // Star rating visual
    const stars = '⭐'.repeat(rating) + '☆'.repeat(10 - rating);

    const embed = funEmbed(
      '⭐ Rating',
      `I'd rate **${thing}** a **${rating}/10**!\n\n${stars}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
