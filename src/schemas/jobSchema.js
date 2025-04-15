/**
 * JobSpy MCP Server - Job Schema Definitions
 * 
 * This file contains schema definitions for job search results
 * based on the structure of data returned by JobSpy.
 */

import { z } from 'zod';

// Job Search Result Schema
// Based on JobSpy's schema
export const jobSchema = z.object({
  // ID
  id: z.string().optional(),

  // Job Information
  jobTitle: z.string().optional(),
  jobSummary: z.string().nullish(),
  description: z.string().optional(),
  keywords: z.array(z.string()).optional(),
  requiredSkills: z.array(z.string()).optional(),
  niceToHaveSkills: z.array(z.string()).optional(),

  // URLs
  jobUrl: z.string().nullish(),
  jobUrlDirect: z.string().nullish(),

  // Misc Information
  location: z.string().nullish(),
  country: z.string().nullish(),
  state: z.string().nullish(),
  city: z.string().nullish(),
  postalCode: z.string().nullish(),

  // Dates
  datePosted: z.string().nullish(),
  jobType: z.string().nullish(),

  // Salary Information
  salary: z.string().nullish(),
  salaryPeriod: z.string().nullish(),
  salarySource: z.string().nullish(),
  salaryCurrency: z.string().nullish(),
  minAmount: z.number().nullish(),
  maxAmount: z.number().nullish(),

  // Job Categorization
  jobs: z.array(z.string()).nullish(),
  isRemote: z.boolean().nullish(),
  jobLevel: z.string().nullish(),
  jobFunction: z.string().nullish(),
  listingType: z.string().nullish(),

  // Experience
  experience: z.string().nullish(),
  experienceRange: z.string().nullish(),

  // Company Information
  companyName: z.string().optional(),
  companyIndustry: z.string().nullish(),
  companyUrl: z.string().nullish(),
  companyLogo: z.string().nullish(),
  companyUrlDirect: z.string().nullish(),
  companyAddresses: z.array(z.string()).nullish(),
  companyNumEmployees: z.string().nullish(),
  companyRevenue: z.string().nullish(),
  companyDescription: z.string().nullish(),
  companyRating: z.string().nullish(),
  companyReviewsCount: z.string().nullish(),

  // Additional Information
  postingStatus: z.string().nullish(),
  vacancyCount: z.string().nullish(),
  workFromHomeType: z.string().nullish(),
});

export const jobSearchResultSchema = z.object({
  query: z.object({
    searchTerm: z.string().nullable(),
    location: z.string().nullable(),
    sitesSearched: z.array(z.string()),
    date: z.string().nullable(),
    error: z.string().nullable().optional(),
  }),
  count: z.number().optional(),
  jobs: z.array(jobSchema),
  message: z.string().optional(),
});
