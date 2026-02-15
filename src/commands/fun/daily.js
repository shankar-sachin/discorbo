/**
 * Daily Command
 * Daily rewards and streak system
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { successEmbed, errorEmbed, funEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('daily')
    .setDescription('Claim your daily rewards and check stats')
    .addSubcommand(subcommand =>
      subcommand
        .setName('claim')
        .setDescription('Claim your daily reward'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('stats')
        .setDescription('View your reward stats'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('leaderboard')
        .setDescription('View the coins leaderboard')),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const rewardsData = readJSON('daily-rewards.json');

    if (!rewardsData[interaction.user.id]) {
      rewardsData[interaction.user.id] = {
        username: interaction.user.username,
        coins: 0,
        lastClaim: 0,
        streak: 0,
        totalClaims: 0,
        bossKills: 0
      };
    }

    const userData = rewardsData[interaction.user.id];
    userData.username = interaction.user.username;

    if (subcommand === 'claim') {
      const now = Date.now();
      const oneDayMs = 24 * 60 * 60 * 1000;
      const timeSinceLastClaim = now - userData.lastClaim;

      // Check if already claimed today
      if (timeSinceLastClaim < oneDayMs) {
        const timeLeft = oneDayMs - timeSinceLastClaim;
        const hoursLeft = Math.floor(timeLeft / (1000 * 60 * 60));
        const minutesLeft = Math.floor((timeLeft % (1000 * 60 * 60)) / (1000 * 60));

        const embed = errorEmbed(
          '⏰ Already Claimed Today',
          `You can claim your next daily reward in **${hoursLeft}h ${minutesLeft}m**!`
        );

        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Check if streak continues (claimed within 48 hours)
      const twoDaysMs = 2 * oneDayMs;
      if (timeSinceLastClaim < twoDaysMs) {
        userData.streak++;
      } else {
        userData.streak = 1;
      }

      // Calculate reward (base + streak bonus)
      const baseReward = 100;
      const streakBonus = Math.min(userData.streak * 10, 500); // Max 500 bonus
      const totalReward = baseReward + streakBonus;

      // Random bonus chance (10% chance for 2x)
      const bonusMultiplier = Math.random() < 0.1 ? 2 : 1;
      const finalReward = totalReward * bonusMultiplier;

      userData.coins += finalReward;
      userData.lastClaim = now;
      userData.totalClaims++;

      writeJSON('daily-rewards.json', rewardsData);

      const embed = successEmbed(
        '🎁 Daily Reward Claimed!',
        `You received **${finalReward} coins**!${bonusMultiplier > 1 ? '\n💎 **LUCKY BONUS! 2x Reward!**' : ''}\n\n` +
        `**Breakdown:**\n` +
        `• Base: ${baseReward} coins\n` +
        `• Streak Bonus (${userData.streak} days): +${streakBonus} coins\n\n` +
        `**Your Stats:**\n` +
        `• Total Coins: ${userData.coins.toLocaleString()}\n` +
        `• Streak: ${userData.streak} day(s)\n` +
        `• Total Claims: ${userData.totalClaims}`
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'stats') {
      const nextClaimTime = userData.lastClaim + (24 * 60 * 60 * 1000);
      const canClaimNow = Date.now() >= nextClaimTime;

      const embed = new EmbedBuilder()
        .setColor(0xFFD700)
        .setTitle(`💰 ${interaction.user.username}'s Rewards Stats`)
        .addFields(
          { name: 'Total Coins', value: userData.coins.toLocaleString(), inline: true },
          { name: 'Current Streak', value: `${userData.streak} day(s)`, inline: true },
          { name: 'Total Claims', value: userData.totalClaims.toString(), inline: true },
          { name: 'Boss Kills', value: userData.bossKills.toString(), inline: true },
          { name: 'Next Claim', value: canClaimNow ? '✅ Available Now!' : `<t:${Math.floor(nextClaimTime / 1000)}:R>`, inline: true }
        )
        .setThumbnail(interaction.user.displayAvatarURL({ dynamic: true }))
        .setTimestamp()
        .setFooter({ text: 'Discorbo • Daily Rewards' });

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'leaderboard') {
      const leaderboard = Object.entries(rewardsData)
        .map(([userId, data]) => ({
          userId,
          username: data.username,
          coins: data.coins,
          streak: data.streak
        }))
        .sort((a, b) => b.coins - a.coins)
        .slice(0, 10);

      if (leaderboard.length === 0) {
        const embed = funEmbed(
          '📊 Rewards Leaderboard',
          'No one has claimed rewards yet! Use `/daily claim` to get started.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      let leaderboardText = '';
      leaderboard.forEach((player, index) => {
        const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
        const streak = player.streak > 0 ? ` 🔥${player.streak}` : '';
        leaderboardText += `${medal} **${player.username}** - ${player.coins.toLocaleString()} coins${streak}\n`;
      });

      const embed = funEmbed('💰 Coins Leaderboard', leaderboardText);
      await interaction.reply({ embeds: [embed] });
    }
  }
};
