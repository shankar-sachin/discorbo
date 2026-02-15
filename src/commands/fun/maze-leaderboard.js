/**
 * Maze Leaderboard Command
 * View top maze completion times
 */

const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { readJSON } = require('../../utils/dataManager');

const LEVEL_NAMES = [
    "Beginner's Path (Easy)",
    "Spike Corridor (Medium)",
    "Monster Maze (Hard)"
];

module.exports = {
    data: new SlashCommandBuilder()
        .setName('maze-leaderboard')
        .setDescription('View maze game leaderboards')
        .addIntegerOption(option =>
            option.setName('level')
                .setDescription('Filter by level')
                .addChoices(
                    { name: 'Easy - Beginner\'s Path', value: 0 },
                    { name: 'Medium - Spike Corridor', value: 1 },
                    { name: 'Hard - Monster Maze', value: 2 },
                    { name: 'All Levels', value: -1 }
                )
                .setRequired(false)),

    category: 'fun',
    cooldown: 5,

    async execute(interaction) {
        const levelFilter = interaction.options.getInteger('level') ?? -1;
        const data = readJSON('maze-scores.json');

        if (Object.keys(data).length === 0) {
            const embed = new EmbedBuilder()
                .setColor(0xEB459E)
                .setTitle('🏆 Maze Leaderboard')
                .setDescription('No one has completed any mazes yet!\nUse `/maze` to be the first!');

            return interaction.reply({ embeds: [embed] });
        }

        if (levelFilter === -1) {
            // Show total coins leaderboard
            const leaderboard = Object.entries(data)
                .map(([userId, info]) => ({
                    userId,
                    username: info.username,
                    totalCoins: info.totalCoins,
                    completions: info.completions.length
                }))
                .sort((a, b) => b.totalCoins - a.totalCoins)
                .slice(0, 10);

            let leaderboardText = '';
            leaderboard.forEach((player, index) => {
                const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
                leaderboardText += `${medal} **${player.username}**\n`;
                leaderboardText += `　🪙 ${player.totalCoins} coins | ${player.completions} completions\n`;
            });

            const embed = new EmbedBuilder()
                .setColor(0xEB459E)
                .setTitle('🏆 Maze Leaderboard - Total Coins')
                .setDescription(leaderboardText)
                .setFooter({ text: 'Use /maze to play!' });

            await interaction.reply({ embeds: [embed] });
        } else {
            // Show fastest times for specific level
            const allCompletions = [];

            Object.entries(data).forEach(([userId, info]) => {
                info.completions
                    .filter(c => c.level === levelFilter)
                    .forEach(completion => {
                        allCompletions.push({
                            userId,
                            username: info.username,
                            ...completion
                        });
                    });
            });

            if (allCompletions.length === 0) {
                const embed = new EmbedBuilder()
                    .setColor(0xEB459E)
                    .setTitle(`🏆 ${LEVEL_NAMES[levelFilter]} - Leaderboard`)
                    .setDescription('No completions for this level yet!\nBe the first to complete it!');

                return interaction.reply({ embeds: [embed] });
            }

            // Sort by fastest time
            allCompletions.sort((a, b) => a.time - b.time);
            const top10 = allCompletions.slice(0, 10);

            let leaderboardText = '';
            top10.forEach((entry, index) => {
                const medal = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
                leaderboardText += `${medal} **${entry.username}**\n`;
                leaderboardText += `　⏱️ ${entry.time}s | 🪙 ${entry.coins} coins\n`;
            });

            const embed = new EmbedBuilder()
                .setColor(0xEB459E)
                .setTitle(`🏆 ${LEVEL_NAMES[levelFilter]} - Fastest Times`)
                .setDescription(leaderboardText)
                .setFooter({ text: `Total completions: ${allCompletions.length}` });

            await interaction.reply({ embeds: [embed] });
        }
    }
};
