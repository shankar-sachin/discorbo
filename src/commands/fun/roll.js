/**
 * Roll Command
 * Custom dice rolling with dramatic responses and crit animations
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');

module.exports = {
  data: new SlashCommandBuilder()
    .setName('roll')
    .setDescription('Roll custom dice with dramatic flair')
    .addStringOption(option =>
      option.setName('dice')
        .setDescription('Dice notation (e.g., d20, 3d6, 2d10+5)')
        .setRequired(true)),

  category: 'fun',
  cooldown: 3,

  async execute(interaction) {
    const diceNotation = interaction.options.getString('dice').toLowerCase().trim();

    // Parse dice notation (supports: d20, 3d6, 2d10+5, 1d20-2)
    const regex = /^(\d*)d(\d+)([\+\-]\d+)?$/;
    const match = diceNotation.match(regex);

    if (!match) {
      const embed = errorEmbed(
        'Invalid Dice Notation',
        'Use format like:\n• `d20` - Roll a d20\n• `3d6` - Roll three d6\n• `2d10+5` - Roll 2d10 and add 5\n• `1d20-2` - Roll d20 and subtract 2'
      );
      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    const count = parseInt(match[1] || '1');
    const sides = parseInt(match[2]);
    const modifier = match[3] ? parseInt(match[3]) : 0;

    // Validation
    if (count > 20) {
      const embed = errorEmbed('Too Many Dice', 'Maximum 20 dice at once!');
      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    if (sides < 2 || sides > 1000) {
      const embed = errorEmbed('Invalid Sides', 'Dice must have between 2 and 1000 sides!');
      return interaction.reply({ embeds: [embed], ephemeral: true });
    }

    // Roll the dice
    const rolls = [];
    let total = 0;

    for (let i = 0; i < count; i++) {
      const roll = Math.floor(Math.random() * sides) + 1;
      rolls.push(roll);
      total += roll;
    }

    const finalTotal = total + modifier;

    // Check for crits
    const isCrit = count === 1 && rolls[0] === sides;
    const isCritFail = count === 1 && rolls[0] === 1;

    let title = '🎲 Dice Roll';
    let description = '';
    let color = 0xEB459E;

    if (isCrit) {
      title = '💥 CRITICAL SUCCESS! 💥';
      description = '**NAT ' + sides + '!** The dice gods smile upon you!\n\n';
      color = 0xFFD700;
    } else if (isCritFail) {
      title = '💀 CRITICAL FAILURE! 💀';
      description = '**NAT 1!** The dice gods have forsaken you!\n\n';
      color = 0xFF0000;
    }

    // Build result text
    if (count === 1) {
      description += `**Result:** ${rolls[0]}${modifier !== 0 ? ` ${modifier >= 0 ? '+' : ''}${modifier}` : ''}\n`;
      if (modifier !== 0) {
        description += `**Total:** ${finalTotal}\n`;
      }
    } else {
      description += `**Rolls:** [${rolls.join(', ')}]\n`;
      description += `**Sum:** ${total}${modifier !== 0 ? ` ${modifier >= 0 ? '+' : ''}${modifier}` : ''}\n`;
      description += `**Total:** ${finalTotal}\n`;
    }

    description += `\n**Notation:** \`${diceNotation}\``;

    // Add flavor text
    if (isCrit) {
      const critMessages = [
        'Legendary!',
        'Absolutely perfect!',
        'The stars align!',
        'Divine intervention!',
        'Flawless execution!'
      ];
      description += `\n\n*${critMessages[Math.floor(Math.random() * critMessages.length)]}*`;
    } else if (isCritFail) {
      const failMessages = [
        'Catastrophic failure!',
        'Everything hurts.',
        'Why even try?',
        'The universe laughs at you.',
        'Better luck next time...'
      ];
      description += `\n\n*${failMessages[Math.floor(Math.random() * failMessages.length)]}*`;
    } else if (finalTotal >= sides * count * 0.8) {
      description += '\n\n*Excellent roll!*';
    } else if (finalTotal <= sides * count * 0.2) {
      description += '\n\n*Oof, rough roll.*';
    }

    const embed = funEmbed(title, description).setColor(color);

    await interaction.reply({ embeds: [embed] });
  }
};
