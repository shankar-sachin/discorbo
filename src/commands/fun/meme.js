/**
 * Meme Command
 * Fetch random memes from Reddit
 */

const { SlashCommandBuilder } = require('discord.js');
const axios = require('axios');
const { funEmbed } = require('../../utils/embedBuilder');
const { handleApiError } = require('../../utils/errorHandler');

const subreddits = ['memes', 'dankmemes', 'wholesomememes', 'me_irl', 'AdviceAnimals'];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('meme')
    .setDescription('Get a random meme from Reddit'),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    await interaction.deferReply();

    try {
      // Pick a random subreddit
      const subreddit = subreddits[Math.floor(Math.random() * subreddits.length)];

      const response = await axios.get(`https://www.reddit.com/r/${subreddit}/hot.json`, {
        params: { limit: 100 },
        headers: { 'User-Agent': 'Discorbo Bot' },
        timeout: 5000
      });

      // Filter for image posts
      const posts = response.data.data.children
        .map(child => child.data)
        .filter(post =>
          !post.stickied &&
          !post.is_video &&
          (post.url.endsWith('.jpg') || post.url.endsWith('.png') || post.url.endsWith('.gif'))
        );

      if (posts.length === 0) {
        throw new Error('No memes found');
      }

      // Pick random post
      const post = posts[Math.floor(Math.random() * posts.length)];

      const embed = funEmbed(post.title, `👍 ${post.ups} upvotes | 💬 ${post.num_comments} comments`)
        .setImage(post.url)
        .setFooter({ text: `r/${subreddit} • Discorbo` })
        .setURL(`https://reddit.com${post.permalink}`);

      await interaction.editReply({ embeds: [embed] });

    } catch (error) {
      await handleApiError(error, interaction, 'Reddit');
    }
  }
};
