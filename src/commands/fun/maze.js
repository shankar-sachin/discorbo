/**
 * Maze Command
 * Interactive maze game with directional button controls
 */

const { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');
const { ensureMazeEngineReady, runMazeStep } = require('../../utils/mazeEngine');

const TILE = {
  PLAYER: 'P',
  WALL: '#',
  PATH: '.',
  GOAL: 'G',
  COIN: 'C',
  SPIKE: 'S',
  ENEMY: 'E'
};

const LEGEND = 'Key: P=You, .=Path, #=Wall, G=Goal, C=Coin, S=Spike, E=Enemy';

const LEVELS = [
  {
    name: "Beginner's Path",
    difficulty: 'Easy',
    maze: [
      '#######',
      '#P.C..#',
      '#.##.##',
      '#C...G#',
      '#######'
    ],
    startX: 1,
    startY: 1,
    timeLimit: 120
  },
  {
    name: 'Spike Corridor',
    difficulty: 'Medium',
    maze: [
      '#########',
      '#P.C.S.C#',
      '#.##.##.#',
      '#C.S.C..#',
      '#.#.##.G#',
      '#########'
    ],
    startX: 1,
    startY: 1,
    timeLimit: 90
  },
  {
    name: 'Monster Maze',
    difficulty: 'Hard',
    maze: [
      '###########',
      '#P.C.#C..C#',
      '#.#.##.##.#',
      '#C..E..S..#',
      '###C#.##.##',
      '#C....S..G#',
      '###########'
    ],
    startX: 1,
    startY: 1,
    timeLimit: 60
  }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('maze')
    .setDescription('Play an interactive maze game with directional controls!')
    .addIntegerOption(option =>
      option.setName('level')
        .setDescription('Choose difficulty level')
        .addChoices(
          { name: "Easy - Beginner's Path", value: 0 },
          { name: 'Medium - Spike Corridor', value: 1 },
          { name: 'Hard - Monster Maze', value: 2 }
        )
        .setRequired(false)),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    const levelIndex = interaction.options.getInteger('level') ?? 0;
    const level = LEVELS[levelIndex];

    try {
      await ensureMazeEngineReady();
    } catch (error) {
      await interaction.reply({
        content: `Maze engine is unavailable: ${error.message}`,
        ephemeral: true
      });
      return;
    }

    const gameState = {
      userId: interaction.user.id,
      level: levelIndex,
      playerX: level.startX,
      playerY: level.startY,
      coins: 0,
      lives: 3,
      moves: 0,
      startTime: Date.now(),
      board: level.maze.map(row => row.split('')),
      gameOver: false,
      won: false
    };

    const { embed, components } = createGameMessage(gameState, level);
    const gameMessage = await interaction.reply({ embeds: [embed], components, fetchReply: true });

    const collector = gameMessage.createMessageComponentCollector({
      filter: i => i.user.id === interaction.user.id && i.customId.startsWith('maze_'),
      time: level.timeLimit * 1000
    });

    collector.on('collect', async i => {
      if (gameState.gameOver) {
        await i.deferUpdate();
        return;
      }

      const direction = i.customId.split('_')[1];
      const wasWon = gameState.won;

      try {
        const nextState = await runMazeStep(gameState, direction);
        Object.assign(gameState, nextState);
      } catch (error) {
        await i.reply({
          content: `Maze engine error: ${error.message}`,
          ephemeral: true
        });
        return;
      }

      if (!wasWon && gameState.won) {
        const completionTime = Math.floor((Date.now() - gameState.startTime) / 1000);
        saveMazeCompletion(interaction.user.id, interaction.user.username, levelIndex, completionTime, gameState.coins);
      }

      const { embed: updatedEmbed, components: updatedComponents } = createGameMessage(gameState, level);

      if (gameState.gameOver) {
        await i.update({ embeds: [updatedEmbed], components: [] });
        collector.stop();
      } else {
        await i.update({ embeds: [updatedEmbed], components: updatedComponents });
      }
    });

    collector.on('end', async () => {
      if (!gameState.gameOver) {
        gameState.gameOver = true;
        gameState.won = false;
        const { embed: timeoutEmbed } = createGameMessage(gameState, level, true);
        await interaction.editReply({ embeds: [timeoutEmbed], components: [] }).catch(() => {});
      }
    });
  }
};

function createGameMessage(gameState, level, timeout = false) {
  const boardDisplay = gameState.board.map(row => row.join('')).join('\n');
  const elapsed = Math.floor((Date.now() - gameState.startTime) / 1000);
  const timeLeft = Math.max(0, level.timeLimit - elapsed);

  let title;
  let description;
  let color;

  if (gameState.gameOver) {
    if (gameState.won) {
      title = 'Maze: Victory';
      const completionTime = Math.floor((Date.now() - gameState.startTime) / 1000);
      description = `${boardDisplay}\n\nYou escaped the maze.\n\n` +
        `Time: ${completionTime}s\nCoins: ${gameState.coins}\nMoves: ${gameState.moves}\n\n` +
        LEGEND;
      color = 0x57F287;
    } else if (timeout) {
      title = 'Maze: Time Up';
      description = `${boardDisplay}\n\nTime ran out.\n\n` +
        `Coins: ${gameState.coins}\nMoves: ${gameState.moves}\n\n` +
        LEGEND;
      color = 0xFEE75C;
    } else {
      title = 'Maze: Game Over';
      description = `${boardDisplay}\n\nYou lost all lives.\n\n` +
        `Coins: ${gameState.coins}\nMoves: ${gameState.moves}\n\n` +
        LEGEND;
      color = 0xED4245;
    }
  } else {
    title = `Maze: ${level.name} (${level.difficulty})`;
    description = `${boardDisplay}\n\n` +
      `Lives: ${gameState.lives}/3\n` +
      `Coins: ${gameState.coins}\n` +
      `Time: ${timeLeft}s\n` +
      `Moves: ${gameState.moves}\n\n` +
      LEGEND;
    color = 0xEB459E;
  }

  const embed = funEmbed(title, description).setColor(color);

  const components = gameState.gameOver ? [] : [
    new ActionRowBuilder().addComponents(
      new ButtonBuilder()
        .setCustomId('maze_up')
        .setLabel('Up')
        .setStyle(ButtonStyle.Primary)
    ),
    new ActionRowBuilder().addComponents(
      new ButtonBuilder()
        .setCustomId('maze_left')
        .setLabel('Left')
        .setStyle(ButtonStyle.Primary),
      new ButtonBuilder()
        .setCustomId('maze_down')
        .setLabel('Down')
        .setStyle(ButtonStyle.Primary),
      new ButtonBuilder()
        .setCustomId('maze_right')
        .setLabel('Right')
        .setStyle(ButtonStyle.Primary)
    )
  ];

  return { embed, components };
}

function saveMazeCompletion(userId, username, level, time, coins) {
  const data = readJSON('maze-scores.json');

  if (!data[userId]) {
    data[userId] = {
      username,
      completions: [],
      totalCoins: 0
    };
  }

  data[userId].completions.push({
    level,
    time,
    coins,
    timestamp: Date.now()
  });

  data[userId].totalCoins += coins;
  data[userId].username = username;

  writeJSON('maze-scores.json', data);
}
