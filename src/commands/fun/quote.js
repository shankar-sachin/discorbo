/**
 * Quote Command
 * Save and retrieve iconic server quotes
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('quote')
    .setDescription('Manage iconic server quotes')
    .addSubcommand(subcommand =>
      subcommand
        .setName('add')
        .setDescription('Add a quote to the collection')
        .addStringOption(option =>
          option.setName('text')
            .setDescription('The quote text')
            .setRequired(true))
        .addUserOption(option =>
          option.setName('author')
            .setDescription('Who said it')
            .setRequired(false)))
    .addSubcommand(subcommand =>
      subcommand
        .setName('random')
        .setDescription('Get a random quote'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('list')
        .setDescription('List recent quotes'))
    .addSubcommand(subcommand =>
      subcommand
        .setName('remove')
        .setDescription('Remove a quote by ID (moderators only)')
        .addIntegerOption(option =>
          option.setName('id')
            .setDescription('Quote ID to remove')
            .setRequired(true))),

  category: 'fun',
  cooldown: 5,

  async execute(interaction) {
    const subcommand = interaction.options.getSubcommand();
    const quotesData = readJSON('quotes.json');

    const guildId = interaction.guild.id;

    if (!quotesData[guildId]) {
      quotesData[guildId] = { quotes: [] };
    }

    const guildQuotes = quotesData[guildId].quotes;

    if (subcommand === 'add') {
      const text = interaction.options.getString('text');
      const author = interaction.options.getUser('author') || interaction.user;

      const quote = {
        id: guildQuotes.length + 1,
        text: text,
        author: {
          id: author.id,
          username: author.username,
          avatar: author.displayAvatarURL()
        },
        addedBy: {
          id: interaction.user.id,
          username: interaction.user.username
        },
        timestamp: Date.now(),
        guildId: guildId
      };

      guildQuotes.push(quote);
      writeJSON('quotes.json', quotesData);

      const embed = successEmbed(
        '💬 Quote Added!',
        `**Quote #${quote.id}**\n"${text}"\n\n— ${author.username}`
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'random') {
      if (guildQuotes.length === 0) {
        const embed = errorEmbed(
          'No Quotes Yet',
          'This server has no quotes! Use `/quote add` to add one.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      const randomQuote = guildQuotes[Math.floor(Math.random() * guildQuotes.length)];

      const embed = new EmbedBuilder()
        .setColor(0xEB459E)
        .setDescription(`"${randomQuote.text}"`)
        .setAuthor({
          name: randomQuote.author.username,
          iconURL: randomQuote.author.avatar
        })
        .setFooter({ text: `Quote #${randomQuote.id} • Added by ${randomQuote.addedBy.username}` })
        .setTimestamp(randomQuote.timestamp);

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'list') {
      if (guildQuotes.length === 0) {
        const embed = errorEmbed(
          'No Quotes Yet',
          'This server has no quotes! Use `/quote add` to add one.'
        );
        return interaction.reply({ embeds: [embed] });
      }

      // Show last 10 quotes
      const recentQuotes = guildQuotes.slice(-10).reverse();

      let quotesText = '';
      recentQuotes.forEach(quote => {
        const preview = quote.text.length > 60 ? quote.text.substring(0, 60) + '...' : quote.text;
        quotesText += `**#${quote.id}** "${preview}" — ${quote.author.username}\n`;
      });

      if (guildQuotes.length > 10) {
        quotesText += `\n*Showing last 10 of ${guildQuotes.length} quotes*`;
      }

      const embed = funEmbed(
        '💬 Recent Quotes',
        quotesText
      );

      await interaction.reply({ embeds: [embed] });
    }

    else if (subcommand === 'remove') {
      // Check if user has manage messages permission
      if (!interaction.member.permissions.has('ManageMessages')) {
        const embed = errorEmbed(
          'Permission Denied',
          'You need the **Manage Messages** permission to remove quotes.'
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      const quoteId = interaction.options.getInteger('id');
      const quoteIndex = guildQuotes.findIndex(q => q.id === quoteId);

      if (quoteIndex === -1) {
        const embed = errorEmbed(
          'Quote Not Found',
          `No quote found with ID ${quoteId}.`
        );
        return interaction.reply({ embeds: [embed], ephemeral: true });
      }

      const removedQuote = guildQuotes[quoteIndex];
      guildQuotes.splice(quoteIndex, 1);
      writeJSON('quotes.json', quotesData);

      const embed = successEmbed(
        'Quote Removed',
        `Removed quote #${quoteId}:\n"${removedQuote.text}"`
      );

      await interaction.reply({ embeds: [embed] });
    }
  }
};
