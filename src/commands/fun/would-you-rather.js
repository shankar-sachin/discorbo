/**
 * Would You Rather Command
 * Present random "would you rather" questions
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const questions = [
  { option1: 'Have the ability to fly', option2: 'Have the ability to be invisible' },
  { option1: 'Live without music', option2: 'Live without movies' },
  { option1: 'Be able to speak all languages', option2: 'Be able to talk to animals' },
  { option1: 'Have unlimited money', option2: 'Have unlimited time' },
  { option1: 'Always be 10 minutes late', option2: 'Always be 20 minutes early' },
  { option1: 'Have no Internet', option2: 'Have no phone' },
  { option1: 'Live in the past', option2: 'Live in the future' },
  { option1: 'Never eat your favorite food again', option2: 'Only eat your favorite food' },
  { option1: 'Be able to read minds', option2: 'Be able to see the future' },
  { option1: 'Have a rewind button', option2: 'Have a pause button' },
  { option1: 'Be famous', option2: 'Be the best friend of someone famous' },
  { option1: 'Live without air conditioning', option2: 'Live without heating' },
  { option1: 'Have a personal chef', option2: 'Have a personal driver' },
  { option1: 'Fight one horse-sized duck', option2: 'Fight 100 duck-sized horses' },
  { option1: 'Never have to sleep', option2: 'Never have to eat' },
  { option1: 'Be able to teleport', option2: 'Be able to time travel' },
  { option1: 'Live in space', option2: 'Live underwater' },
  { option1: 'Have super strength', option2: 'Have super speed' },
  { option1: 'Give up social media', option2: 'Give up streaming services' },
  { option1: 'Know how you die', option2: 'Know when you die' },
  { option1: 'Have unlimited battery life', option2: 'Have unlimited storage' },
  { option1: 'Be always too hot', option2: 'Be always too cold' },
  { option1: 'Have the ability to change the past', option2: 'Have the ability to see into the future' },
  { option1: 'Lose your sense of taste', option2: 'Lose your sense of smell' },
  { option1: 'Have a photographic memory', option2: 'Have an IQ of 200' }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('would-you-rather')
    .setDescription('Get a "would you rather" question'),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const question = questions[Math.floor(Math.random() * questions.length)];

    const embed = funEmbed(
      '🤔 Would You Rather...',
      `**Option 1:** ${question.option1}\n\n**OR**\n\n**Option 2:** ${question.option2}`
    );

    await interaction.reply({ embeds: [embed] });

    // Add reaction options
    const message = await interaction.fetchReply();
    await message.react('1️⃣');
    await message.react('2️⃣');
  }
};
