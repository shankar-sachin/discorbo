/**
 * Remind Command
 * Set reminders that the bot will DM you about
 */

const { SlashCommandBuilder } = require('discord.js');
const { successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');
const config = require('../../config');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('remind')
    .setDescription('Set a reminder (bot will DM you)')
    .addStringOption(option =>
      option.setName('time')
        .setDescription('Time (e.g., 30m, 2h, 1d)')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('message')
        .setDescription('What to remind you about')
        .setRequired(true)),

  cooldown: 5,

  async execute(interaction) {
    const timeString = interaction.options.getString('time');
    const message = interaction.options.getString('message');

    // Parse time string
    const duration = parseTime(timeString);

    if (!duration || duration <= 0) {
      const embed = errorEmbed(
        'Invalid Time Format',
        'Please use a valid time format:\n• `30s` - 30 seconds\n• `5m` - 5 minutes\n• `2h` - 2 hours\n• `1d` - 1 day'
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    // Maximum reminder time: 30 days
    const maxDuration = 30 * 24 * 60 * 60 * 1000;
    if (duration > maxDuration) {
      const embed = errorEmbed(
        'Time Too Long',
        'Maximum reminder time is 30 days.'
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    // Check user's reminder limit
    const reminders = readJSON('reminders.json');
    const userReminders = reminders.filter(r => r.userId === interaction.user.id);

    if (userReminders.length >= config.limits.reminderMax) {
      const embed = errorEmbed(
        'Reminder Limit Reached',
        `You have reached the maximum of ${config.limits.reminderMax} active reminders.\nUse \`/reminders\` to manage your reminders.`
      );

      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    // Create reminder
    const reminder = {
      userId: interaction.user.id,
      message: message,
      createdAt: Date.now(),
      dueTime: Date.now() + duration,
      guildId: interaction.guild?.id || null
    };

    reminders.push(reminder);
    writeJSON('reminders.json', reminders);

    // Calculate relative time
    const dueTimestamp = Math.floor(reminder.dueTime / 1000);

    const embed = successEmbed(
      'Reminder Set!',
      `I'll remind you <t:${dueTimestamp}:R>\n**Message:** ${message}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};

/**
 * Parse time string to milliseconds
 */
function parseTime(timeString) {
  const regex = /^(\d+)([smhd])$/;
  const match = timeString.toLowerCase().match(regex);

  if (!match) return null;

  const value = parseInt(match[1]);
  const unit = match[2];

  const multipliers = {
    s: 1000,           // seconds
    m: 60 * 1000,      // minutes
    h: 60 * 60 * 1000, // hours
    d: 24 * 60 * 60 * 1000 // days
  };

  return value * (multipliers[unit] || 0);
}
