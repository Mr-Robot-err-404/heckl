package store

type ReviewSession struct {
	ID                int64
	Owner             string
	Repo              string
	PRNumber          int
	HeadSHA           string
	OpencodeSessionID string
	Summary           string
	CreatedAt         string
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
