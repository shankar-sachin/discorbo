/**
 * User Info Command
 * Display detailed information about a user
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('userinfo')
    .setDescription('Display detailed information about a user')
    .addUserOption(option =>
      option.setName('user')
        .setDescription('The user to get information about')
        .setRequired(false)),

  cooldown: 3,

  async execute(interaction) {
    const user = interaction.options.getUser('user') || interaction.user;
    const member = await interaction.guild.members.fetch(user.id).catch(() => null);

    const embed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle(`User Information: ${user.tag}`)
      .setThumbnail(user.displayAvatarURL({ dynamic: true }))
      .addFields(
        { name: 'Username', value: user.username, inline: true },
        { name: 'User ID', value: user.id, inline: true },
        { name: 'Bot Account', value: user.bot ? 'Yes' : 'No', inline: true },
        { name: 'Account Created', value: `<t:${Math.floor(user.createdTimestamp / 1000)}:F>`, inline: false },
        { name: 'Created', value: `<t:${Math.floor(user.createdTimestamp / 1000)}:R>`, inline: true }
      )
      .setTimestamp()
      .setFooter({ text: 'Discorbo • User Information' });

    // Add member-specific information if available
    if (member) {
      embed.addFields(
        { name: 'Joined Server', value: `<t:${Math.floor(member.joinedTimestamp / 1000)}:F>`, inline: false },
        { name: 'Join Position', value: `#${getJoinPosition(interaction.guild, member)}`, inline: true },
        { name: 'Roles', value: member.roles.cache.size > 1 ? member.roles.cache.filter(r => r.id !== interaction.guild.id).map(r => r).join(', ') : 'None', inline: false }
      );

      if (member.nickname) {
        embed.addFields({ name: 'Nickname', value: member.nickname, inline: true });
      }
    }

    await interaction.reply({ embeds: [embed] });
  }
};

/**
 * Get the user's join position in the server
 */
function getJoinPosition(guild, member) {
  const members = Array.from(guild.members.cache.values());
  members.sort((a, b) => a.joinedTimestamp - b.joinedTimestamp);
  return members.findIndex(m => m.id === member.id) + 1;
}
