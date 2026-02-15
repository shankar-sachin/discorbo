/**
 * 8ball Command
 * Magic 8-ball predictions
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const responses = [
  // Affirmative
  'It is certain.',
  'It is decidedly so.',
  'Without a doubt.',
  'Yes definitely.',
  'You may rely on it.',
  'As I see it, yes.',
  'Most likely.',
  'Outlook good.',
  'Yes.',
  'Signs point to yes.',

  // Non-committal
  'Reply hazy, try again.',
  'Ask again later.',
  'Better not tell you now.',
  'Cannot predict now.',
  'Concentrate and ask again.',

  // Negative
  'Don\'t count on it.',
  'My reply is no.',
  'My sources say no.',
  'Outlook not so good.',
  'Very doubtful.'
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('8ball')
    .setDescription('Ask the magic 8-ball a question')
    .addStringOption(option =>
      option.setName('question')
        .setDescription('Your yes/no question')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const question = interaction.options.getString('question');
    const response = responses[Math.floor(Math.random() * responses.length)];

    const embed = funEmbed(
      '🎱 Magic 8-Ball',
      `**Question:** ${question}\n**Answer:** ${response}`
    );

    await interaction.reply({ embeds: [embed] });
  }
};
