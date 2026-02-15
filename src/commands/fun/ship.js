/**
 * Ship Command
 * Calculate compatibility between two users
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('ship')
    .setDescription('Calculate compatibility between two users')
    .addUserOption(option =>
      option.setName('user1')
        .setDescription('First user')
        .setRequired(true))
    .addUserOption(option =>
      option.setName('user2')
        .setDescription('Second user')
        .setRequired(true)),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const user1 = interaction.options.getUser('user1');
    const user2 = interaction.options.getUser('user2');

    // Generate deterministic compatibility based on user IDs
    const combined = user1.id + user2.id;
    let hash = 0;
    for (let i = 0; i < combined.length; i++) {
      hash = ((hash << 5) - hash) + combined.charCodeAt(i);
      hash = hash & hash;
    }

    const compatibility = Math.abs(hash % 101); // 0-100

    // Create ship name
    const name1 = user1.username;
    const name2 = user2.username;
    const shipName = name1.slice(0, Math.ceil(name1.length / 2)) +
                     name2.slice(Math.floor(name2.length / 2));

    // Get compatibility message
    let message = '';
    let emoji = '';

    if (compatibility < 20) {
      message = 'Not a good match...';
      emoji = '💔';
    } else if (compatibility < 40) {
      message = 'Could work with effort.';
      emoji = '❤️‍🩹';
    } else if (compatibility < 60) {
      message = 'Decent compatibility!';
      emoji = '💗';
    } else if (compatibility < 80) {
      message = 'Great match!';
      emoji = '💖';
    } else {
      message = 'Perfect match!';
      emoji = '💕';
    }

    // Create progress bar
    const barLength = 20;
    const filledLength = Math.floor(barLength * compatibility / 100);
    const bar = '█'.repeat(filledLength) + '░'.repeat(barLength - filledLength);

    const embed = funEmbed(
      `${emoji} Ship Calculator`,
      `**${user1.username}** 💕 **${user2.username}**\n\n` +
      `**Ship Name:** ${shipName}\n` +
      `**Compatibility:** ${compatibility}%\n` +
      `${bar}\n\n` +
      `*${message}*`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
