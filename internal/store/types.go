package store

const (
	ReviewStatusDone  = "done"
	ReviewStatusError = "error"
)

type ReviewSession struct {
	ID                int64  `json:"id"`
	Owner             string `json:"owner"`
	Repo              string `json:"repo"`
	PRNumber          int    `json:"prNumber"`
	HeadSHA           string `json:"headSha"`
	OpencodeSessionID string `json:"-"`
	Summary           string `json:"summary"`
	Status            string `json:"status"`
	Error             string `json:"error,omitempty"`
	DurationMS        int64  `json:"durationMs"`
	CreatedAt         string `json:"createdAt"`
}

type NewReviewSession struct {
	Owner             string
	Repo              string
	PRNumber          int
	HeadSHA           string
	OpencodeSessionID string
	Summary           string
	Status            string
	Error             string
	DurationMS        int64
}

type RecentReviewSession struct {
	*ReviewSession
	ConcernCount        int64  `json:"concernCount"`
	HighCount           int64  `json:"highCount"`
	OpencodeSessionPath string `json:"opencodeSessionPath,omitempty"`
}

type ReviewConcern struct {
	ID        int64
	SessionID int64
	File      string
	Line      *int
	Side      string
	Severity  string
	Title     string
	Body      string
	CreatedAt string
}
