/**
 * Hotseat Command
 * Picks a random server member and asks them a spicy or funny question
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const questions = [
  "What's your most embarrassing Discord moment?",
  "If you could delete one app from existence, what would it be and why?",
  "What's a secret talent nobody here knows about?",
  "Confess: How many hours do you ACTUALLY spend on Discord daily?",
  "What's the weirdest thing in your search history right now?",
  "If you had to fight one person in this server, who would it be?",
  "What's your most unpopular opinion about anything?",
  "Reveal your top 3 most played songs this month.",
  "What's something you're supposed to be doing right now instead of being here?",
  "If you could swap lives with anyone in this server for a day, who and why?",
  "What's the dumbest way you've injured yourself?",
  "Spill: What's your current phone battery percentage?",
  "What's a lie you've told that spiraled out of control?",
  "What's the last screenshot in your camera roll and why?",
  "If your pets could talk, what secret would they expose about you?",
  "What's your guilty pleasure that you're actually not guilty about?",
  "Be honest: How often do you leave people on read?",
  "What's the worst take you've ever had?",
  "If you could permanently mute one sound, what would it be?",
  "What's something you've Googled that you hope nobody finds out?",
  "Confess: How many unread Discord DMs do you have?",
  "What's the cringiest thing you did as a teenager?",
  "If you could erase one memory, which one?",
  "What's your most toxic trait?",
  "Rate your sleep schedule out of 10 and explain.",
  "What's the most impulsive thing you've bought recently?",
  "If you had to delete all but 3 apps from your phone, which do you keep?",
  "What's a conspiracy theory you lowkey believe?",
  "What's the longest you've gone without showering?",
  "If you could automate one part of your life, what would it be?"
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('hotseat')
    .setDescription('Put a random server member in the hotseat with a spicy question'),

  category: 'fun',
  cooldown: 30,

  async execute(interaction) {
    await interaction.deferReply();

    try {
      // Fetch all members (cache might not have all)
      const members = await interaction.guild.members.fetch();

      // Filter out bots and get random member
      const humanMembers = members.filter(member => !member.user.bot);

      if (humanMembers.size === 0) {
        return interaction.editReply('No human members found in this server!');
      }

      const randomMember = humanMembers.random();
      const randomQuestion = questions[Math.floor(Math.random() * questions.length)];

      const embed = funEmbed(
        '🔥 You\'re in the Hotseat!',
        `${randomMember}, the spotlight is on you!\n\n**Question:**\n*${randomQuestion}*\n\n**The server is waiting for your answer...**`
      )
        .setThumbnail(randomMember.user.displayAvatarURL({ dynamic: true }))
        .setFooter({ text: `Called by ${interaction.user.username}` });

      await interaction.editReply({ content: `${randomMember}`, embeds: [embed] });

    } catch (error) {
      console.error('Error fetching members:', error);
      await interaction.editReply('Failed to fetch server members. Make sure I have the proper permissions!');
    }
  }
};
