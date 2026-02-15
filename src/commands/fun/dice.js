/**
 * Dice Command
 * Roll dice with customizable sides
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed, errorEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('dice')
    .setDescription('Roll a dice')
    .addIntegerOption(option =>
      option.setName('sides')
        .setDescription('Number of sides (default: 6)')
        .setRequired(false)
        .setMinValue(2)
        .setMaxValue(100))
    .addIntegerOption(option =>
      option.setName('count')
        .setDescription('Number of dice to roll (default: 1)')
        .setRequired(false)
        .setMinValue(1)
        .setMaxValue(10)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const sides = interaction.options.getInteger('sides') || 6;
    const count = interaction.options.getInteger('count') || 1;

    const rolls = [];
    let total = 0;

    for (let i = 0; i < count; i++) {
      const roll = Math.floor(Math.random() * sides) + 1;
      rolls.push(roll);
      total += roll;
    }

    const embed = funEmbed(
      '🎲 Dice Roll',
      count === 1
        ? `You rolled a **${rolls[0]}** on a ${sides}-sided die!`
        : `**Rolls:** ${rolls.join(', ')}\n**Total:** ${total}\n**Average:** ${(total / count).toFixed(2)}`
    )
      .addFields({ name: 'Dice', value: `${count}d${sides}`, inline: true });

    await interaction.reply({ embeds: [embed] });
  }
};
