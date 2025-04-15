import { execSync } from 'child_process';

import logger from './logger.js';

/**
 * @typedef {Object} JobSearchParams
 * @property {string} [site_names] - Names of job sites to search
 * @property {string} [search_term] - Term to search for
 * @property {string} [location] - Job location
 * @property {string} [google_search_term] - Term for Google job search
 * @property {number} [results_wanted] - Number of results to return
 * @property {number} [hours_old] - Filter for jobs posted within specified hours
 * @property {string} [country_indeed] - Country code for Indeed search
 * @property {boolean} [linkedin_fetch_description] - Whether to fetch LinkedIn job descriptions
 * @property {string} [proxies] - Comma-separated list of proxies
 * @property {'json'|'csv'} [format] - Output format: JSON or CSV
 */

/**
 * Handler for the search_jobs MCP tool
 * @param {JobSearchParams} params - Search parameters
 * @returns {Promise<object>} Search results
 */
export async function searchJobsHandler(params) {
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
      jobs: data,
      count: data.length || 0,
      message: 'Job search completed successfully'
    };
  } catch (error) {
    logger.error('Error in searchJobsHandler', { error: error.message, result });
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
  if (params.site_name) args.push('--site_names', `"${params.site_names}"`);
  if (params.search_term) args.push('--search_term', `"${params.search_term}"`);
  if (params.location) args.push('--location', `"${params.location}"`);
  if (params.google_search_term) args.push('--google_search_term', `"${params.google_search_term}"`);
  if (params.results_wanted) args.push('--results_wanted', `"${params.results_wanted}"`);
  if (params.hours_old) args.push('--hours_old', `"${params.hours_old}"`);
  if (params.country_indeed) args.push('--country_indeed', `"${params.country_indeed}"`);
  if (!!params.linkedin_fetch_description) args.push('--linkedin_fetch_description');
  if (params.proxies) args.push('--proxies', `"${params.proxies}"`);
  args.push('--format', 'json');  
  return args;
}
