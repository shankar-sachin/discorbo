/**
 * Maze Leaderboard Command
 * View top maze completion times
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { readJSON } = require('../../utils/dataManager');

const LEVEL_NAMES = [
  "Beginner's Path (Easy)",
  'Spike Corridor (Medium)',
  'Monster Maze (Hard)'
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('maze-leaderboard')
    .setDescription('View maze game leaderboards')
    .addIntegerOption(option =>
      option.setName('level')
        .setDescription('Filter by level')
        .addChoices(
          { name: "Easy - Beginner's Path", value: 0 },
          { name: 'Medium - Spike Corridor', value: 1 },
          { name: 'Hard - Monster Maze', value: 2 },
          { name: 'All Levels', value: -1 }
        )
        .setRequired(false)),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const levelFilter = interaction.options.getInteger('level') ?? -1;
    const data = normalizeMazeData(readJSON('maze-scores.json'));

    if (Object.keys(data).length === 0) {
      const embed = new EmbedBuilder()
        .setColor(0xEB459E)
        .setTitle('Maze Leaderboard')
        .setDescription('No one has completed any mazes yet. Use `/maze` to be the first.');

      await interaction.reply({ embeds: [embed] });
      return;
    }

    if (levelFilter === -1) {
      const leaderboard = Object.entries(data)
        .map(([userId, info]) => ({
          userId,
          username: info.username,
          totalCoins: info.totalCoins,
          completions: info.completions.length
        }))
        .sort((a, b) => {
          if (b.totalCoins !== a.totalCoins) return b.totalCoins - a.totalCoins;
          return b.completions - a.completions;
        })
        .slice(0, 10);

      const leaderboardText = leaderboard.map((player, index) =>
        `${index + 1}. **${player.username}**\n` +
        `   Coins: ${player.totalCoins} | Completions: ${player.completions}`
      ).join('\n');

      const embed = new EmbedBuilder()
        .setColor(0xEB459E)
        .setTitle('Maze Leaderboard - Total Coins')
        .setDescription(leaderboardText)
        .setFooter({ text: 'Use /maze to play.' });

      await interaction.reply({ embeds: [embed] });
      return;
    }

    const allCompletions = [];

    Object.entries(data).forEach(([userId, info]) => {
      info.completions
        .filter(completion => completion.level === levelFilter)
        .forEach(completion => {
          allCompletions.push({
            userId,
            username: info.username,
            ...completion
          });
        });
    });

    if (allCompletions.length === 0) {
      const embed = new EmbedBuilder()
        .setColor(0xEB459E)
        .setTitle(`${LEVEL_NAMES[levelFilter]} - Leaderboard`)
        .setDescription('No completions for this level yet. Be the first to complete it.');

      await interaction.reply({ embeds: [embed] });
      return;
    }

    allCompletions.sort((a, b) => {
      if (a.time !== b.time) return a.time - b.time;
      return b.coins - a.coins;
    });

    const top10 = allCompletions.slice(0, 10);

    const leaderboardText = top10.map((entry, index) =>
      `${index + 1}. **${entry.username}**\n` +
      `   Time: ${entry.time}s | Coins: ${entry.coins}`
    ).join('\n');

    const embed = new EmbedBuilder()
      .setColor(0xEB459E)
      .setTitle(`${LEVEL_NAMES[levelFilter]} - Fastest Times`)
      .setDescription(leaderboardText)
      .setFooter({ text: `Total completions: ${allCompletions.length}` });

    await interaction.reply({ embeds: [embed] });
  }
};

function normalizeMazeData(raw) {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return {};
  }

  const normalized = {};

  for (const [userId, info] of Object.entries(raw)) {
    if (!info || typeof info !== 'object') continue;

    const username = typeof info.username === 'string' && info.username.trim()
      ? info.username.trim()
      : 'Unknown User';

    const completionsRaw = Array.isArray(info.completions) ? info.completions : [];
    const completions = completionsRaw
      .filter(c => c && typeof c === 'object')
      .map(c => ({
        level: Number.isInteger(c.level) ? c.level : Number.parseInt(c.level, 10),
        time: Number.isFinite(Number(c.time)) ? Number(c.time) : null,
        coins: Number.isFinite(Number(c.coins)) ? Number(c.coins) : 0,
        timestamp: Number.isFinite(Number(c.timestamp)) ? Number(c.timestamp) : Date.now()
      }))
      .filter(c =>
        Number.isInteger(c.level) &&
        c.level >= 0 &&
        c.level < LEVEL_NAMES.length &&
        Number.isFinite(c.time) &&
        c.time > 0
      );

    const totalCoins = Number.isFinite(Number(info.totalCoins))
      ? Number(info.totalCoins)
      : completions.reduce((sum, c) => sum + c.coins, 0);

    if (completions.length > 0 || totalCoins > 0) {
      normalized[userId] = {
        username,
        completions,
        totalCoins
      };
    }
  }

  return normalized;
}
