/**
 * JSON file data persistence manager
 * Handles reading, writing, and managing JSON data files
 */

const fs = require('fs');
const path = require('path');
const logger = require('./logger');

/**
 * Default schemas for data files
 */
const defaultSchemas = {
  'trivia-scores.json': {},
  'reminders.json': [],
  'afk-users.json': {},
  'command-usage.json': {},
  'quests.json': {},
  'battle-stats.json': {},
  'loot.json': {},
  'quotes.json': {},
  'bossraid.json': {},
  'daily-rewards.json': {},
  'maze-scores.json': {}
};

/**
 * Ensure data directory and file exist
 */
function ensureDataFile(filename) {
  const dataDir = path.join(__dirname, '..', 'data');
  const filepath = path.join(dataDir, filename);

  // Create data directory if it doesn't exist
  if (!fs.existsSync(dataDir)) {
    fs.mkdirSync(dataDir, { recursive: true });
    logger.info(`Created data directory: ${dataDir}`);
  }

  // Create file with default schema if it doesn't exist
  if (!fs.existsSync(filepath)) {
    const defaultData = defaultSchemas[filename] || {};
    fs.writeFileSync(filepath, JSON.stringify(defaultData, null, 2), 'utf8');
    logger.info(`Created data file: ${filename}`);
  }

  return filepath;
}

/**
 * Read JSON data from file
 * @param {string} filename - Name of the JSON file (e.g., 'trivia-scores.json')
 * @returns {Object|Array} Parsed JSON data
 */
function readJSON(filename) {
  try {
    const filepath = ensureDataFile(filename);
    const data = fs.readFileSync(filepath, 'utf8');

    // Handle empty files
    if (!data.trim()) {
      return defaultSchemas[filename] || {};
    }

    return JSON.parse(data);
  } catch (error) {
    logger.error(`Error reading ${filename}:`, error.message);

    // Return default schema on error
    return defaultSchemas[filename] || {};
  }
}

/**
 * Write JSON data to file with atomic write and backup
 * @param {string} filename - Name of the JSON file
 * @param {Object|Array} data - Data to write
 */
function writeJSON(filename, data) {
  try {
    const filepath = ensureDataFile(filename);
    const tempPath = `${filepath}.tmp`;
    const backupPath = `${filepath}.backup`;

    // Write to temporary file first
    fs.writeFileSync(tempPath, JSON.stringify(data, null, 2), 'utf8');

    // Create backup of existing file if it exists
    if (fs.existsSync(filepath)) {
      fs.copyFileSync(filepath, backupPath);
    }

    // Rename temp file to actual file (atomic operation)
    fs.renameSync(tempPath, filepath);

    // Clean up old backup
    if (fs.existsSync(backupPath)) {
      try {
        fs.unlinkSync(backupPath);
      } catch (cleanupError) {
        // On Windows, concurrent processes can briefly lock this file.
        if (cleanupError.code !== 'ENOENT' && cleanupError.code !== 'EPERM') {
          throw cleanupError;
        }
      }
    }

    logger.debug(`Wrote data to ${filename}`);
  } catch (error) {
    logger.error(`Error writing ${filename}:`, error.message);
    throw error;
  }
}

/**
 * Update a specific user's data in a JSON file
 * @param {string} filename - Name of the JSON file
 * @param {string} userId - Discord user ID
 * @param {Object} userData - User data to save
 */
function updateUserData(filename, userId, userData) {
  const data = readJSON(filename);
  data[userId] = userData;
  writeJSON(filename, data);
}

/**
 * Get a specific user's data from a JSON file
 * @param {string} filename - Name of the JSON file
 * @param {string} userId - Discord user ID
 * @returns {Object|null} User data or null if not found
 */
function getUserData(filename, userId) {
  const data = readJSON(filename);
  return data[userId] || null;
}

/**
 * Delete a specific user's data from a JSON file
 * @param {string} filename - Name of the JSON file
 * @param {string} userId - Discord user ID
 */
function deleteUserData(filename, userId) {
  const data = readJSON(filename);
  delete data[userId];
  writeJSON(filename, data);
}

/**
 * Clear all user data for GDPR compliance
 * @param {string} userId - Discord user ID to remove from all files
 */
function clearAllUserData(userId) {
  const files = [
    'trivia-scores.json',
    'afk-users.json',
    'command-usage.json',
    'quests.json',
    'battle-stats.json',
    'loot.json',
    'daily-rewards.json',
    'maze-scores.json'
  ];

  files.forEach(filename => {
    deleteUserData(filename, userId);
  });

  // Handle reminders separately (it's an array)
  const reminders = readJSON('reminders.json');
  const filteredReminders = reminders.filter(r => r.userId !== userId);
  writeJSON('reminders.json', filteredReminders);

  // Handle quotes separately (nested structure)
  const quotes = readJSON('quotes.json');
  Object.keys(quotes).forEach(guildId => {
    if (quotes[guildId].quotes) {
      quotes[guildId].quotes = quotes[guildId].quotes.filter(q => q.author.id !== userId && q.addedBy.id !== userId);
    }
  });
  writeJSON('quotes.json', quotes);

  // Handle boss raids separately (nested structure)
  const bossRaids = readJSON('bossraid.json');
  Object.keys(bossRaids).forEach(guildId => {
    if (bossRaids[guildId].participants) {
      delete bossRaids[guildId].participants[userId];
    }
  });
  writeJSON('bossraid.json', bossRaids);

  logger.info(`Cleared all data for user ${userId}`);
}

module.exports = {
  readJSON,
  writeJSON,
  updateUserData,
  getUserData,
  deleteUserData,
  clearAllUserData,
  ensureDataFile
};
