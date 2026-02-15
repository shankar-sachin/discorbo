/**
 * Poll Command
 * Create a reaction-based poll
 */

const { SlashCommandBuilder } = require('discord.js');
const { infoEmbed } = require('../../utils/embedBuilder');

// Number emojis for poll options
const numberEmojis = ['1️⃣', '2️⃣', '3️⃣', '4️⃣', '5️⃣', '6️⃣', '7️⃣', '8️⃣', '9️⃣', '🔟'];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('poll')
    .setDescription('Create a poll with up to 10 options')
    .addStringOption(option =>
      option.setName('question')
        .setDescription('The poll question')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('option1')
        .setDescription('First option')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('option2')
        .setDescription('Second option')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('option3')
        .setDescription('Third option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option4')
        .setDescription('Fourth option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option5')
        .setDescription('Fifth option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option6')
        .setDescription('Sixth option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option7')
        .setDescription('Seventh option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option8')
        .setDescription('Eighth option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option9')
        .setDescription('Ninth option')
        .setRequired(false))
    .addStringOption(option =>
      option.setName('option10')
        .setDescription('Tenth option')
        .setRequired(false)),

  cooldown: 5,

  async execute(interaction) {
    const question = interaction.options.getString('question');

    // Collect all provided options
    const options = [];
    for (let i = 1; i <= 10; i++) {
      const option = interaction.options.getString(`option${i}`);
      if (option) {
        options.push(option);
      }
    }

    // Build poll description
    let description = '';
    options.forEach((option, index) => {
      description += `${numberEmojis[index]} ${option}\n`;
    });

    const embed = infoEmbed(question, description)
      .setFooter({ text: `Poll by ${interaction.user.tag}` });

    // Send poll
    await interaction.reply({ embeds: [embed] });

    // Get the message to add reactions
    const message = await interaction.fetchReply();

    // Add reaction for each option
    for (let i = 0; i < options.length; i++) {
      try {
        await message.react(numberEmojis[i]);
      } catch (error) {
        console.error('Failed to add reaction:', error);
      }
    }
  }
};
