/**
 * Quest Command
 * Generates mini RPG-style quests with XP and leaderboard
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed, successEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

const questTemplates = [
  { task: 'Retrieve the sacred pizza from the Forbidden Fridge', difficulty: 'easy', xp: 50 },
  { task: 'Defeat the Lag Monster in the Router Realm', difficulty: 'medium', xp: 100 },
  { task: 'Survive a family gathering without checking your phone', difficulty: 'hard', xp: 200 },
  { task: 'Touch grass for 5 consecutive minutes', difficulty: 'medium', xp: 75 },
  { task: 'Complete a group project where everyone actually contributes', difficulty: 'legendary', xp: 500 },
  { task: 'Find the TV remote (last seen 3 days ago)', difficulty: 'easy', xp: 30 },
  { task: 'Convince your parents you\'re actually studying', difficulty: 'hard', xp: 150 },
  { task: 'Achieve inbox zero', difficulty: 'legendary', xp: 300 },
  { task: 'Wake up before your alarm', difficulty: 'medium', xp: 80 },
  { task: 'Resist the urge to reply "ok" in a serious conversation', difficulty: 'hard', xp: 120 },
  { task: 'Escape the YouTube rabbit hole before 3 AM', difficulty: 'hard', xp: 180 },
  { task: 'Successfully parallel park on the first try', difficulty: 'legendary', xp: 400 },
  { task: 'Remember what you walked into this room for', difficulty: 'medium', xp: 90 },
  { task: 'Drink 8 glasses of water today', difficulty: 'easy', xp: 40 },
  { task: 'Actually read the Terms & Conditions', difficulty: 'legendary', xp: 600 },
  { task: 'Defeat the Final Boss: Monday Morning', difficulty: 'hard', xp: 200 },
  { task: 'Find matching socks in the void known as your drawer', difficulty: 'medium', xp: 70 },
  { task: 'Resist ordering food delivery for 24 hours', difficulty: 'hard', xp: 140 },
  { task: 'Clear your browser history before someone asks to use your computer', difficulty: 'easy', xp: 50 },
  { task: 'Venture into the Basement of Forgotten Hobbies', difficulty: 'medium', xp: 110 }
];

const difficultyEmojis = {
  easy: '⭐',
  medium: '⭐⭐',
  hard: '⭐⭐⭐',
  legendary: '👑'
};

module.exports = {
  data: new SlashCommandBuilder()
    .setName('quest')
    .setDescription('Get a random quest to complete!')
    .addSubcommand(subcommand =>
      subcommand
        .setName('get')
        .setDescription('Get a new quest'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('complete')
        .setDescription('Mark your current quest as complete'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('leaderboard')
        .setDescription('View the quest leaderboard'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('stats')
        .setDescription('View your quest statistics')),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const questData = readJSON('quests.json');

    if (!questData[interaction.user.id]) {
      questData[interaction.user.id] = {
        username: interaction.user.username,
        totalXP: 0,
        completedQuests: 0,
        currentQuest: null
      };
    }

    const userData = questData[interaction.user.id];
    userData.username = interaction.user.username; // Update username

    if (subcommand === 'get') {
      if (userData.currentQuest) {
        const embed = funEmbed(
          '📜 Active Quest',
          `You already have an active quest!\n\n**Quest:** ${userData.currentQuest.task}\n**Difficulty:** ${difficultyEmojis[userData.currentQuest.difficulty]} ${userData.currentQuest.difficulty}\n**Reward:** ${userData.currentQuest.xp} XP\n\nUse \`/quest complete\` when done!`
        );
        return interaction.reply({ embeds: [embed] });
      }

      // Assign new quest
      const quest = questTemplates[Math.floor(Math.random() * questTemplates.length)];
      userData.currentQuest = { ...quest, assignedAt: Date.now() };

      writeJSON('quests.json', questData);

      const embed = funEmbed(
        '⚔️ New Quest Assigned!',
        `**Quest:** ${quest.task}\n**Difficulty:** ${difficultyEmojis[quest.difficulty]} ${quest.difficulty}\n**Reward:** ${quest.xp} XP\n\nGood luck, adventurer! Use \`/quest complete\` when finished.`
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'complete') {
      if (!userData.currentQuest) {
        const embed = funEmbed(
          '❌ No Active Quest',
          'You don\'t have an active quest! Use `/quest get` to start one.'
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Complete quest
      const quest = userData.currentQuest;
      userData.totalXP += quest.xp;
      userData.completedQuests += 1;
      userData.currentQuest = null;

      writeJSON('quests.json', questData);

      const embed = successEmbed(
        '✅ Quest Completed!',
        `**Quest:** ${quest.task}\n**Earned:** +${quest.xp} XP\n\n**Your Stats:**\n• Total XP: ${userData.totalXP}\n• Quests Completed: ${userData.completedQuests}\n• Level: ${Math.floor(userData.totalXP / 100) + 1}`
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'leaderboard') {
      const leaderboard = Object.entries(questData)
        .map(([userId, data]) => ({
          userId,
          username: data.username,
          totalXP: data.totalXP,
          completed: data.completedQuests
        }))
        .sort((a, b) => b.totalXP - a.totalXP)
        .slice(0, 10);

      if (leaderboard.length === 0) {
        const embed = funEmbed(
          '📊 Quest Leaderboard',
          'No one has completed any quests yet! Use `/quest get` to start.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      let leaderboardText = '';
      leaderboard.forEach((player, index) => {
        const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
        const level = Math.floor(player.totalXP / 100) + 1;
        leaderboardText += `${medal} **${player.username}** - Lvl ${level} (${player.totalXP} XP, ${player.completed} quests)\n`;
      });

      const embed = funEmbed('📊 Quest Leaderboard', leaderboardText);
      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'stats') {
      const level = Math.floor(userData.totalXP / 100) + 1;
      const xpToNextLevel = (level * 100) - userData.totalXP;

      const embed = funEmbed(
        `📊 ${interaction.user.username}'s Quest Stats`,
        `**Level:** ${level}\n**Total XP:** ${userData.totalXP}\n**XP to Next Level:** ${xpToNextLevel}\n**Quests Completed:** ${userData.completedQuests}${userData.currentQuest ? `\n\n**Current Quest:**\n${userData.currentQuest.task}\n*Difficulty: ${difficultyEmojis[userData.currentQuest.difficulty]} ${userData.currentQuest.difficulty}*` : '\n\n*No active quest*'}`
      )
        .setThumbnail(interaction.user.displayAvatarURL({ dynamic: true }));

      await interaction.reply({ embeds: [embed] });
    }
  }
};
