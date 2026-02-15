/**
 * Joke Command
 * Fetch jokes from JokeAPI
 */

const { SlashCommandBuilder } = require('discord.js');
const axios = require('axios');
const { funEmbed } = require('../../utils/embedBuilder');
const { handleApiError } = require('../../utils/errorHandler');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('joke')
    .setDescription('Get a random joke')
    .addStringOption(option =>
      option.setName('category')
        .setDescription('Joke category')
        .setRequired(false)
        .addChoices(
          { name: 'Any', value: 'Any' },
          { name: 'Programming', value: 'Programming' },
          { name: 'Miscellaneous', value: 'Misc' },
          { name: 'Pun', value: 'Pun' },
          { name: 'Spooky', value: 'Spooky' },
          { name: 'Christmas', value: 'Christmas' }
        )),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const category = interaction.options.getString('category') || 'Any';

    await interaction.deferReply();

    try {
      const response = await axios.get(`https://v2.jokeapi.dev/joke/${category}`, {
        params: {
          blacklistFlags: 'nsfw,religious,political,racist,sexist,explicit'
        },
        timeout: 5000
      });

      const joke = response.data;

      let jokeText = '';
      if (joke.type === 'single') {
        jokeText = joke.joke;
      } else {
        jokeText = `${joke.setup}\n\n||${joke.delivery}||`;
      }

      const embed = funEmbed(
        `😂 ${joke.category} Joke`,
        jokeText
      );

      await interaction.editReply({ embeds: [embed] });

    } catch (error) {
      await handleApiError(error, interaction, 'Joke API');
    }
  }
};
