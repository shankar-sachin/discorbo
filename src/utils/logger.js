/**
 * Colored console logging utility
 * Provides consistent, colored log output for different message types
 */

const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',

  // Foreground colors
  black: '\x1b[30m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',

  // Background colors
  bgRed: '\x1b[41m',
  bgGreen: '\x1b[42m',
  bgYellow: '\x1b[43m',
  bgBlue: '\x1b[44m',
};

/**
 * Get current timestamp
 */
function getTimestamp() {
  const now = new Date();
  return now.toLocaleTimeString('en-US', { hour12: false });
}

/**
 * Log info message (blue)
 */
function info(message, ...args) {
  console.log(
    `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.blue}[INFO]${colors.reset} ${message}`,
    ...args
  );
}

/**
 * Log success message (green)
 */
function success(message, ...args) {
  console.log(
    `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.green}[SUCCESS]${colors.reset} ${message}`,
    ...args
  );
}

/**
 * Log warning message (yellow)
 */
function warn(message, ...args) {
  console.log(
    `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.yellow}[WARN]${colors.reset} ${message}`,
    ...args
  );
}

/**
 * Log error message (red)
 */
function error(message, ...args) {
  console.log(
    `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.red}[ERROR]${colors.reset} ${message}`,
    ...args
  );
}

/**
 * Log debug message (magenta)
 */
function debug(message, ...args) {
  if (process.env.NODE_ENV === 'development') {
    console.log(
      `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.magenta}[DEBUG]${colors.reset} ${message}`,
      ...args
    );
  }
}

/**
 * Log command execution (cyan)
 */
function command(username, commandName, ...args) {
  console.log(
    `${colors.dim}[${getTimestamp()}]${colors.reset} ${colors.cyan}[CMD]${colors.reset} ${colors.bright}${username}${colors.reset} used ${colors.bright}/${commandName}${colors.reset}`,
    ...args
  );
}

module.exports = {
  info,
  success,
  warn,
  error,
  debug,
  command,
  colors
};
