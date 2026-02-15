/**
 * Boss Raid Command
 * Server-wide boss battle where everyone can attack
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

const bosses = [
  {
    name: 'The Lag Monster',
    emoji: '👾',
    maxHP: 5000,
    description: 'A horrifying creature that feeds on your WiFi connection',
    loot: ['Fiber Optic Cable', 'Router of Legends', 'Ping Reducer', '5G Crystal']
  },
  {
    name: 'The Procrastination Demon',
    emoji: '😈',
    maxHP: 6000,
    description: 'An ancient evil that steals motivation and productivity',
    loot: ['Infinite Motivation', 'Time Management Scroll', 'Focus Potion', 'Deadline Destroyer']
  },
  {
    name: 'Captain Cringe',
    emoji: '🤡',
    maxHP: 4500,
    description: 'Keeper of embarrassing memories from middle school',
    loot: ['Memory Eraser', 'Confidence Boost', 'Second Chance Token', 'Cringe Shield']
  },
  {
    name: 'The Monday Overlord',
    emoji: '📅',
    maxHP: 7000,
    description: 'The most feared entity in existence',
    loot: ['Weekend Extension', 'Coffee of Awakening', 'Skip Monday Pass', 'Motivation Elixir']
  },
  {
    name: 'Social Anxiety Dragon',
    emoji: '🐉',
    maxHP: 5500,
    description: 'Guards the treasure of social skills',
    loot: ['Conversation Starter Kit', 'Confidence Amulet', 'Eye Contact Training', 'Small Talk Guide']
  },
  {
    name: 'The Imposter Syndrome Wraith',
    emoji: '👻',
    maxHP: 6500,
    description: 'Whispers doubts into the minds of the talented',
    loot: ['Self-Esteem Boost', 'Achievement Validator', 'Skill Recognition', 'Worth Affirmer']
  }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('bossraid')
    .setDescription('Participate in the server-wide boss raid!')
    .addSubcommand(subcommand =>
      subcommand
        .setName('status')
        .setDescription('Check the current boss status'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('attack')
        .setDescription('Attack the boss!'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('spawn')
        .setDescription('Spawn a new boss (Admin only)'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('leaderboard')
        .setDescription('View raid damage leaderboard')),

  category: 'fun',
  cooldown: 0, // No cooldown for status/leaderboard

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const bossData = readJSON('bossraid.json');
    const guildId = interaction.guild.id;

    if (!bossData[guildId]) {
      bossData[guildId] = {
        boss: null,
        participants: {},
        spawnedAt: null,
        totalDamage: 0
      };
    }

    const raidData = bossData[guildId];

    if (subcommand === 'status') {
      if (!raidData.boss) {
        const embed = errorEmbed(
          'No Active Boss',
          'There is no boss currently active! An admin can spawn one with `/bossraid spawn`.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      const boss = raidData.boss;
      const hpPercentage = (boss.currentHP / boss.maxHP) * 100;
      const hpBar = createHPBar(boss.currentHP, boss.maxHP);

      const participantCount = Object.keys(raidData.participants).length;
      const topDamagers = Object.entries(raidData.participants)
        .sort((a, b) => b[1].damage - a[1].damage)
        .slice(0, 3);

      let topDamagersText = '';
      topDamagers.forEach(([userId, data], index) => {
        const medal = ['🥇', '🥈', '🥉'][index];
        topDamagersText += `${medal} ${data.username}: ${data.damage.toLocaleString()} damage\n`;
      });

      const embed = new EmbedBuilder()
        .setColor(hpPercentage < 25 ? 0xFF0000 : hpPercentage < 50 ? 0xFFA500 : 0x00FF00)
        .setTitle(`${boss.emoji} ${boss.name}`)
        .setDescription(`*${boss.description}*\n\n**HP:** ${hpBar}\n${boss.currentHP.toLocaleString()} / ${boss.maxHP.toLocaleString()} (${hpPercentage.toFixed(1)}%)\n\n**Participants:** ${participantCount}\n**Total Damage:** ${raidData.totalDamage.toLocaleString()}`)
        .setTimestamp()
        .setFooter({ text: 'Use /bossraid attack to deal damage!' });

      if (topDamagersText) {
        embed.addFields({ name: '🏆 Top Damage Dealers', value: topDamagersText, inline: false });
      }

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'attack') {
      if (!raidData.boss) {
        const embed = errorEmbed(
          'No Active Boss',
          'There is no boss currently active! An admin can spawn one with `/bossraid spawn`.'
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      if (raidData.boss.currentHP <= 0) {
        const embed = errorEmbed(
          'Boss Already Defeated',
          'This boss has already been defeated! Wait for the next one to spawn.'
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Check cooldown (5 minutes per user)
      if (!raidData.participants[interaction.user.id]) {
        raidData.participants[interaction.user.id] = {
          username: interaction.user.username,
          damage: 0,
          lastAttack: 0,
          attacks: 0
        };
      }

      const userData = raidData.participants[interaction.user.id];
      const now = Date.now();
      const cooldownTime = 5 * 60 * 1000; // 5 minutes

      if (now - userData.lastAttack < cooldownTime) {
        const timeLeft = Math.ceil((cooldownTime - (now - userData.lastAttack)) / 1000 / 60);
        const embed = errorEmbed(
          'Attack Cooldown',
          `You can attack again in ${timeLeft} minute(s)!`
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Calculate damage (random 100-500)
      const damage = Math.floor(Math.random() * 401) + 100;
      const isCrit = Math.random() < 0.15; // 15% crit chance
      const finalDamage = isCrit ? Math.floor(damage * 2) : damage;

      // Apply damage
      raidData.boss.currentHP = Math.max(0, raidData.boss.currentHP - finalDamage);
      userData.damage += finalDamage;
      userData.lastAttack = now;
      userData.attacks++;
      userData.username = interaction.user.username;
      raidData.totalDamage += finalDamage;

      const boss = raidData.boss;
      const defeated = raidData.boss.currentHP <= 0;

      writeJSON('bossraid.json', bossData);

      if (defeated) {
        // Boss defeated - distribute loot
        const lootItem = boss.loot[Math.floor(Math.random() * boss.loot.length)];

        const embed = successEmbed(
          `💀 ${boss.name} Defeated!`,
          `${interaction.user} dealt the final blow with ${finalDamage.toLocaleString()} damage!${isCrit ? ' 💥 **CRITICAL HIT!**' : ''}\n\n**Raid Complete!**\n• Total Damage: ${raidData.totalDamage.toLocaleString()}\n• Participants: ${Object.keys(raidData.participants).length}\n\n🎁 **Loot Drop:** ${lootItem}\n\nAll participants have received rewards!`
        );

        // Give rewards to all participants
        const rewardsData = readJSON('daily-rewards.json');
        Object.entries(raidData.participants).forEach(([userId, data]) => {
          if (!rewardsData[userId]) {
            rewardsData[userId] = { username: data.username, coins: 0, bossKills: 0 };
          }
          const reward = Math.floor(data.damage / 10); // 1 coin per 10 damage
          rewardsData[userId].coins += reward;
          rewardsData[userId].bossKills++;
          rewardsData[userId].username = data.username;
        });
        writeJSON('daily-rewards.json', rewardsData);

        await interaction.reply({ embeds: [embed] });
      } else {
        const hpBar = createHPBar(boss.currentHP, boss.maxHP);

        const embed = funEmbed(
          `⚔️ ${interaction.user.username} Attacks!`,
          `Dealt **${finalDamage.toLocaleString()}** damage!${isCrit ? ' 💥 **CRITICAL HIT!**' : ''}\n\n**${boss.emoji} ${boss.name}**\n${hpBar}\n${boss.currentHP.toLocaleString()} / ${boss.maxHP.toLocaleString()} HP\n\n**Your Stats:**\n• Total Damage: ${userData.damage.toLocaleString()}\n• Attacks: ${userData.attacks}`
        );

        await interaction.reply({ embeds: [embed] });
      }
    }

    else if (subcommand === 'spawn') {
      // Check if user is admin
      if (!interaction.member.permissions.has('Administrator')) {
        const embed = errorEmbed(
          'Permission Denied',
          'Only administrators can spawn bosses!'
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Check if boss already active
      if (raidData.boss && raidData.boss.currentHP > 0) {
        const embed = errorEmbed(
          'Boss Already Active',
          `${raidData.boss.emoji} **${raidData.boss.name}** is still alive with ${raidData.boss.currentHP.toLocaleString()} HP!`
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Spawn new boss
      const bossTemplate = bosses[Math.floor(Math.random() * bosses.length)];
      raidData.boss = {
        ...bossTemplate,
        currentHP: bossTemplate.maxHP,
        spawnedAt: Date.now()
      };
      raidData.participants = {};
      raidData.totalDamage = 0;

      writeJSON('bossraid.json', bossData);

      const embed = funEmbed(
        '🚨 BOSS RAID STARTED! 🚨',
        `${bossTemplate.emoji} **${bossTemplate.name}** has appeared!\n\n*${bossTemplate.description}*\n\n**HP:** ${bossTemplate.maxHP.toLocaleString()}\n**Possible Loot:** ${bossTemplate.loot.join(', ')}\n\nUse \`/bossraid attack\` to join the fight!`
      )
        .setColor(0xFF0000);

      await interaction.reply({ content: '@everyone', embeds: [embed] });
    }

    else if (subcommand === 'leaderboard') {
      if (!raidData.boss) {
        const embed = errorEmbed(
          'No Active Boss',
          'There is no boss currently active!'
        );
        return interaction.reply({ embeds: [embed] });
      }

      const leaderboard = Object.entries(raidData.participants)
        .map(([userId, data]) => ({ userId, ...data }))
        .sort((a, b) => b.damage - a.damage)
        .slice(0, 10);

      if (leaderboard.length === 0) {
        const embed = funEmbed(
          '📊 Raid Leaderboard',
          'No one has attacked yet! Use `/bossraid attack` to join the fight.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      let leaderboardText = '';
      leaderboard.forEach((player, index) => {
        const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
        leaderboardText += `${medal} **${player.username}** - ${player.damage.toLocaleString()} damage (${player.attacks} attacks)\n`;
      });

      const embed = funEmbed(
        `📊 ${raidData.boss.name} - Damage Leaderboard`,
        leaderboardText
      );

      await interaction.reply({ embeds: [embed] });
    }
  }
};

function createHPBar(current, max) {
  const percentage = current / max;
  const barLength = 20;
  const filled = Math.ceil(barLength * percentage);

  let emoji = '🟢';
  if (percentage < 0.5) emoji = '🟡';
  if (percentage < 0.25) emoji = '🔴';

  return emoji.repeat(Math.max(0, filled)) + '⚫'.repeat(Math.max(0, barLength - filled));
}
