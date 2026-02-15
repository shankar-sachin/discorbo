/**
 * Summon Command
 * Dramatically pings someone with a fantasy-style summoning message
 */

const { SlashCommandBuilder } = require('discord.js');
const { funEmbed } = require('../../utils/embedBuilder');

const summonMessages = [
  {
    title: '⚡ Divine Summons',
    message: 'The council demands your presence, {user}. Your wisdom is required.',
    footer: 'Failure to appear will result in consequences.'
  },
  {
    title: '🔮 Mystic Summoning',
    message: 'Through ancient rites and forbidden magic, we call upon {user}!',
    footer: 'The ritual is complete.'
  },
  {
    title: '👑 Royal Decree',
    message: 'By order of the realm, {user} is hereby summoned to the court!',
    footer: 'Attendance is not optional.'
  },
  {
    title: '💀 Necromantic Ritual',
    message: 'From the shadows we call upon {user}. Rise and answer!',
    footer: 'The dead shall walk.'
  },
  {
    title: '🌟 Celestial Beckoning',
    message: 'The stars align. The cosmos speaks. {user}, you are needed.',
    footer: 'Destiny awaits.'
  },
  {
    title: '🗡️ Warrior\'s Call',
    message: 'The drums of war sound! {user}, take up arms and join us!',
    footer: 'Victory or Valhalla!'
  },
  {
    title: '📜 Ancient Scroll',
    message: 'As foretold in the prophecy, {user} must answer the call!',
    footer: 'It is written.'
  },
  {
    title: '🐉 Dragon\'s Roar',
    message: '{user}! The great beast has spoken your name. Come forth!',
    footer: 'Heed the call or face the flames.'
  },
  {
    title: '⚔️ Trial by Combat',
    message: 'The arena calls for {user}! Step forward and face your destiny!',
    footer: 'Honor demands your presence.'
  },
  {
    title: '🌙 Lunar Invocation',
    message: 'Under the light of the moon, we summon {user}!',
    footer: 'The night is yours to claim.'
  },
  {
    title: '🎭 Theatrical Entrance',
    message: 'Ladies and gentlemen, the stage is set for {user}!',
    footer: 'Break a leg!'
  },
  {
    title: '🎪 Circus Announcement',
    message: 'Step right up! {user}, you\'re the main attraction!',
    footer: 'The show must go on.'
  },
  {
    title: '🧙 Wizard\'s Summons',
    message: 'By the power of the seven realms, {user} is summoned!',
    footer: 'Magic compels you.'
  },
  {
    title: '👻 Paranormal Activity',
    message: 'A disturbance in the spiritual realm! {user}, your presence is required!',
    footer: 'The veil grows thin.'
  },
  {
    title: '🎮 Boss Battle Alert',
    message: 'BOSS FIGHT: {user} has entered the arena!',
    footer: 'Difficulty: Legendary'
  }
];

module.exports = {
  data: new SlashCommandBuilder()
    .setName('summon')
    .setDescription('Dramatically summon someone to the conversation')
    .addUserOption(option =>
      option.setName('user')
        .setDescription('The person to summon')
        .setRequired(true))
    .addStringOption(option =>
      option.setName('reason')
        .setDescription('Why are you summoning them?')
        .setRequired(false)),

  category: 'fun',
  cooldown: 10,

  async execute(interaction) {
    const user = interaction.options.getUser('user');
    const reason = interaction.options.getString('reason');

    if (user.bot) {
      return interaction.reply({ content: 'You cannot summon bots!', ephemeral: true });
    }

    // Get random summon message
    const summon = summonMessages[Math.floor(Math.random() * summonMessages.length)];

    // Replace {user} placeholder
    const message = summon.message.replace('{user}', user.toString());

    let description = message;
    if (reason) {
      description += `\n\n**Purpose:** ${reason}`;
    }

    const embed = funEmbed(summon.title, description)
      .setFooter({ text: `${summon.footer} • Summoned by ${interaction.user.username}` })
      .setThumbnail(user.displayAvatarURL({ dynamic: true }));

    await interaction.reply({ content: user.toString(), embeds: [embed] });
  }
};
