import { z } from 'zod';

export const searchParams = {
  siteNames: z
    .string()
    .describe(
      'Comma-separated list of job sites to search. Options: indeed,linkedin,zip_recruiter,glassdoor,google,bayt,naukri',
    )
    .refine(
      (val) => {
        const sites = val.split(',').map((site) => site.trim());
        const validSites = [
          'indeed',
          'linkedin',
          'zip_recruiter',
          'glassdoor',
          'google',
          'bayt',
          'naukri',
        ];
        return sites.every((site) => validSites.includes(site));
      },
      {
        message:
          'Invalid site names. Allowed values: indeed, linkedin, zip_recruiter, glassdoor, google, bayt, naukri',
      },
    )
    .default('indeed'),
  searchTerm: z
    .string()
    .describe('Search term for jobs')
    .default('software engineer'),
  location: z
    .string()
    .describe('Location for job search')
    .default('San Francisco, CA'),
  googleSearchTerm: z
    .string()
    .nullable()
    .describe('Google specific search term')
    .default(null),
  resultsWanted: z
    .number()
    .int()
    .describe('Number of results wanted')
    .default(20),
  hoursOld: z
    .number()
    .int()
    .describe('How many hours old the jobs can be')
    .default(72),
  countryIndeed: z
    .string()
    .describe('Country for Indeed search')
    .default('USA'),
  linkedinFetchDescription: z
    .boolean()
    .describe('Whether to fetch LinkedIn job descriptions (slower)')
    .default(false),
  proxies: z
    .string()
    .nullable()
    .describe('Comma-separated list of proxies')
    .default(null),
  format: z.enum(['json', 'csv']).describe('Output format').default('json'),
};
