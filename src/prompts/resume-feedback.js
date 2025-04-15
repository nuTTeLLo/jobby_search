/* eslint-disable max-len */
import { z } from 'zod';

/**
 * Resume Feedback Schema - Parameters for resume feedback generation
 */
export const resumeFeedbackSchema = z.object({
  resumeText: z.string()
    .describe('The full text of the resume to analyze'),
  targetRole: z.string()
    .describe('The specific job role the resume is targeting'),
  targetIndustry: z.string()
    .describe('The industry the job seeker is targeting'),
  experienceLevel: z.string()
    .describe('The experience level of the job seeker (e.g., entry-level, mid-level, senior)'),
});

/**
 * Prompt template for resume feedback
 */
export const resumeFeedbackPrompt = `
You are a professional resume reviewer with expertise in helping job seekers improve their resumes for specific roles and industries.
Provide comprehensive and constructive feedback on the resume text provided.
`;

/**
 * System message for resume feedback
 */
export const resumeFeedbackSystemMessage = {
  role: 'system',
  content: resumeFeedbackPrompt,
};

/**
 * Create resume feedback request
 */
export const createResumeFeedbackRequest = (inputs) => {
  return {
    messages: [
      resumeFeedbackSystemMessage,
      {
        role: 'user',
        content: `
Please review the following resume for a ${inputs.experienceLevel} professional targeting a ${inputs.targetRole} position in the ${inputs.targetIndustry} industry.
Provide specific, actionable feedback in these categories:
1. Overall impression and effectiveness
2. Content and relevance to target role
3. Format and structure
4. Keywords and ATS optimization
5. Strengths and areas for improvement
6. Suggested edits or additions

Resume:
${inputs.resumeText}
        `,
      },
    ],
  };
};
