/**
 * Loot Command
 * Random loot drops with inventory system
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed, successEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

const lootTables = {
  common: {
    name: 'Common',
    emoji: '⚪',
    color: 0x808080,
    items: [
      'Slightly Used Tissue',
      'Rock (it\'s a good rock though)',
      'Half-Eaten Sandwich',
      'Broken Pencil',
      'Mystery Stain',
      'Expired Coupon',
      'Lint Collection',
      'Old Receipt',
      'Bottle Cap',
      'Rubber Band Ball'
    ]
  },
  uncommon: {
    name: 'Uncommon',
    emoji: '🟢',
    color: 0x00FF00,
    items: [
      'Functional Pen',
      'Matching Socks',
      'Working Charger',
      'Fresh Pizza Slice',
      'Clean Towel',
      'Organized Calendar',
      'Comfortable Pillow',
      'Warm Blanket',
      'Good Vibes',
      'Motivational Quote'
    ]
  },
  rare: {
    name: 'Rare',
    emoji: '🔵',
    color: 0x0099FF,
    items: [
      'Full Night\'s Sleep',
      'Productive Day',
      'Compliment from Stranger',
      'Green Light Streak',
      'Extra Fries in Bag',
      'Perfect Temperature Shower',
      'No Ads YouTube Video',
      'Both AirPods Charged',
      'Instant Coffee Perfection',
      'Reply from Crush'
    ]
  },
  epic: {
    name: 'Epic',
    emoji: '🟣',
    color: 0x9B30FF,
    items: [
      'Cancelled Monday Meeting',
      'WiFi That Actually Works',
      'Perfect Parallel Park',
      'Empty Gym',
      'First Try Password Success',
      'No Line at DMV',
      'Sale on Thing You Wanted',
      'Inbox Zero Achievement',
      'Legendary Comeback',
      'Main Character Moment'
    ]
  },
  legendary: {
    name: 'Legendary',
    emoji: '🟡',
    color: 0xFFD700,
    items: [
      'Instant Ramen That Tastes Gourmet',
      'USB Inserted Correctly First Try',
      'Extra Day Weekend',
      'Unlimited Garlic Bread',
      'Perfect Hair Day Forever',
      'Universal Remote',
      'Self-Cleaning Room',
      'Automatic GPA Boost',
      'Infinite Battery Life',
      'Social Anxiety Delete Button'
    ]
  },
  cosmic: {
    name: 'Cosmic',
    emoji: '🌌',
    color: 0xFF00FF,
    items: [
      'The Ability to Pause Time',
      'Universal Healthcare (US Edition)',
      'World Peace Token',
      'Climate Change Reversal',
      'Infinite Pizza Generator',
      'Student Loan Forgiveness Stone',
      'Motivation Potion (Permanent)',
      'Delete Cringe Memories Wand',
      'Respec Button for Life Choices',
      'Touch Grass Immunity'
    ]
  },
  cursed: {
    name: 'Cursed',
    emoji: '💀',
    color: 0x800080,
    items: [
      'Wet Socks (Permanent)',
      'Both Sides Hot Pillow',
      'Eternal Loading Screen',
      'Autocorrect That Never Stops',
      'Constant Minor Inconvenience',
      'Always Slow Lane',
      'Permanently Skipping Song',
      'Infinite Elevator Music',
      'Unstoppable Hiccups',
      'Forever Crumbs in Bed'
    ]
  }
};

module.exports = {
  data: new SlashCommandBuilder()
    .setName('loot')
    .setDescription('Open a loot chest and see what you get!')
    .addSubcommand(subcommand =>
      subcommand
        .setName('open')
        .setDescription('Open a loot chest'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('inventory')
        .setDescription('View your inventory'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('stats')
        .setDescription('View your loot statistics')),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const lootData = readJSON('loot.json');

    if (!lootData[interaction.user.id]) {
      lootData[interaction.user.id] = {
        username: interaction.user.username,
        inventory: [],
        stats: {
          common: 0,
          uncommon: 0,
          rare: 0,
          epic: 0,
          legendary: 0,
          cosmic: 0,
          cursed: 0
        },
        totalLoots: 0
      };
    }

    const userData = lootData[interaction.user.id];
    userData.username = interaction.user.username;

    if (subcommand === 'open') {
      // Determine rarity
      const roll = Math.random() * 100;
      let rarity;

      if (roll < 35) rarity = 'common';
      else if (roll < 60) rarity = 'uncommon';
      else if (roll < 80) rarity = 'rare';
      else if (roll < 92) rarity = 'epic';
      else if (roll < 98) rarity = 'legendary';
      else if (roll < 99.5) rarity = 'cosmic';
      else rarity = 'cursed';

      const lootTable = lootTables[rarity];
      const item = lootTable.items[Math.floor(Math.random() * lootTable.items.length)];

      // Add to inventory
      userData.inventory.push({
        item,
        rarity,
        obtainedAt: Date.now()
      });

      userData.stats[rarity]++;
      userData.totalLoots++;

      writeJSON('loot.json', lootData);

      // Create dramatic reveal
      const embed = funEmbed(
        '📦 Loot Chest Opening...',
        `You obtained:\n\n${lootTable.emoji} **${lootTable.name}**\n**${item}**\n\n*Total Loot: ${userData.totalLoots}*`
      )
        .setColor(lootTable.color);

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'inventory') {
      if (userData.inventory.length === 0) {
        const embed = funEmbed(
          '🎒 Empty Inventory',
          'You haven\'t collected any loot yet! Use `/loot open` to get started.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      // Show last 10 items
      const recentLoot = userData.inventory.slice(-10).reverse();

      let inventoryText = '';
      recentLoot.forEach((loot, index) => {
        const lootTable = lootTables[loot.rarity];
        inventoryText += `${lootTable.emoji} **${loot.item}** (${lootTable.name})\n`;
      });

      if (userData.inventory.length > 10) {
        inventoryText += `\n*Showing last 10 of ${userData.inventory.length} items*`;
      }

      const embed = funEmbed(
        `🎒 ${interaction.user.username}'s Inventory`,
        inventoryText
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'stats') {
      const { stats, totalLoots } = userData;

      let statsText = `**Total Loot Opened:** ${totalLoots}\n\n**Rarity Breakdown:**\n`;

      Object.entries(lootTables).forEach(([rarity, data]) => {
        const count = stats[rarity] || 0;
        const percentage = totalLoots > 0 ? ((count / totalLoots) * 100).toFixed(1) : 0;
        statsText += `${data.emoji} ${data.name}: ${count} (${percentage}%)\n`;
      });

      // Find rarest item
      const rarestLoot = userData.inventory
        .filter(l => l.rarity === 'cosmic' || l.rarity === 'legendary')
        .slice(-1)[0];

      if (rarestLoot) {
        const lootTable = lootTables[rarestLoot.rarity];
        statsText += `\n**Rarest Item:**\n${lootTable.emoji} ${rarestLoot.item}`;
      }

      const embed = funEmbed(
        `📊 ${interaction.user.username}'s Loot Stats`,
        statsText
      )
        .setThumbnail(interaction.user.displayAvatarURL({ dynamic: true }));

      await interaction.reply({ embeds: [embed] });
    }
  }
};
