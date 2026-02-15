/**
 * Server Info Command
 * Display detailed information about the server
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('serverinfo')
    .setDescription('Display information about this server'),

  cooldown: 5,

  async execute(interaction) {
    const { guild } = interaction;

    // Fetch owner
    const owner = await guild.fetchOwner().catch(() => null);

    // Count channels by type
    const textChannels = guild.channels.cache.filter(c => c.type === 0).size;
    const voiceChannels = guild.channels.cache.filter(c => c.type === 2).size;
    const categories = guild.channels.cache.filter(c => c.type === 4).size;

    // Member statistics
    const members = guild.members.cache;
    const humans = members.filter(m => !m.user.bot).size;
    const bots = members.filter(m => m.user.bot).size;

    // Boost information
    const boostLevel = guild.premiumTier || 0;
    const boostCount = guild.premiumSubscriptionCount || 0;

    const embed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle(guild.name)
      .setThumbnail(guild.iconURL({ dynamic: true, size: 1024 }))
      .addFields(
        { name: 'Server ID', value: guild.id, inline: true },
        { name: 'Owner', value: owner ? `${owner.user.tag}` : 'Unknown', inline: true },
        { name: 'Created', value: `<t:${Math.floor(guild.createdTimestamp / 1000)}:R>`, inline: true },
        { name: 'Members', value: `👥 ${guild.memberCount}\n👤 ${humans} Humans\n🤖 ${bots} Bots`, inline: true },
        { name: 'Channels', value: `💬 ${textChannels} Text\n🔊 ${voiceChannels} Voice\n📁 ${categories} Categories`, inline: true },
        { name: 'Boosts', value: `Level ${boostLevel}\n${boostCount} boosts`, inline: true },
        { name: 'Roles', value: `${guild.roles.cache.size} roles`, inline: true },
        { name: 'Emojis', value: `${guild.emojis.cache.size} emojis`, inline: true },
        { name: 'Verification Level', value: getVerificationLevel(guild.verificationLevel), inline: true }
      )
      .setTimestamp()
      .setFooter({ text: 'Discorbo • Server Information' });

    if (guild.description) {
      embed.setDescription(guild.description);
    }

    if (guild.banner) {
      embed.setImage(guild.bannerURL({ size: 1024 }));
    }

    await interaction.reply({ embeds: [embed] });
  }
};

/**
 * Convert verification level number to readable string
 */
function getVerificationLevel(level) {
  const levels = {
    0: 'None',
    1: 'Low',
    2: 'Medium',
    3: 'High',
    4: 'Very High'
  };
  return levels[level] || 'Unknown';
}
