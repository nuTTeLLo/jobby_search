const { jobSchema, jobSearchResponseSchema } = require('../src/schemas/jobSchema');
const fs = require('fs').promises;
const path = require('path');

describe('Job Schema', () => {
  let jobsData;

  beforeAll(async () => {
    try {
      // Load sample data from jobs.json
      const jobsPath = path.join(__dirname, '../../jobSpy/jobs.json');
      const jobsContent = await fs.readFile(jobsPath, 'utf-8');
      jobsData = JSON.parse(jobsContent);
    } catch (error) {
      console.error('Error loading test data:', error);
      // Provide fallback test data if file can't be loaded
      jobsData = [{
        id: 'test-123',
        site: 'indeed',
        job_url: 'https://www.indeed.com/viewjob?jk=test123',
        title: 'Software Engineer',
        company: 'Test Company',
        location: 'San Francisco, CA'
      }];
    }
  });

  test('validates a single job object', () => {
    const job = jobsData[0];
    const { error } = jobSchema.validate(job);
    expect(error).toBeUndefined();
  });

  test('validates a complete job search response', () => {
    const response = {
      jobs: jobsData,
      metadata: {
        count: jobsData.length,
        search_term: 'software engineer',
        location: 'San Francisco',
        sites_searched: ['indeed'],
        timestamp: new Date().toISOString()
      }
    };

    const { error } = jobSearchResponseSchema.validate(response);
    expect(error).toBeUndefined();
  });

  test('identifies missing required fields', () => {
    const invalidJob = {
      // Missing title which is required
      id: 'test-456',
      company: 'Invalid Test'
    };

    const { error } = jobSchema.validate(invalidJob);
    expect(error).toBeDefined();
    expect(error.message).toContain('"title" is required');
  });

  test('validates field types', () => {
    const invalidTypeJob = {
      id: 12345, // Should be string
      title: 'Software Engineer',
      is_remote: 'yes' // Should be boolean
    };

    const { error } = jobSchema.validate(invalidTypeJob);
    expect(error).toBeDefined();
    expect(error.message).toContain('"id" must be a string');
  });
});