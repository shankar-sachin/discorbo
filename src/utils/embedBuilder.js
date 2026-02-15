/**
 * Standardized Discord embed builder
 * Provides consistent embed styling across all commands
 */

const { EmbedBuilder } = require('discord.js');

/**
 * Base embed with common properties
 */
function createBaseEmbed() {
  return new EmbedBuilder()
    .setTimestamp()
    .setFooter({ text: 'Discorbo • Made with discord.js' });
}

/**
 * Success embed (green)
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @returns {EmbedBuilder}
 */
function successEmbed(title, description) {
  return createBaseEmbed()
    .setColor(0x57F287) // Discord green
    .setTitle(title)
    .setDescription(description);
}

/**
 * Error embed (red)
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @returns {EmbedBuilder}
 */
function errorEmbed(title, description) {
  return createBaseEmbed()
    .setColor(0xED4245) // Discord red
    .setTitle(title)
    .setDescription(description);
}

/**
 * Info embed (blue)
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @returns {EmbedBuilder}
 */
function infoEmbed(title, description) {
  return createBaseEmbed()
    .setColor(0x5865F2) // Discord blurple
    .setTitle(title)
    .setDescription(description);
}

/**
 * Warning embed (yellow)
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @returns {EmbedBuilder}
 */
function warningEmbed(title, description) {
  return createBaseEmbed()
    .setColor(0xFEE75C) // Discord yellow
    .setTitle(title)
    .setDescription(description);
}

/**
 * Fun/game embed (purple)
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @returns {EmbedBuilder}
 */
function funEmbed(title, description) {
  return createBaseEmbed()
    .setColor(0xEB459E) // Discord fuchsia
    .setTitle(title)
    .setDescription(description);
}

/**
 * Custom color embed
 * @param {string} title - Embed title
 * @param {string} description - Embed description
 * @param {number} color - Hex color code
 * @returns {EmbedBuilder}
 */
function customEmbed(title, description, color) {
  return createBaseEmbed()
    .setColor(color)
    .setTitle(title)
    .setDescription(description);
}

module.exports = {
  successEmbed,
  errorEmbed,
  infoEmbed,
  warningEmbed,
  funEmbed,
  customEmbed,
  createBaseEmbed
};
