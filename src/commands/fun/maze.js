/**
 * Maze Command
 * Interactive maze game with arrow button controls
 */

const { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { readJSON, writeJSON } = require('../../utils/dataManager');

// Game constants
const PLAYER = '🧑';
const WALL = '⬛';
const PATH = '⬜';
const GOAL = '🏁';
const COIN = '🪙';
const SPIKE = '🔺';
const ENEMY = '👹';

// Maze levels
const LEVELS = [
    {
        name: "Beginner's Path",
        difficulty: "Easy",
        width: 7,
        height: 5,
        maze: [
            '⬛⬛⬛⬛⬛⬛⬛',
            '⬛🧑⬜🪙⬜⬜⬛',
            '⬛⬜⬛⬛⬜⬛⬛',
            '⬛🪙⬜⬜⬜🏁⬛',
            '⬛⬛⬛⬛⬛⬛⬛'
        ],
        startX: 1,
        startY: 1,
        timeLimit: 120
    },
    {
        name: "Spike Corridor",
        difficulty: "Medium",
        width: 9,
        height: 6,
        maze: [
            '⬛⬛⬛⬛⬛⬛⬛⬛⬛',
            '⬛🧑⬜🪙⬜🔺⬜🪙⬛',
            '⬛⬜⬛⬛⬜⬛⬛⬜⬛',
            '⬛🪙⬜🔺⬜🪙⬜⬜⬛',
            '⬛⬜⬛⬜⬛⬛⬜🏁⬛',
            '⬛⬛⬛⬛⬛⬛⬛⬛⬛'
        ],
        startX: 1,
        startY: 1,
        timeLimit: 90
    },
    {
        name: "Monster Maze",
        difficulty: "Hard",
        width: 11,
        height: 7,
        maze: [
            '⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛',
            '⬛🧑⬜🪙⬜⬛🪙⬜⬜🪙⬛',
            '⬛⬜⬛⬜⬛⬛⬜⬛⬛⬜⬛',
            '⬛🪙⬜⬜👹⬜⬜🔺⬜⬜⬛',
            '⬛⬛⬛🪙⬛⬜⬛⬛⬜⬛⬛',
            '⬛🪙⬜⬜⬜⬜🔺⬜⬜🏁⬛',
            '⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛'
        ],
        startX: 1,
        startY: 1,
        timeLimit: 60
    }
];

module.exports = {
    data: new SlashCommandBuilder()
        .setName('maze')
        .setDescription('Play an interactive maze game with arrow controls!')
        .addIntegerOption(option =>
            option.setName('level')
                .setDescription('Choose difficulty level')
                .addChoices(
                    { name: '🟢 Easy - Beginner\'s Path', value: 0 },
                    { name: '🟡 Medium - Spike Corridor', value: 1 },
                    { name: '🔴 Hard - Monster Maze', value: 2 }
                )
                .setRequired(false)),

    category: 'fun',
    cooldown: 10,

    async execute(interaction) {
        const levelIndex = interaction.options.getInteger('level') ?? 0;
        const level = LEVELS[levelIndex];

        // Initialize game state
        const gameState = {
            userId: interaction.user.id,
            level: levelIndex,
            playerX: level.startX,
            playerY: level.startY,
            coins: 0,
            lives: 3,
            moves: 0,
            startTime: Date.now(),
            timeLimit: level.timeLimit,
            board: level.maze.map(row => row.split('')),
            gameOver: false,
            won: false
        };

        // Create and send initial game board
        const { embed, components } = createGameMessage(gameState, level);
        await interaction.reply({ embeds: [embed], components });

        // Create collector for button interactions
        const filter = i => i.user.id === interaction.user.id;
        const collector = interaction.channel.createMessageComponentCollector({
            filter,
            time: level.timeLimit * 1000
        });

        collector.on('collect', async i => {
            if (gameState.gameOver) {
                await i.update({});
                return;
            }

            // Get direction from button
            const direction = i.customId.split('_')[1];
            const newX = gameState.playerX + (direction === 'right' ? 1 : direction === 'left' ? -1 : 0);
            const newY = gameState.playerY + (direction === 'down' ? 1 : direction === 'up' ? -1 : 0);

            // Check if move is valid
            if (newY < 0 || newY >= gameState.board.length ||
                newX < 0 || newX >= gameState.board[0].length) {
                await i.update({});
                return;
            }

            const targetCell = gameState.board[newY][newX];

            // Check for wall
            if (targetCell === WALL) {
                await i.update({});
                return;
            }

            // Clear current position
            gameState.board[gameState.playerY][gameState.playerX] = PATH;

            // Handle different cell types
            if (targetCell === COIN) {
                gameState.coins++;
            } else if (targetCell === SPIKE || targetCell === ENEMY) {
                gameState.lives--;
                if (gameState.lives <= 0) {
                    gameState.gameOver = true;
                    gameState.won = false;
                }
            } else if (targetCell === GOAL) {
                gameState.gameOver = true;
                gameState.won = true;

                // Save completion time
                const completionTime = Math.floor((Date.now() - gameState.startTime) / 1000);
                saveMazeCompletion(interaction.user.id, interaction.user.username, levelIndex, completionTime, gameState.coins);
            }

            // Move player
            gameState.playerX = newX;
            gameState.playerY = newY;
            gameState.board[newY][newX] = PLAYER;
            gameState.moves++;

            // Update message
            const { embed, components } = createGameMessage(gameState, level);

            if (gameState.gameOver) {
                await i.update({ embeds: [embed], components: [] });
                collector.stop();
            } else {
                await i.update({ embeds: [embed], components });
            }
        });

        collector.on('end', async () => {
            if (!gameState.gameOver) {
                gameState.gameOver = true;
                gameState.won = false;
                const { embed } = createGameMessage(gameState, level, true);
                await interaction.editReply({ embeds: [embed], components: [] }).catch(() => {});
            }
        });
    }
};

/**
 * Create game message with board and controls
 */
function createGameMessage(gameState, level, timeout = false) {
    // Build board display
    let boardDisplay = '';
    gameState.board.forEach(row => {
        boardDisplay += row.join('') + '\n';
    });

    // Calculate time remaining
    const elapsed = Math.floor((Date.now() - gameState.startTime) / 1000);
    const timeLeft = Math.max(0, level.timeLimit - elapsed);

    let title, description, color;

    if (gameState.gameOver) {
        if (gameState.won) {
            title = '🎉 Victory!';
            const completionTime = Math.floor((Date.now() - gameState.startTime) / 1000);
            description = `${boardDisplay}\n**You escaped the maze!**\n\n` +
                `⏱️ Time: ${completionTime}s\n` +
                `🪙 Coins: ${gameState.coins}\n` +
                `👣 Moves: ${gameState.moves}`;
            color = 0x57F287;
        } else if (timeout) {
            title = '⏰ Time\'s Up!';
            description = `${boardDisplay}\n**You ran out of time!**\n\n` +
                `🪙 Coins: ${gameState.coins}\n` +
                `👣 Moves: ${gameState.moves}`;
            color = 0xFEE75C;
        } else {
            title = '💀 Game Over!';
            description = `${boardDisplay}\n**You lost all your lives!**\n\n` +
                `🪙 Coins: ${gameState.coins}\n` +
                `👣 Moves: ${gameState.moves}`;
            color = 0xED4245;
        }
    } else {
        title = `🎮 ${level.name} (${level.difficulty})`;
        description = `${boardDisplay}\n` +
            `❤️ Lives: ${'❤️'.repeat(gameState.lives)}${'🖤'.repeat(3 - gameState.lives)}\n` +
            `🪙 Coins: ${gameState.coins}\n` +
            `⏱️ Time: ${timeLeft}s\n` +
            `👣 Moves: ${gameState.moves}\n\n` +
            `**Goal:** Reach 🏁 | Avoid: 🔺 👹`;
        color = 0xEB459E;
    }

    const embed = funEmbed(title, description).setColor(color);

    // Create control buttons
    const components = gameState.gameOver ? [] : [
        new ActionRowBuilder().addComponents(
            new ButtonBuilder()
                .setCustomId('maze_up')
                .setEmoji('⬆️')
                .setStyle(ButtonStyle.Primary)
                .setLabel('Up')
        ),
        new ActionRowBuilder().addComponents(
            new ButtonBuilder()
                .setCustomId('maze_left')
                .setEmoji('⬅️')
                .setStyle(ButtonStyle.Primary)
                .setLabel('Left'),
            new ButtonBuilder()
                .setCustomId('maze_down')
                .setEmoji('⬇️')
                .setStyle(ButtonStyle.Primary)
                .setLabel('Down'),
            new ButtonBuilder()
                .setCustomId('maze_right')
                .setEmoji('➡️')
                .setStyle(ButtonStyle.Primary)
                .setLabel('Right')
        )
    ];

    return { embed, components };
}

/**
 * Save maze completion to leaderboard
 */
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
