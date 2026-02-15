/**
 * Trivia Command
 * Trivia questions with scoring system
 */

const { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } = require('discord.js');
const axios = require('axios');
const { funEmbed, successEmbed, errorEmbed } = require('../../utils/embedBuilder');
const { handleApiError } = require('../../utils/errorHandler');
const { readJSON, writeJSON } = require('../../utils/dataManager');

// Decode HTML entities
function decodeHTML(html) {
  const entities = {
    '&quot;': '"',
    '&#039;': "'",
    '&amp;': '&',
    '&lt;': '<',
    '&gt;': '>',
    '&ldquo;': '"',
    '&rdquo;': '"',
    '&rsquo;': "'"
  };

  return html.replace(/&[^;]+;/g, match => entities[match] || match);
}

module.exports = {
  data: new SlashCommandBuilder()
    .setName('trivia')
    .setDescription('Answer trivia questions and compete for high scores!')
    .addStringOption(option =>
      option.setName('category')
        .setDescription('Question category')
        .setRequired(false)
        .addChoices(
          { name: 'General Knowledge', value: '9' },
          { name: 'Science & Nature', value: '17' },
          { name: 'Computers', value: '18' },
          { name: 'Mathematics', value: '19' },
          { name: 'Sports', value: '21' },
          { name: 'Geography', value: '22' },
          { name: 'History', value: '23' },
          { name: 'Animals', value: '27' }
        ))
    .addStringOption(option =>
      option.setName('difficulty')
        .setDescription('Question difficulty')
        .setRequired(false)
        .addChoices(
          { name: 'Easy', value: 'easy' },
          { name: 'Medium', value: 'medium' },
          { name: 'Hard', value: 'hard' }
        )),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    const category = interaction.options.getString('category');
    const difficulty = interaction.options.getString('difficulty');

    await interaction.deferReply();

    try {
      // Fetch question from Open Trivia DB
      const params = { amount: 1 };
      if (category) params.category = category;
      if (difficulty) params.difficulty = difficulty;

      const response = await axios.get('https://opentdb.com/api.php', {
        params,
        timeout: 5000
      });

      if (response.data.response_code !== 0 || !response.data.results.length) {
        throw new Error('No questions available');
      }

      const question = response.data.results[0];

      // Decode HTML entities
      const questionText = decodeHTML(question.question);
      const correctAnswer = decodeHTML(question.correct_answer);
      const incorrectAnswers = question.incorrect_answers.map(a => decodeHTML(a));

      // Combine and shuffle answers
      const allAnswers = [correctAnswer, ...incorrectAnswers];
      allAnswers.sort(() => Math.random() - 0.5);

      // Find correct answer index
      const correctIndex = allAnswers.indexOf(correctAnswer);

      // Create buttons
      const row1 = new ActionRowBuilder();
      const row2 = new ActionRowBuilder();

      allAnswers.forEach((answer, index) => {
        const button = new ButtonBuilder()
          .setCustomId(`trivia_${interaction.id}_${index}`)
          .setLabel(answer.substring(0, 80)) // Discord label limit
          .setStyle(ButtonStyle.Primary);

        if (index < 2) {
          row1.addComponents(button);
        } else {
          row2.addComponents(button);
        }
      });

      const components = allAnswers.length <= 2 ? [row1] : [row1, row2];

      // Point values based on difficulty
      const points = { easy: 10, medium: 20, hard: 30 }[question.difficulty] || 15;

      const embed = funEmbed(
        `🧠 Trivia - ${question.category}`,
        `**${questionText}**\n\n*Difficulty: ${question.difficulty.toUpperCase()}*\n*Points: ${points}*`
      );

      await interaction.editReply({ embeds: [embed], components });

      // Create collector for button interactions
      const filter = i => i.customId.startsWith(`trivia_${interaction.id}_`) && i.user.id === interaction.user.id;
      const collector = interaction.channel.createMessageComponentCollector({ filter, time: 30000, max: 1 });

      collector.on('collect', async i => {
        const selectedIndex = parseInt(i.customId.split('_')[2]);
        const isCorrect = selectedIndex === correctIndex;

        if (isCorrect) {
          // Update score
          const scores = readJSON('trivia-scores.json');
          if (!scores[i.user.id]) {
            scores[i.user.id] = { username: i.user.username, score: 0, correct: 0, total: 0 };
          }

          scores[i.user.id].score += points;
          scores[i.user.id].correct += 1;
          scores[i.user.id].total += 1;
          scores[i.user.id].username = i.user.username; // Update username

          writeJSON('trivia-scores.json', scores);

          const resultEmbed = successEmbed(
            '✅ Correct!',
            `**Answer:** ${correctAnswer}\n\n**+${points} points!**\nYour total score: **${scores[i.user.id].score}**`
          );

          await i.update({ embeds: [resultEmbed], components: [] });
        } else {
          // Update total questions
          const scores = readJSON('trivia-scores.json');
          if (!scores[i.user.id]) {
            scores[i.user.id] = { username: i.user.username, score: 0, correct: 0, total: 0 };
          }

          scores[i.user.id].total += 1;
          scores[i.user.id].username = i.user.username;

          writeJSON('trivia-scores.json', scores);

          const resultEmbed = errorEmbed(
            '❌ Incorrect!',
            `**Correct answer:** ${correctAnswer}\n\nYour total score: **${scores[i.user.id].score}**`
          );

          await i.update({ embeds: [resultEmbed], components: [] });
        }
      });

      collector.on('end', collected => {
        if (collected.size === 0) {
          const timeoutEmbed = errorEmbed(
            '⏱️ Time\'s Up!',
            `**Correct answer:** ${correctAnswer}`
          );

          interaction.editReply({ embeds: [timeoutEmbed], components: [] }).catch(() => {});
        }
      });

    } catch (error) {
      await handleApiError(error, interaction, 'Trivia API');
    }
  }
};
