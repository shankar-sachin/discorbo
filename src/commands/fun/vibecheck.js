/**
 * Vibe Check Command
 * Returns a random vibe rating with animated emoji response
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const vibes = [
  {
    rating: 'Immaculate ✨',
    emoji: '✨',
    message: 'Your vibe is absolutely *chef\'s kiss*. The energy is unmatched.',
    color: 0xFFD700
  },
  {
    rating: 'Main Character Energy 🌟',
    emoji: '🌟',
    message: 'You\'re the protagonist and everyone else is just a side character.',
    color: 0xFF69B4
  },
  {
    rating: 'Suspicious 🤨',
    emoji: '🤨',
    message: 'Something\'s off... Can\'t quite put my finger on it.',
    color: 0xFFA500
  },
  {
    rating: 'Villain Arc 😈',
    emoji: '😈',
    message: 'You\'ve chosen chaos. Respect.',
    color: 0x8B0000
  },
  {
    rating: 'Chaotic Neutral 🌀',
    emoji: '🌀',
    message: 'Unpredictable. Dangerous. We love to see it.',
    color: 0x9370DB
  },
  {
    rating: 'Unhinged 🎪',
    emoji: '🎪',
    message: 'Absolutely feral. No thoughts, just vibes.',
    color: 0xFF1493
  },
  {
    rating: 'NPC Energy 👤',
    emoji: '👤',
    message: 'You\'re here. That\'s about it.',
    color: 0x808080
  },
  {
    rating: 'Legendary 👑',
    emoji: '👑',
    message: 'Rare drop. Divine tier. Everyone bow down.',
    color: 0xFFD700
  },
  {
    rating: 'Touch Grass Tier 🌱',
    emoji: '🌱',
    message: 'When was the last time you saw the sun?',
    color: 0x90EE90
  },
  {
    rating: 'Cosmic Horror 🌌',
    emoji: '🌌',
    message: 'Your existence defies comprehension. Fear.',
    color: 0x4B0082
  }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('vibecheck')
    .setDescription('Check your current vibe rating')
    .addUserOption(option =>
      option.setName('user')
        .setDescription('Check someone else\'s vibe')
        .setRequired(false)),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const targetUser = interaction.options.getUser('user') || interaction.user;

    // Deterministic vibe based on user ID and current day
    const today = new Date().toDateString();
    const seed = parseInt(targetUser.id) + today.split('').reduce((a, b) => a + b.charCodeAt(0), 0);
    const vibeIndex = seed % vibes.length;
    const vibe = vibes[vibeIndex];

    // Create progress bar
    const percentage = ((vibeIndex + 1) / vibes.length * 100).toFixed(0);
    const barLength = 20;
    const filledLength = Math.floor(barLength * percentage / 100);
    const bar = '█'.repeat(filledLength) + '░'.repeat(barLength - filledLength);

    const embed = funEmbed(
      `${vibe.emoji} Vibe Check: ${targetUser.username}`,
      `**Status:** ${vibe.rating}\n\n${vibe.message}\n\n**Vibe Meter:**\n${bar} ${percentage}%`
    )
      .setColor(vibe.color)
      .setThumbnail(targetUser.displayAvatarURL({ dynamic: true }));

    await interaction.reply({ embeds: [embed] });
  }
};
