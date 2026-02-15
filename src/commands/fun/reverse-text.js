/**
 * Reverse Text Command
 * Reverse the order of characters in text
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('reverse-text')
    .setDescription('Reverse text backwards')
    .addStringOption(option =>
      option.setName('text')
        .setDescription('Text to reverse')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const text = interaction.options.getString('text');
    const reversed = text.split('').reverse().join('');

    const embed = funEmbed(
      '🔄 Reversed Text',
      `**Original:** ${text}\n**Reversed:** ${reversed}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
