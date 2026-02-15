/**
 * Trivia Leaderboard Command
 * Display top trivia players
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { readJSON } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('trivia-leaderboard')
    .setDescription('View the trivia leaderboard'),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const scores = readJSON('trivia-scores.json');

    // Convert to array and sort by score
    const leaderboard = Object.entries(scores)
      .map(([userId, data]) => ({
        userId,
        username: data.username,
        score: data.score,
        correct: data.correct,
        total: data.total,
        accuracy: data.total > 0 ? ((data.correct / data.total) * 100).toFixed(1) : 0
      }))
      .sort((a, b) => b.score - a.score)
      .slice(0, 10); // Top 10

    if (leaderboard.length === 0) {
      const embed = new EmbedBuilder()
        .setColor(0xEB459E)
        .setTitle('🏆 Trivia Leaderboard')
        .setDescription('No scores yet! Use `/trivia` to start playing.')
        .setTimestamp()
        .setFooter({ text: 'Discorbo • Trivia Leaderboard' });

      return interaction.reply({ embeds: [embed] });
    }

    // Build leaderboard text
    let leaderboardText = '';
    leaderboard.forEach((player, index) => {
      const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
      leaderboardText += `${medal} **${player.username}** - ${player.score} pts (${player.accuracy}% accuracy)\n`;
    });

    // Find user's rank if not in top 10
    const allScores = Object.entries(scores)
      .map(([userId, data]) => ({ userId, score: data.score }))
      .sort((a, b) => b.score - a.score);

    const userRank = allScores.findIndex(s => s.userId === interaction.user.id) + 1;
    const userScore = scores[interaction.user.id];

    const embed = new EmbedBuilder()
      .setColor(0xEB459E)
      .setTitle('🏆 Trivia Leaderboard')
      .setDescription(leaderboardText)
      .setTimestamp()
      .setFooter({ text: 'Discorbo • Trivia Leaderboard' });

    // Add user's stats if they have played
    if (userScore) {
      const accuracy = userScore.total > 0 ? ((userScore.correct / userScore.total) * 100).toFixed(1) : 0;
      embed.addFields({
        name: 'Your Stats',
        value: `**Rank:** #${userRank}\n**Score:** ${userScore.score} pts\n**Correct:** ${userScore.correct}/${userScore.total} (${accuracy}%)`,
        inline: false
      });
    }

    await interaction.reply({ embeds: [embed] });
  }
};
