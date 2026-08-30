package store

type ReviewSession struct {
	ID                int64
	Owner             string
	Repo              string
	PRNumber          int
	HeadSHA           string
	OpencodeSessionID string
	CreatedAt         string
}

type ReviewConcern struct {
	ID        int64
	SessionID int64
	File      string
	Line      *int
	Severity  string
	Title     string
	Body      string
	CreatedAt string
}
