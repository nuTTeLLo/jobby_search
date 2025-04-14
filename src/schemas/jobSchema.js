/**
 * JobSpy MCP Server - Job Schema Definitions
 * 
 * This file contains schema definitions for job search results
 * based on the structure of data returned by JobSpy.
 */

const Joi = require('joi');

/**
 * Schema for a single job result
 */
const jobSchema = Joi.object({
  // Core job identification
  id: Joi.string().description('Unique job identifier'),
  site: Joi.string().description('Job listing site (indeed, linkedin, etc.)'),
  
  // URLs
  job_url: Joi.string().uri().description('URL to the job posting on the job site'),
  job_url_direct: Joi.string().uri().allow(null).description('Direct URL to employer\'s job posting'),
  
  // Basic job information
  title: Joi.string().required().description('Job title'),
  company: Joi.string().allow(null).description('Company name'),
  location: Joi.string().description('Job location'),
  date_posted: Joi.number().description('Timestamp of when the job was posted'),
  job_type: Joi.string().allow(null).description('Type of job (fulltime, contract, etc.)'),
  description: Joi.string().allow(null).description('Full job description'),
  
  // Salary information
  salary_source: Joi.string().allow(null).description('Source of salary information'),
  interval: Joi.string().allow(null).description('Salary interval (yearly, hourly, etc.)'),
  min_amount: Joi.number().allow(null).description('Minimum salary amount'),
  max_amount: Joi.number().allow(null).description('Maximum salary amount'),
  currency: Joi.string().allow(null).description('Salary currency'),
  
  // Job metadata
  is_remote: Joi.boolean().allow(null).description('Whether the job is remote'),
  job_level: Joi.string().allow(null).description('Job level (entry, senior, etc.)'),
  job_function: Joi.string().allow(null).description('Job function/department'),
  listing_type: Joi.string().allow(null).description('Type of job listing'),
  emails: Joi.array().items(Joi.string()).allow(null).description('Contact emails found in job posting'),
  skills: Joi.array().items(Joi.string()).allow(null).description('Skills required for the job'),
  experience_range: Joi.string().allow(null).description('Experience range required'),
  
  // Company information
  company_industry: Joi.string().allow(null).description('Company industry'),
  company_url: Joi.string().uri().allow(null).description('Company website URL on job site'),
  company_logo: Joi.string().allow(null).description('URL to company logo'),
  company_url_direct: Joi.string().uri().allow(null).description('Direct URL to company website'),
  company_addresses: Joi.string().allow(null).description('Company addresses'),
  company_num_employees: Joi.string().allow(null).description('Company size by number of employees'),
  company_revenue: Joi.string().allow(null).description('Company revenue'),
  company_description: Joi.string().allow(null).description('Company description'),
  company_rating: Joi.number().allow(null).description('Company rating'),
  company_reviews_count: Joi.number().allow(null).description('Number of company reviews'),
  
  // Additional fields
  vacancy_count: Joi.number().allow(null).description('Number of similar vacancies'),
  work_from_home_type: Joi.string().allow(null).description('Work from home type/policy')
});

/**
 * Schema for the complete job search response
 */
const jobSearchResponseSchema = Joi.object({
  jobs: Joi.array().items(jobSchema).description('List of jobs found'),
  metadata: Joi.object({
    count: Joi.number().description('Total number of jobs found'),
    search_term: Joi.string().description('Search term used'),
    location: Joi.string().allow(null).description('Location used for search'),
    sites_searched: Joi.array().items(Joi.string()).description('List of job sites searched'),
    timestamp: Joi.string().isoDate().description('Timestamp of the search')
  }).description('Metadata about the search')
});

module.exports = {
  jobSchema,
  jobSearchResponseSchema
};