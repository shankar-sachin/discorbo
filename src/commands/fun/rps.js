/**
 * Rock Paper Scissors Command
 * Play rock, paper, scissors against the bot
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const choices = ['rock', 'paper', 'scissors'];
const emojis = {
  rock: '🪨',
  paper: '📄',
  scissors: '✂️'
};

module.exports = {
  data: new SlashCommandBuilder()
    .setName('rps')
    .setDescription('Play rock, paper, scissors')
    .addStringOption(option =>
      option.setName('choice')
        .setDescription('Your choice')
        .setRequired(true)
        .addChoices(
          { name: '🪨 Rock', value: 'rock' },
          { name: '📄 Paper', value: 'paper' },
          { name: '✂️ Scissors', value: 'scissors' }
        )),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const userChoice = interaction.options.getString('choice');
    const botChoice = choices[Math.floor(Math.random() * choices.length)];

    // Determine winner
    let result = '';
    let color = 0xEB459E;

    if (userChoice === botChoice) {
      result = 'It\'s a tie!';
      color = 0xFEE75C; // Yellow
    } else if (
      (userChoice === 'rock' && botChoice === 'scissors') ||
      (userChoice === 'paper' && botChoice === 'rock') ||
      (userChoice === 'scissors' && botChoice === 'paper')
    ) {
      result = 'You win!';
      color = 0x57F287; // Green
    } else {
      result = 'I win!';
      color = 0xED4245; // Red
    }

    const embed = funEmbed(
      '🎮 Rock Paper Scissors',
      `You chose ${emojis[userChoice]} **${userChoice}**\n` +
      `I chose ${emojis[botChoice]} **${botChoice}**\n\n` +
      `**${result}**`
    )
      .setColor(color);

    await interaction.reply({ embeds: [embed] });
  }
};
