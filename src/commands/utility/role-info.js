/**
 * Role Info Command
 * Display detailed information about a role
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('role-info')
    .setDescription('Display detailed information about a role')
    .addRoleOption(option =>
      option.setName('role')
        .setDescription('The role to get information about')
        .setRequired(true)),

  cooldown: 3,

  async execute(interaction) {
    const role = interaction.options.getRole('role');

    // Get members with this role
    const membersWithRole = interaction.guild.members.cache.filter(m =>
      m.roles.cache.has(role.id)
    ).size;

    // Get position (inverted, higher number = higher in list)
    const position = interaction.guild.roles.cache.size - role.position;

    const embed = new EmbedBuilder()
      .setColor(role.color || 0x5865F2)
      .setTitle(`Role Information: ${role.name}`)
      .addFields(
        { name: 'Role ID', value: role.id, inline: true },
        { name: 'Color', value: role.hexColor, inline: true },
        { name: 'Position', value: `${position}`, inline: true },
        { name: 'Members', value: `${membersWithRole}`, inline: true },
        { name: 'Mentionable', value: role.mentionable ? 'Yes' : 'No', inline: true },
        { name: 'Hoisted', value: role.hoist ? 'Yes' : 'No', inline: true },
        { name: 'Created', value: `<t:${Math.floor(role.createdTimestamp / 1000)}:R>`, inline: false },
        { name: 'Mention', value: `${role}`, inline: true }
      )
      .setTimestamp()
      .setFooter({ text: 'Discorbo • Role Information' });

    // Add permissions if role has any
    if (role.permissions.toArray().length > 0) {
      const permissions = role.permissions.toArray()
        .map(p => p.replace(/([A-Z])/g, ' $1').trim())
        .join(', ');

      embed.addFields({
        name: 'Key Permissions',
        value: permissions.length > 1024 ? permissions.substring(0, 1021) + '...' : permissions,
        inline: false
      });
    }

    await interaction.reply({ embeds: [embed] });
  }
};
