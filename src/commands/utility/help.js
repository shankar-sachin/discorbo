/**
 * Help Command
 * Displays all available commands organized by category
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('help')
    .setDescription('Display all available commands and their descriptions')
    .addStringOption(option =>
      option.setName('category')
        .setDescription('Filter commands by category')
        .addChoices(
          { name: 'Fun & Games', value: 'fun' },
          { name: 'Utility', value: 'utility' },
          { name: 'All Commands', value: 'all' }
        )),

  cooldown: 5,

  async execute(interaction) {
    const category = interaction.options.getString('category') || 'all';
    const commands = interaction.client.commands;

    // Organize commands by category
    const funCommands = [];
    const utilityCommands = [];

    commands.forEach(cmd => {
      const commandInfo = `\`/${cmd.data.name}\` - ${cmd.data.description}`;

      // Determine category based on file path or command properties
      if (cmd.category === 'fun' || isFunCommand(cmd.data.name)) {
        funCommands.push(commandInfo);
      } else {
        utilityCommands.push(commandInfo);
      }
    });

    // Build embed
    const embed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle('Discorbo - Command List')
      .setDescription('Here are all available commands:')
      .setTimestamp()
      .setFooter({ text: 'Use /help <category> to filter commands' });

    // Add fields based on category filter
    if (category === 'all' || category === 'fun') {
      if (funCommands.length > 0) {
        embed.addFields({
          name: '🎮 Fun & Games',
          value: funCommands.join('\n') || 'No fun commands available',
          inline: false
        });
      }
    }

    if (category === 'all' || category === 'utility') {
      if (utilityCommands.length > 0) {
        embed.addFields({
          name: '🛠️ Utility',
          value: utilityCommands.join('\n') || 'No utility commands available',
          inline: false
        });
      }
    }

    embed.addFields({
      name: 'Total Commands',
      value: `${commands.size} commands available`,
      inline: false
    });

    await interaction.reply({ embeds: [embed] });
  }
};

/**
 * Determine if a command is a fun command based on its name
 */
function isFunCommand(name) {
  const funCommands = [
    '8ball', 'coinflip', 'dice', 'rps', 'joke', 'meme', 'trivia',
    'trivia-leaderboard', 'would-you-rather', 'random-number',
    'random-choice', 'flip-text', 'mock-text', 'reverse-text',
    'rate', 'ship'
  ];

  return funCommands.includes(name);
}
