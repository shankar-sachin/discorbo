/**
 * Help Command
 * Displays all available commands organized by category
 */

const {
  SlashCommandBuilder,
  EmbedBuilder,
  ActionRowBuilder,
  ButtonBuilder,
  ButtonStyle,
  ComponentType
} = require('discord.js');

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

    const funCommands = [];
    const utilityCommands = [];

    commands.forEach((cmd) => {
      const commandInfo = `\`/${cmd.data.name}\` - ${cmd.data.description}`;

      if (cmd.category === 'fun' || isFunCommand(cmd.data.name)) {
        funCommands.push(commandInfo);
      } else {
        utilityCommands.push(commandInfo);
      }
    });

    funCommands.sort((a, b) => a.localeCompare(b));
    utilityCommands.sort((a, b) => a.localeCompare(b));

    const baseEmbed = new EmbedBuilder()
      .setColor(0x5865F2)
      .setTitle('Discorbo - Command List')
      .setDescription('Browse commands with the buttons below. You can also filter with `/help category:Fun` or `/help category:Utility`.')
      .setTimestamp();

    const pageData = [];

    if (category === 'all' || category === 'fun') {
      pageData.push(...buildCategoryPages(funCommands, 'Fun & Games'));
    }

    if (category === 'all' || category === 'utility') {
      pageData.push(...buildCategoryPages(utilityCommands, 'Utility'));
    }

    if (pageData.length === 0) {
      const emptyEmbed = EmbedBuilder.from(baseEmbed)
        .addFields({ name: 'No commands found', value: 'No commands are available for this filter.' })
        .setFooter({ text: `${commands.size} total commands` });

      await interaction.reply({ embeds: [emptyEmbed] });
      return;
    }

    const pages = pageData.map((page, index) => (
      EmbedBuilder.from(baseEmbed)
        .addFields({ name: page.name, value: page.value, inline: false })
        .setFooter({ text: `${commands.size} total commands | Page ${index + 1}/${pageData.length}` })
    ));

    let currentPage = 0;

    const getComponents = () => {
      if (pages.length <= 1) {
        return [];
      }

      return [
        new ActionRowBuilder().addComponents(
          new ButtonBuilder()
            .setCustomId('help_prev')
            .setLabel('Previous')
            .setStyle(ButtonStyle.Secondary)
            .setDisabled(currentPage === 0),
          new ButtonBuilder()
            .setCustomId('help_page')
            .setLabel(`${currentPage + 1}/${pages.length}`)
            .setStyle(ButtonStyle.Primary)
            .setDisabled(true),
          new ButtonBuilder()
            .setCustomId('help_next')
            .setLabel('Next')
            .setStyle(ButtonStyle.Secondary)
            .setDisabled(currentPage === pages.length - 1)
        )
      ];
    };

    await interaction.reply({
      embeds: [pages[currentPage]],
      components: getComponents()
    });

    if (pages.length <= 1) {
      return;
    }

    const replyMessage = await interaction.fetchReply();
    const collector = replyMessage.createMessageComponentCollector({
      componentType: ComponentType.Button,
      time: 120000
    });

    collector.on('collect', async (buttonInteraction) => {
      if (buttonInteraction.user.id !== interaction.user.id) {
        await buttonInteraction.reply({
          content: 'Only the command user can change help pages.',
          ephemeral: true
        });
        return;
      }

      if (buttonInteraction.customId === 'help_prev') {
        currentPage = Math.max(0, currentPage - 1);
      } else if (buttonInteraction.customId === 'help_next') {
        currentPage = Math.min(pages.length - 1, currentPage + 1);
      } else {
        await buttonInteraction.deferUpdate();
        return;
      }

      await buttonInteraction.update({
        embeds: [pages[currentPage]],
        components: getComponents()
      });
    });

    collector.on('end', async () => {
      await interaction.editReply({ components: [] }).catch(() => {});
    });
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
    'rate', 'ship', 'vibecheck', 'quest', 'roll', 'battle', 'loot',
    'hotseat', 'quote', 'summon', 'bossraid', 'daily', 'maze',
    'maze-leaderboard'
  ];

  return funCommands.includes(name);
}

function buildCategoryPages(commandList, categoryName) {
  if (commandList.length === 0) {
    return [{
      name: categoryName,
      value: 'No commands available.'
    }];
  }

  const chunks = [];
  let currentChunk = [];
  let currentLength = 0;

  for (const cmd of commandList) {
    const lineLength = cmd.length + 1;

    if (currentLength + lineLength > 1024) {
      chunks.push(currentChunk);
      currentChunk = [cmd];
      currentLength = lineLength;
    } else {
      currentChunk.push(cmd);
      currentLength += lineLength;
    }
  }

  if (currentChunk.length > 0) {
    chunks.push(currentChunk);
  }

  return chunks.map((chunk, index) => ({
    name: `${categoryName}${chunks.length > 1 ? ` (${index + 1}/${chunks.length})` : ''}`,
    value: chunk.join('\n')
  }));
}
