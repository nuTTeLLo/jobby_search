// Fit scores run 1-5, set by the daily scrape against the user's fit profile.
// Postings at or below this score are what "Hide low fits" hides.
export const LOW_FIT_MAX = 2;

export const isLowFit = (job) => job.fit_score != null && job.fit_score <= LOW_FIT_MAX;
