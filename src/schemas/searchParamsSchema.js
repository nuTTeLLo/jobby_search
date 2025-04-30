import { z } from 'zod';

export const searchParams = {
  siteNames: z
    .union([
      z
        .string()
        .describe(
          'Comma-separated list of job sites to search. Options: indeed,linkedin,zip_recruiter,glassdoor,google,bayt,naukri'
        ),
      z
        .array(z.string())
        .describe(
          'Array of job sites to search. Options: indeed,linkedin,zip_recruiter,glassdoor,google,bayt,naukri'
        ),
    ])
    .transform((val) => {
      // If it's already a string, return it as is
      if (typeof val === 'string') {
        return val;
      }
      // If it's an array, join it with commas
      if (Array.isArray(val)) {
        return val.join(',');
      }
      return val;
    })
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
      }
    )
    .default('indeed'),
  searchTerm: z
    .string()
    .describe('Search term for jobs')
    .default('software engineer'),
  location: z
    .string()
    .describe('Location for job search')
    .default('remote'),
  googleSearchTerm: z
    .string()
    .nullable()
    .describe('Google specific search term')
    .default(null),
  resultsWanted: z
    .number()
    .int()
    .describe('Number of results wanted')
    .transform(val => val === 0 ? 20 : val)
    .default(20),
  hoursOld: z
    .number()
    .int()
    .describe('How many hours old the jobs can be')
    .transform(val => val === 0 ? 72 : val)
    .default(72),
  countryIndeed: z
    .string()
    .describe('Country for Indeed search')
    .default('USA'),
  linkedinFetchDescription: z
    .any()
    .describe(
      'Whether to fetch LinkedIn job descriptions (slower). Accepts any truthy value.'
    )
    .transform((val) => {
      // Convert any truthy value to boolean
      if (typeof val === 'string') {
        // For strings, check for common "true" values
        return ['true', 'yes', '1', 'on', 'y'].includes(val.toLowerCase());
      }
      // For other types, use Boolean conversion
      return Boolean(val);
    })
    .default(false),
  proxies: z
    .string()
    .nullable()
    .describe('Comma-separated list of proxies')
    .default(null),
  format: z.enum(['json', 'csv']).describe('Output format').default('json'),
};
