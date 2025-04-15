import { execSync } from 'child_process';
import { ZodError } from 'zod';
import logger from './logger.js';
import { jobSearchResultSchema } from './schemas/jobSchema.js';

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
    args.push('--site_names', `"${params.siteNames}"`);
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

/**
 * Process job search results
 * @param {object} results - Job search results
 * @param {string} toolName - Name of the tool used for the search
 * @returns {object|null} Processed results
 */
export const processJobSearchResults = async (results, toolName) => {
  try {
    let parsedResults = null;

    if (toolName === '590_search_jobs') {
      if (results.jobs && results.jobs.length > 0) {
        // Create jobpy-mcp specific results
        parsedResults = {
          query: {
            searchTerm: results.query.searchTerm || null,
            location: results.query.location || null,
            sitesSearched: results.query.sitesSearched || ['indeed'],
            date: new Date().toISOString(),
          },
          count: results.jobs.length,
          jobs: results.jobs,
        };
      } else {
        // No results returned
        parsedResults = {
          query: {
            searchTerm: results.query?.searchTerm || null,
            location: results.query?.location || null,
            sitesSearched: results.query?.sitesSearched || ['indeed'],
            date: new Date().toISOString(),
            error: 'No results found',
          },
          count: 0,
          jobs: [],
          message: 'No job results found for the given search criteria',
        };
      }

      if (logger.level === 'debug') {
        logger.debug('JobSpy results:', JSON.stringify(parsedResults, null, 2));
      }

      try {
        // Validate against schema
        return jobSearchResultSchema.parse(parsedResults);
      } catch (error) {
        logger.error('Schema validation error', error);
        if (error instanceof ZodError) {
          logger.error('Validation errors:', error.errors);
        }
        return null;
      }
    }

    return parsedResults;
  } catch (error) {
    logger.error('Error processing job search results', error);
    return null;
  }
};
