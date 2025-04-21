import logger from '../logger.js';
import { searchParams } from '../schemas/searchParamsSchema.js';
import { execSync } from 'child_process';

/**
 * @typedef {Object} JobSearchParams
 * @property {string} [siteNames] - Names of job sites to search
 * @property {string} [searchTerm] - Term to search for
 * @property {string} [location] - Job location
 * @property {string} [googleSearchTerm] - Term for Google job search
 * @property {number} [resultsWanted] - Number of results to return
 * @property {number} [hoursOld] - Filter for jobs posted within specified hours
 * @property {string} [countryIndeed] - Country code for Indeed search
 * @property {boolean} [linkedinFetchDescription] - Whether to fetch LinkedIn job descriptions
 * @property {string} [proxies] - Comma-separated list of proxies
 * @property {'json'|'csv'} [format] - Output format: JSON or CSV
 */

export const searchJobsTool = (server, sseManager) => server.tool(
  'search_jobs',
  'Search for jobs across various job listing websites',
  searchParams,
  async (params, extra) => {
    let progressInterval;
    try {
      logger.info('Received search_jobs request', { params, extra });

      // Track progress for SSE clients
      if (extra.sessionId && sseManager.hasConnection(extra.sessionId)) {
        let progress = 0;
        progressInterval = setInterval(() => {
          progress += 5;
          if (progress > 90) {
            progress = 90; // Cap at 90% until complete
          }

          // Send progress to all connected clients
          sseManager.notificationProgress(
            {
              type: 'progress',
              tool: 'search_jobs',
              progress,
              message: `Searching for jobs (${progress}%)...`,
            },
            extra.sessionId,
          );
        }, 2000);
      }

      // Execute job search
      const result = searchJobsHandler(params);

      // Clean up progress interval
      if (progressInterval) {
        clearInterval(progressInterval);

        // Send 100% progress update to all connected clients
        if (extra.sessionId && sseManager.hasConnection(extra.sessionId)) {
          sseManager.notificationProgress(
            {
              type: 'progress',
              tool: 'search_jobs',
              progress: 100,
              message: 'Job search completed',
            },
            extra.sessionId,
          );
        }
      }

      return {
        isError: false,
        content: [
          {
            type: 'text',
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    } catch (error) {
      if (progressInterval) {
        clearInterval(progressInterval);
      }
      logger.error('Error in search_jobs handler', { error: error.message });
      return {
        isError: true,
        error: {
          message: error.message,
          code: 'INTERNAL_SERVER_ERROR',
        },
      };
    }
  },
);

/**
 * Handler for the search_jobs MCP tool
 * @param {JobSearchParams} params - Search parameters
 * @returns {Promise<object>} Search results
 */
export function searchJobsHandler(params) {
  let result;
  try {
    logger.info('Starting job search with parameters', { params });

    const args = buildCommandArgs(params);
    const cmd = `docker run jobspy ${args.join(' ')}`;
    logger.info(`Spawning process with args: ${cmd}`);

    result = execSync(cmd).toString();
    const data = JSON.parse(result);

    logger.info(`Found jobs: ${data.length}`);
    return {
      count: data.length || 0,
      message: 'Job search completed successfully',
      jobs: data,
    };
  } catch (error) {
    logger.error('Error in searchJobsHandler', {
      error: error.message,
      result,
    });
    throw error;
  }
}



/**
 * Build command arguments from parameters
 * @param {JobSearchParams} params - Search parameters
 * @returns {string[]} Command line arguments
 */
function buildCommandArgs(params) {
  const args = [];

  // Add each parameter as a command line argument
  if (params.siteNames) {
    args.push('--site_name', `"${params.siteNames}"`);
  }
  if (params.searchTerm) {
    args.push('--search_term', `"${params.searchTerm}"`);
  }
  if (params.location) {
    args.push('--location', `"${params.location}"`);
  }
  if (params.googleSearchTerm) {
    args.push('--google_search_term', `"${params.googleSearchTerm}"`);
  }
  if (params.resultsWanted) {
    args.push('--results_wanted', `"${params.resultsWanted}"`);
  }
  if (params.hoursOld) {
    args.push('--hours_old', `"${params.hoursOld}"`);
  }
  if (params.countryIndeed) {
    args.push('--country_indeed', `"${params.countryIndeed}"`);
  }
  if (!!params.linkedinFetchDescription) {
    args.push('--linkedin_fetch_description');
  }
  if (params.proxies) {
    args.push('--proxies', `"${params.proxies}"`);
  }
  args.push('--format', 'json');
  return args;
}
