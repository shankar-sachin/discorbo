/**
 * Calculator Command
 * Perform mathematical calculations safely
 */

const { SlashCommandBuilder } = require('discord.js');
const { evaluate } = require('mathjs');
const { successEmbed, errorEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('calc')
    .setDescription('Perform mathematical calculations')
    .addStringOption(option =>
      option.setName('expression')
        .setDescription('The mathematical expression to evaluate (e.g., 5*5+10)')
        .setRequired(true)),

  cooldown: 3,

  async execute(interaction) {
    const expression = interaction.options.getString('expression');

    try {
      // Use mathjs for safe evaluation (prevents code injection)
      const result = evaluate(expression);

      const embed = successEmbed(
        'Calculator',
        `**Expression:** \`${expression}\`\n**Result:** \`${result}\``
      );

      await interaction.reply({ embeds: [embed] });
    } catch (error) {
      const embed = errorEmbed(
        'Invalid Expression',
        'Please provide a valid mathematical expression.\n\n**Examples:**\n• `5 * 5 + 10`\n• `sqrt(16)`\n• `sin(45 deg)`\n• `2^8`'
      );

      await interaction.reply({ embeds: [embed], ephemeral: true });
    }
  }
};
