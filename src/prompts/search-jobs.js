import { z } from 'zod';

/**
 * Search Jobs Schema - Parameters for job search
 */
export const searchJobsSchema = z.object({
  query: z.string().describe('Job search query'),
});

/**
 * Prompt template for job search
 */
export const searchJobsPrompt = `
You are a helpful assistant who helps users search for jobs by understanding their job search requirements.
Based on the user query, extract the search parameters needed to find relevant jobs.
`;

/**
 * System message for job search
 */
export const searchJobsSystemMessage = {
  role: 'system',
  content: searchJobsPrompt,
};

/**
 * Create job search request
 */
export const createSearchJobsRequest = (inputs) => {
  return {
    messages: [
      searchJobsSystemMessage,
      {
        role: 'user',
        content: `
Extract search parameters from the following job search query: "${inputs.query}"

Provide the following information:
1. Job title or keywords
2. Location (if specified, otherwise assume "Remote")
3. Any specific companies mentioned
4. Job type preferences (full-time, part-time, contract, etc.)
5. Experience level requirements (entry, mid, senior)
        `,
      },
    ],
  };
};
