/**
 * Reminder Checker Utility
 * Checks for due reminders and sends DMs to users
 */

const { readJSON, writeJSON } = require('./dataManager');
const { infoEmbed } = require('./embedBuilder');
const logger = require('./logger');

/**
 * Check and send due reminders
 */
async function checkReminders(client) {
  try {
    const reminders = readJSON('reminders.json');

    if (!Array.isArray(reminders) || reminders.length === 0) {
      return;
    }

    const now = Date.now();
    const dueReminders = [];
    const remainingReminders = [];

    // Separate due and remaining reminders
    reminders.forEach(reminder => {
      if (reminder.dueTime <= now) {
        dueReminders.push(reminder);
      } else {
        remainingReminders.push(reminder);
      }
    });

    // Send due reminders
    for (const reminder of dueReminders) {
      try {
        const user = await client.users.fetch(reminder.userId);

        if (user) {
          const embed = infoEmbed(
            'Reminder!',
            `**Message:** ${reminder.message}\n\n*Set <t:${Math.floor(reminder.createdAt / 1000)}:R>*`
          );

          await user.send({ embeds: [embed] });
          logger.success(`Sent reminder to ${user.tag}: ${reminder.message}`);
        }
      } catch (error) {
        logger.error(`Failed to send reminder to user ${reminder.userId}:`, error.message);
      }
    }

    // Update reminders file (remove completed ones)
    if (dueReminders.length > 0) {
      writeJSON('reminders.json', remainingReminders);
      logger.info(`Processed ${dueReminders.length} reminder(s)`);
    }

  } catch (error) {
    logger.error('Error in checkReminders:', error);
  }
}

module.exports = {
  checkReminders
};
