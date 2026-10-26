package domain

// Fit scores run from 1 (not worth applying) to 5 (strong fit).
const (
	MinFitScore = 1
	MaxFitScore = 5
)

// Fit is a model's assessment of one posting against the user's fit profile. The
// fit check endpoint returns it, and the board scrape passes it on unchanged when
// ingesting the posting.
type Fit struct {
	Score     int      `json:"score"`
	Summary   string   `json:"summary"`
	Strengths []string `json:"strengths"`
	Gaps      []string `json:"gaps"`
	Flags     []string `json:"flags"`
	Model     string   `json:"model"`
}

// FitCheckInput asks for one posting to be scored. Give a URL and the posting is
// fetched from it; give the fields and they are used as they are. Fields given
// alongside a URL override what was fetched, e.g. a description when the page
// cannot be read.
type FitCheckInput struct {
	URL         string `json:"url"`
	JobTitle    string `json:"job_title"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	Description string `json:"description"`
	// FitProfile, when set, is scored against instead of the saved profile, for
	// trying out a profile before saving it.
	FitProfile string `json:"fit_profile"`
}

// FitPosting is the posting a fit check scored, as fetched and/or given.
type FitPosting struct {
	URL         string `json:"url,omitempty"`
	Source      string `json:"source"`
	JobTitle    string `json:"job_title"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	Description string `json:"-"`
	// DescriptionLength is in characters; zero means the posting was scored
	// without a description.
	DescriptionLength int `json:"description_length"`
}

type FitCheckResult struct {
	Fit     *Fit        `json:"fit"`
	Posting *FitPosting `json:"posting"`
}
