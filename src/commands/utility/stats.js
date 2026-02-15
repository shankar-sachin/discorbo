/**
 * Stats Command
 * Display bot statistics and performance metrics
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { readJSON } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('stats')
    .setDescription('Display bot statistics and metrics'),

  cooldown: 5,

  async execute(interaction) {
    const { client } = interaction;

    // Calculate uptime
    const uptime = process.uptime();
    const days = Math.floor(uptime / 86400);
    const hours = Math.floor((uptime % 86400) / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    const seconds = Math.floor(uptime % 60);

    const uptimeString = `${days}d ${hours}h ${minutes}m ${seconds}s`;

    // Memory usage
    const memoryUsage = process.memoryUsage();
    const memoryMB = (memoryUsage.heapUsed / 1024 / 1024).toFixed(2);

    // Count total users
    let totalUsers = 0;
    client.guilds.cache.forEach(guild => {
      totalUsers += guild.memberCount;
    });

    // Count total commands used
    let totalCommandsUsed = 0;
    try {
      const commandUsage = readJSON('command-usage.json');
      Object.values(commandUsage).forEach(userCommands => {
        Object.values(userCommands).forEach(count => {
          totalCommandsUsed += count;
        });
      });
    } catch (error) {
      // If file doesn't exist or error reading, just show 0
    }

    const embed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle('Bot Statistics')
      .setThumbnail(client.user.displayAvatarURL())
      .addFields(
        { name: 'Servers', value: `${client.guilds.cache.size}`, inline: true },
        { name: 'Users', value: `${totalUsers.toLocaleString()}`, inline: true },
        { name: 'Commands', value: `${client.commands.size}`, inline: true },
        { name: 'Uptime', value: uptimeString, inline: true },
        { name: 'Memory Usage', value: `${memoryMB} MB`, inline: true },
        { name: 'Ping', value: `${client.ws.ping}ms`, inline: true },
        { name: 'Commands Used', value: `${totalCommandsUsed.toLocaleString()}`, inline: true },
        { name: 'Node.js', value: process.version, inline: true },
        { name: 'Discord.js', value: `v${require('discord.js').version}`, inline: true }
      )
      .setTimestamp()
      .setFooter({ text: 'Discorbo • Bot Statistics' });

    await interaction.reply({ embeds: [embed] });
  }
};
