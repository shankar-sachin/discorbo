/**
 * Ping Command
 * Displays bot latency and API ping
 */

const { SlashCommandBuilder } = require('discord.js');
const { infoEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('ping')
    .setDescription('Check the bot\'s latency and response time'),

  cooldown: 3,

  async execute(interaction) {
    const sent = await interaction.reply({ content: 'Pinging...', fetchReply: true });

    const botLatency = sent.createdTimestamp - interaction.createdTimestamp;
    const apiLatency = Math.round(interaction.client.ws.ping);

    const embed = infoEmbed(
      'Pong!',
      `**Bot Latency:** ${botLatency}ms\n**API Latency:** ${apiLatency}ms`
    );

    await interaction.editReply({ content: null, embeds: [embed] });
  }
};
