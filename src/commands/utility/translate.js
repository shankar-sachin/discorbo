/**
 * Translate Command
 * Translate text between languages
 */

const { SlashCommandBuilder } = require('discord.js');
const axios = require('axios');
const { infoEmbed } = require('../../utils/embedBuilder');
const { handleApiError } = require('../../utils/errorHandler');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('translate')
    .setDescription('Translate text to another language')
    .addStringOption(option =>
      option.setName('text')
        .setDescription('Text to translate')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('to')
        .setDescription('Target language')
        .setRequired(true)
        .addChoices(
          { name: 'Spanish', value: 'es' },
          { name: 'French', value: 'fr' },
          { name: 'German', value: 'de' },
          { name: 'Italian', value: 'it' },
          { name: 'Portuguese', value: 'pt' },
          { name: 'Russian', value: 'ru' },
          { name: 'Japanese', value: 'ja' },
          { name: 'Korean', value: 'ko' },
          { name: 'Chinese (Simplified)', value: 'zh' },
          { name: 'Arabic', value: 'ar' },
          { name: 'Hindi', value: 'hi' },
          { name: 'Dutch', value: 'nl' },
          { name: 'Polish', value: 'pl' },
          { name: 'Turkish', value: 'tr' },
          { name: 'Swedish', value: 'sv' }
        ))
    .addStringOption(option =>
      option.setName('from')
        .setDescription('Source language (auto-detect if not specified)')
        .setRequired(false)
        .addChoices(
          { name: 'English', value: 'en' },
          { name: 'Spanish', value: 'es' },
          { name: 'French', value: 'fr' },
          { name: 'German', value: 'de' },
          { name: 'Italian', value: 'it' },
          { name: 'Portuguese', value: 'pt' },
          { name: 'Russian', value: 'ru' },
          { name: 'Japanese', value: 'ja' },
          { name: 'Korean', value: 'ko' },
          { name: 'Chinese (Simplified)', value: 'zh' }
        )),

  cooldown: 10,

  async execute(interaction) {
    const text = interaction.options.getString('text');
    const targetLang = interaction.options.getString('to');
    const sourceLang = interaction.options.getString('from') || 'auto';

    await interaction.deferReply();

    try {
      // Use LibreTranslate API (free, no auth required)
      const response = await axios.post('https://libretranslate.com/translate', {
        q: text,
        source: sourceLang,
        target: targetLang,
        format: 'text'
      }, {
        timeout: 5000,
        headers: { 'Content-Type': 'application/json' }
      });

      const translatedText = response.data.translatedText;
      const detectedLang = response.data.detectedLanguage?.language || sourceLang;

      const embed = infoEmbed(
        'Translation',
        `**Original (${detectedLang.toUpperCase()}):**\n${text}\n\n**Translated (${targetLang.toUpperCase()}):**\n${translatedText}`
      );

      await interaction.editReply({ embeds: [embed] });

    } catch (error) {
      await handleApiError(error, interaction, 'Translation API');
    }
  }
};
