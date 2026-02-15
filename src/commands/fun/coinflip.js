/**
 * Coinflip Command
 * Flip a coin - heads or tails
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('coinflip')
    .setDescription('Flip a coin'),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const result = Math.random() < 0.5 ? 'Heads' : 'Tails';
    const emoji = result === 'Heads' ? '🪙' : '💰';

    const embed = funEmbed(
      `${emoji} Coin Flip`,
      `The coin landed on **${result}**!`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
