// Simple configurable logger for the JobSpy MCP server

/**
 * Log levels in order of verbosity
 * @type {Object}
 */
const LOG_LEVELS = {
  error: 0,
  warn: 1,
  info: 2,
  debug: 3
};

// Get log level from environment or default to 'warn'
const currentLevel = process.env.LOG_LEVEL || 'warn';
const currentLevelValue = LOG_LEVELS[currentLevel] ?? LOG_LEVELS.warn;

/**
 * Format a log message with timestamp and level
 * @param {string} level - Log level
 * @param {string} message - Log message
 * @param {object} [metadata] - Additional metadata to log
 * @returns {string} Formatted log message
 */
function formatLogMessage(level, message, metadata = {}) {
  const timestamp = new Date().toISOString();
  const metadataStr = Object.keys(metadata).length > 0 
    ? ` ${JSON.stringify(metadata)}`
    : '';
  
  return `${timestamp} [${level.toUpperCase()}] ${message}${metadataStr}`;
}

/**
 * Logger object with methods for each log level
 */
const logger = {
  /**
   * Log error message
   * @param {string} message - Message to log
   * @param {object} [metadata] - Additional metadata
   */
  error(message, metadata) {
    if (currentLevelValue >= LOG_LEVELS.error) {
      console.error(formatLogMessage('error', message, metadata));
    }
  },
  
  /**
   * Log warning message
   * @param {string} message - Message to log
   * @param {object} [metadata] - Additional metadata
   */
  warn(message, metadata) {
    if (currentLevelValue >= LOG_LEVELS.warn) {
      console.error(formatLogMessage('warn', message, metadata));
    }
  },
  
  /**
   * Log info message
   * @param {string} message - Message to log
   * @param {object} [metadata] - Additional metadata
   */
  info(message, metadata) {
    if (currentLevelValue >= LOG_LEVELS.info) {
      console.error(formatLogMessage('info', message, metadata));
    }
  },
  
  /**
   * Log debug message
   * @param {string} message - Message to log
   * @param {object} [metadata] - Additional metadata
   */
  debug(message, metadata) {
    if (currentLevelValue >= LOG_LEVELS.debug) {
      console.error(formatLogMessage('debug', message, metadata));
    }
  }
};

export default logger;
