/**
 * Mock Text Command
 * Convert text to alternating caps (mocking SpongeBob meme)
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('mock-text')
    .setDescription('CoNvErT tExT tO aLtErNaTiNg CaPs')
    .addStringOption(option =>
      option.setName('text')
        .setDescription('Text to mock')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const text = interaction.options.getString('text');

    let mocked = '';
    let shouldBeUpper = false;

    for (const char of text) {
      if (char.match(/[a-zA-Z]/)) {
        mocked += shouldBeUpper ? char.toUpperCase() : char.toLowerCase();
        shouldBeUpper = !shouldBeUpper;
      } else {
        mocked += char;
      }
    }

    const embed = funEmbed(
      '😏 Mock Text',
      `**Original:** ${text}\n**Mocked:** ${mocked}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
