import logger from '../logger.js';
import { searchJobsHandler } from '../handlers.js';
import { searchParams } from '../schemas/searchParamsSchema.js';

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

export const searchJobsTool = {
  name: 'search_jobs',
  description: 'Search for jobs across various job listing websites',
  schema: searchParams,
  callback: async (params, extra) => {
    let progressInterval;
    try {
      logger.info('Received search_jobs request', { params, extra });

      // Track progress for SSE clients
      if (sseManager && sseManager.hasConnection(extra.sessionId)) {
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
        if (sseManager && sseManager.hasConnection(extra.sessionId)) {
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
};
