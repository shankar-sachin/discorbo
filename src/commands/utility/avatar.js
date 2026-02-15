/**
 * Avatar Command
 * Display a user's avatar in high resolution
 */

const { SlashCommandBuilder } = require('discord.js');
const { infoEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('avatar')
    .setDescription('Display a user\'s avatar')
    .addUserOption(option =>
      option.setName('user')
        .setDescription('The user whose avatar to display')
        .setRequired(false)),

  cooldown: 3,

  async execute(interaction) {
    const user = interaction.options.getUser('user') || interaction.user;

    const embed = infoEmbed(
      `${user.username}'s Avatar`,
      `[Download](${user.displayAvatarURL({ size: 4096 })})`
    )
      .setImage(user.displayAvatarURL({ dynamic: true, size: 1024 }));

    await interaction.reply({ embeds: [embed] });
  }
};
