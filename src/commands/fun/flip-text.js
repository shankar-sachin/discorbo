/**
 * Flip Text Command
 * Flip text upside down using Unicode characters
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const flipMap = {
  'a': 'ɐ', 'b': 'q', 'c': 'ɔ', 'd': 'p', 'e': 'ǝ', 'f': 'ɟ', 'g': 'ƃ', 'h': 'ɥ',
  'i': 'ᴉ', 'j': 'ɾ', 'k': 'ʞ', 'l': 'l', 'm': 'ɯ', 'n': 'u', 'o': 'o', 'p': 'd',
  'q': 'b', 'r': 'ɹ', 's': 's', 't': 'ʇ', 'u': 'n', 'v': 'ʌ', 'w': 'ʍ', 'x': 'x',
  'y': 'ʎ', 'z': 'z',
  'A': '∀', 'B': 'q', 'C': 'Ɔ', 'D': 'p', 'E': 'Ǝ', 'F': 'Ⅎ', 'G': 'פ', 'H': 'H',
  'I': 'I', 'J': 'ſ', 'K': 'ʞ', 'L': '˥', 'M': 'W', 'N': 'N', 'O': 'O', 'P': 'Ԁ',
  'Q': 'b', 'R': 'ɹ', 'S': 'S', 'T': '┴', 'U': '∩', 'V': 'Λ', 'W': 'M', 'X': 'X',
  'Y': '⅄', 'Z': 'Z',
  '0': '0', '1': 'Ɩ', '2': 'ᄅ', '3': 'Ɛ', '4': 'ㄣ', '5': 'ϛ', '6': '9', '7': 'ㄥ',
  '8': '8', '9': '6',
  '.': '˙', ',': '\'', '!': '¡', '?': '¿', '\'': ',', '"': ',,', ';': '؛',
  '(': ')', ')': '(', '[': ']', ']': '[', '{': '}', '}': '{', '<': '>', '>': '<',
  '&': '⅋', '_': '‾', ' ': ' '
};

module.exports = {
  data: new SlashCommandBuilder()
    .setName('flip-text')
    .setDescription('Flip text upside down')
    .addStringOption(option =>
      option.setName('text')
        .setDescription('Text to flip')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const text = interaction.options.getString('text');

    // Flip and reverse the text
    let flipped = '';
    for (let i = text.length - 1; i >= 0; i--) {
      const char = text[i];
      flipped += flipMap[char] || char;
    }

    const embed = funEmbed(
      '🔃 Flipped Text',
      `**Original:** ${text}\n**Flipped:** ${flipped}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
