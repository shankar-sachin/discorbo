/**
 * Battle Command
 * Turn-based quick battle with stats, crits, and win streaks
 */

const { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

const attacks = [
  { name: 'Quick Jab', damage: [8, 15], crit: 0.1 },
  { name: 'Power Strike', damage: [15, 25], crit: 0.15 },
  { name: 'Tactical Shot', damage: [10, 20], crit: 0.2 },
  { name: 'Chaos Blast', damage: [5, 35], crit: 0.25 }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('battle')
    .setDescription('Challenge someone to battle!')
    .addUserOption(option =>
      option.setName('opponent')
        .setDescription('The user to battle')
        .setRequired(true)),

  category: 'fun',
  cooldown: 30,

  async execute(interaction) {
    const opponent = interaction.options.getUser('opponent');
    const challenger = interaction.user;

    // Validation
    if (opponent.bot) {
      return interaction.reply({ content: 'You cannot battle bots!', ephemeral: true });
    }

    if (opponent.id === challenger.id) {
      return interaction.reply({ content: 'You cannot battle yourself!', ephemeral: true });
    }

    // Initialize battle
    const battleState = {
      challenger: {
        user: challenger,
        hp: 100,
        maxHp: 100
      },
      opponent: {
        user: opponent,
        hp: 100,
        maxHp: 100
      },
      turn: 'challenger',
      round: 1,
      battleLog: []
    };

    // Create battle embed
    const embed = funEmbed(
      '⚔️ Battle Challenge!',
      `${challenger} has challenged ${opponent} to battle!\n\n${opponent}, do you accept?`
    );

    const row = new ActionRowBuilder()
      .addComponents(
        new ButtonBuilder()
          .setCustomId('battle_accept')
          .setLabel('Accept Battle')
          .setStyle(ButtonStyle.Success)
          .setEmoji('⚔️'),
        new ButtonBuilder()
          .setCustomId('battle_decline')
          .setLabel('Decline')
          .setStyle(ButtonStyle.Danger)
          .setEmoji('🏳️')
      );

    await interaction.reply({ embeds: [embed], components: [row] });

    // Wait for opponent response
    const filter = i => i.user.id === opponent.id;
    const collector = interaction.channel.createMessageComponentCollector({ filter, time: 60000, max: 1 });

    collector.on('collect', async i => {
      if (i.customId === 'battle_decline') {
        await i.update({
          embeds: [errorEmbed('Battle Declined', `${opponent} has declined the battle.`)],
          components: []
        });
        return;
      }

      // Battle accepted - start the battle
      await startBattle(interaction, battleState);
    });

    collector.on('end', collected => {
      if (collected.size === 0) {
        interaction.editReply({
          embeds: [errorEmbed('Battle Timeout', 'The opponent did not respond in time.')],
          components: []
        }).catch(() => {});
      }
    });
  }
};

async function startBattle(interaction, state) {
  const battleEmbed = createBattleEmbed(state);
  const buttons = createBattleButtons(state);

  await interaction.editReply({ embeds: [battleEmbed], components: [buttons] });

  // Create battle collector
  const filter = i => {
    const currentPlayer = state.turn === 'challenger' ? state.challenger.user : state.opponent.user;
    return i.user.id === currentPlayer.id && i.customId.startsWith('attack_');
  };

  const collector = interaction.channel.createMessageComponentCollector({ filter, time: 300000 });

  collector.on('collect', async i => {
    const attackIndex = parseInt(i.customId.split('_')[1]);
    await handleAttack(i, state, attackIndex);

    // Check if battle is over
    if (state.challenger.hp <= 0 || state.opponent.hp <= 0) {
      await endBattle(i, state);
      collector.stop();
    } else {
      // Continue battle
      const battleEmbed = createBattleEmbed(state);
      const buttons = createBattleButtons(state);
      await i.update({ embeds: [battleEmbed], components: [buttons] });
    }
  });
}

function createBattleEmbed(state) {
  const { challenger, opponent, round, battleLog } = state;

  const challengerHPBar = createHPBar(challenger.hp, challenger.maxHp);
  const opponentHPBar = createHPBar(opponent.hp, opponent.maxHp);

  const currentPlayer = state.turn === 'challenger' ? challenger.user : opponent.user;

  let description = `**Round ${round}**\n\n`;
  description += `${challenger.user.username}: ${challengerHPBar} ${challenger.hp}/${challenger.maxHp} HP\n`;
  description += `${opponent.user.username}: ${opponentHPBar} ${opponent.hp}/${opponent.maxHp} HP\n\n`;
  description += `**${currentPlayer.username}'s Turn**\n\n`;

  if (battleLog.length > 0) {
    description += `**Last Action:**\n${battleLog[battleLog.length - 1]}`;
  }

  return funEmbed('⚔️ Battle in Progress', description);
}

function createBattleButtons(state) {
  const row = new ActionRowBuilder();

  attacks.forEach((attack, index) => {
    row.addComponents(
      new ButtonBuilder()
        .setCustomId(`attack_${index}`)
        .setLabel(attack.name)
        .setStyle(ButtonStyle.Primary)
    );
  });

  return row;
}

function createHPBar(current, max) {
  const percentage = current / max;
  const barLength = 10;
  const filled = Math.ceil(barLength * percentage);

  let emoji = '🟢';
  if (percentage < 0.5) emoji = '🟡';
  if (percentage < 0.25) emoji = '🔴';

  return emoji.repeat(Math.max(0, filled)) + '⚫'.repeat(Math.max(0, barLength - filled));
}

async function handleAttack(interaction, state, attackIndex) {
  const attack = attacks[attackIndex];
  const attacker = state.turn === 'challenger' ? state.challenger : state.opponent;
  const defender = state.turn === 'challenger' ? state.opponent : state.challenger;

  // Calculate damage
  const baseDamage = Math.floor(Math.random() * (attack.damage[1] - attack.damage[0] + 1)) + attack.damage[0];
  const isCrit = Math.random() < attack.crit;
  const damage = isCrit ? Math.floor(baseDamage * 1.5) : baseDamage;

  // Apply damage
  defender.hp = Math.max(0, defender.hp - damage);

  // Create battle log entry
  let logEntry = `${attacker.user.username} used **${attack.name}**! `;
  if (isCrit) {
    logEntry += `💥 **CRITICAL HIT!** `;
  }
  logEntry += `Dealt **${damage}** damage to ${defender.user.username}!`;

  state.battleLog.push(logEntry);

  // Switch turn
  state.turn = state.turn === 'challenger' ? 'opponent' : 'challenger';
  state.round++;
}

async function endBattle(interaction, state) {
  const winner = state.challenger.hp > 0 ? state.challenger : state.opponent;
  const loser = state.challenger.hp > 0 ? state.opponent : state.challenger;

  // Update battle stats
  const battleStats = readJSON('battle-stats.json');

  if (!battleStats[winner.user.id]) {
    battleStats[winner.user.id] = { username: winner.user.username, wins: 0, losses: 0, streak: 0 };
  }
  if (!battleStats[loser.user.id]) {
    battleStats[loser.user.id] = { username: loser.user.username, wins: 0, losses: 0, streak: 0 };
  }

  battleStats[winner.user.id].wins++;
  battleStats[winner.user.id].streak++;
  battleStats[winner.user.id].username = winner.user.username;

  battleStats[loser.user.id].losses++;
  battleStats[loser.user.id].streak = 0;
  battleStats[loser.user.id].username = loser.user.username;

  writeJSON('battle-stats.json', battleStats);

  const winnerStats = battleStats[winner.user.id];
  const streakText = winnerStats.streak > 1 ? `\n🔥 **${winnerStats.streak} Win Streak!**` : '';

  const embed = successEmbed(
    '🏆 Battle Complete!',
    `**Winner:** ${winner.user.username}\n**Final HP:** ${winner.hp}/${winner.maxHp}\n\n**Stats:** ${winnerStats.wins}W - ${winnerStats.losses}L${streakText}`
  );

  await interaction.update({ embeds: [embed], components: [] });
}
