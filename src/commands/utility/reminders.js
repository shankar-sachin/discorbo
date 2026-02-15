/**
 * Reminders Command
 * List and manage your active reminders
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { readJSON, writeJSON } = require('../../utils/dataManager');
const { infoEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('reminders')
    .setDescription('Manage your active reminders')
    .addSubcommand(subcommand =>
      subcommand
        .setName('list')
        .setDescription('View all your active reminders'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('clear')
        .setDescription('Delete all your reminders'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('remove')
        .setDescription('Remove a specific reminder')
        .addIntegerOption(option =>
          option.setName('number')
            .setDescription('Reminder number to remove (use /reminders list to see numbers)')
            .setRequired(true)
            .setMinValue(1))),

  cooldown: 5,

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const reminders = readJSON('reminders.json');
    const userReminders = reminders.filter(r => r.userId === interaction.user.id);

    if (subcommand === 'list') {
      if (userReminders.length === 0) {
        const embed = infoEmbed(
          'No Reminders',
          'You don\'t have any active reminders.\nUse `/remind` to create one!'
        );

        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      const embed = new EmbedBuilder()
        .setColor(0x5865F2)
        .setTitle('Your Reminders')
        .setDescription(`You have ${userReminders.length} active reminder(s)`)
        .setTimestamp()
        .setFooter({ text: 'Use /reminders remove <number> to delete a reminder' });

      userReminders.forEach((reminder, index) => {
        const dueTimestamp = Math.floor(reminder.dueTime / 1000);
        embed.addFields({
          name: `${index + 1}. ${reminder.message}`,
          value: `Due: <t:${dueTimestamp}:R> (<t:${dueTimestamp}:F>)`,
          inline: false
        });
      });

      await interaction.reply({ embeds: [embed], ephemeral: true });
    }

    else if (subcommand === 'clear') {
      if (userReminders.length === 0) {
        const embed = infoEmbed(
          'No Reminders',
          'You don\'t have any active reminders to clear.'
        );

        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Remove all user reminders
      const updatedReminders = reminders.filter(r => r.userId !== interaction.user.id);
      writeJSON('reminders.json', updatedReminders);

      const embed = successEmbed(
        'Reminders Cleared',
        `Deleted ${userReminders.length} reminder(s).`
      );

      await interaction.reply({ embeds: [embed], ephemeral: true });
    }

    else if (subcommand === 'remove') {
      const number = interaction.options.getInteger('number');

      if (number > userReminders.length) {
        const embed = errorEmbed(
          'Invalid Reminder',
          `You only have ${userReminders.length} reminder(s).\nUse \`/reminders list\` to see your reminders.`
        );

        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      // Get the reminder to remove
      const reminderToRemove = userReminders[number - 1];

      // Remove from all reminders
      const updatedReminders = reminders.filter(r =>
        !(r.userId === reminderToRemove.userId &&
          r.createdAt === reminderToRemove.createdAt &&
          r.message === reminderToRemove.message)
      );

      writeJSON('reminders.json', updatedReminders);

      const embed = successEmbed(
        'Reminder Deleted',
        `Removed reminder: **${reminderToRemove.message}**`
      );

      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  }
};
