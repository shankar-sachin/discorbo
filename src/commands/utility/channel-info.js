/**
 * Channel Info Command
 * Display detailed information about a channel
 */

const { SlashCommandBuilder, EmbedBuilder, ChannelType } = require('discord.js');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('channel-info')
    .setDescription('Display detailed information about a channel')
    .addChannelOption(option =>
      option.setName('channel')
        .setDescription('The channel to get information about')
        .setRequired(false)),

  cooldown: 3,

  async execute(interaction) {
    const channel = interaction.options.getChannel('channel') || interaction.channel;

    const embed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle(`Channel Information: #${channel.name}`)
      .addFields(
        { name: 'Channel ID', value: channel.id, inline: true },
        { name: 'Type', value: getChannelType(channel.type), inline: true },
        { name: 'Created', value: `<t:${Math.floor(channel.createdTimestamp / 1000)}:R>`, inline: true }
      )
      .setTimestamp()
      .setFooter({ text: 'Discorbo • Channel Information' });

    // Add description if available
    if (channel.topic) {
      embed.addFields({ name: 'Description', value: channel.topic, inline: false });
    }

    // Add text channel specific info
    if (channel.type === ChannelType.GuildText) {
      embed.addFields(
        { name: 'NSFW', value: channel.nsfw ? 'Yes' : 'No', inline: true },
        { name: 'Slowmode', value: channel.rateLimitPerUser ? `${channel.rateLimitPerUser}s` : 'Disabled', inline: true }
      );
    }

    // Add voice channel specific info
    if (channel.type === ChannelType.GuildVoice) {
      embed.addFields(
        { name: 'Bitrate', value: `${channel.bitrate / 1000}kbps`, inline: true },
        { name: 'User Limit', value: channel.userLimit || 'Unlimited', inline: true }
      );
    }

    // Add category if channel is in one
    if (channel.parent) {
      embed.addFields({ name: 'Category', value: channel.parent.name, inline: true });
    }

    // Add position
    embed.addFields({ name: 'Position', value: `${channel.position}`, inline: true });

    await interaction.reply({ embeds: [embed] });
  }
};

/**
 * Convert channel type to readable string
 */
function getChannelType(type) {
  const types = {
    0: 'Text Channel',
    2: 'Voice Channel',
    4: 'Category',
    5: 'Announcement Channel',
    10: 'Announcement Thread',
    11: 'Public Thread',
    12: 'Private Thread',
    13: 'Stage Channel',
    15: 'Forum Channel'
  };
  return types[type] || 'Unknown';
}
