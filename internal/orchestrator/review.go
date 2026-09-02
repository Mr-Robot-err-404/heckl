package orchestrator

import "time"

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

const (
	StageFetch    = "fetch"
	StageCheckout = "checkout"
	StageSession  = "session"
	StagePrompt   = "prompt"
	StageParse    = "parse"
	StageStore    = "store"
)

const agentName = "pr-reviewer"

var stageOrder = []string{StageFetch, StageCheckout, StageSession, StagePrompt, StageParse, StageStore}

type Stage struct {
	Name       string     `json:"name"`
	Status     Status     `json:"status"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	DurationMS int64      `json:"durationMs"`
	Detail     string     `json:"detail,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type Agent struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
}

type Concern struct {
	File     string `json:"file"`
	Line     *int   `json:"line,omitempty"`
	Side     string `json:"side,omitempty"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type Review struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	PRNumber int    `json:"prNumber"`
	HeadSHA  string `json:"headSha"`

	Status Status  `json:"status"`
	Stages []Stage `json:"stages"`
	Agents []Agent `json:"agents"`

	SessionID           int64  `json:"sessionId,omitempty"`
	OpencodeSessionPath string `json:"opencodeSessionPath,omitempty"`

	Summary  string    `json:"summary,omitempty"`
	Concerns []Concern `json:"concerns"`
	Error    string    `json:"error,omitempty"`

	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

func newReview(id, owner, repo string, prNumber int) *Review {
	stages := make([]Stage, len(stageOrder))
	for i, name := range stageOrder {
		stages[i] = Stage{Name: name, Status: StatusPending}
	}
	return &Review{
		ID:        id,
		Owner:     owner,
		Repo:      repo,
		PRNumber:  prNumber,
		Status:    StatusPending,
		Stages:    stages,
		Agents:    []Agent{{Name: agentName, Status: StatusPending}},
		Concerns:  []Concern{},
		StartedAt: time.Now().UTC(),
	}
}

func (r *Review) clone() *Review {
	c := *r
	c.Stages = append(make([]Stage, 0, len(r.Stages)), r.Stages...)
	c.Agents = append(make([]Agent, 0, len(r.Agents)), r.Agents...)
	c.Concerns = append(make([]Concern, 0, len(r.Concerns)), r.Concerns...)
	return &c
}

func (r *Review) stage(name string) *Stage {
	for i := range r.Stages {
		if r.Stages[i].Name == name {
			return &r.Stages[i]
		}
	}
	return nil
}

func (r *Review) startStage(name string) {
	s := r.stage(name)
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.Status = StatusRunning
	s.StartedAt = &now
}

func (r *Review) endStage(name, detail string, err error) {
	s := r.stage(name)
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.EndedAt = &now
	s.Detail = detail
	if s.StartedAt != nil {
		s.DurationMS = now.Sub(*s.StartedAt).Milliseconds()
	}
	if err != nil {
		s.Status = StatusError
		s.Error = err.Error()
		return
	}
	s.Status = StatusDone
}

func (r *Review) setAgent(name string, status Status) {
	for i := range r.Agents {
		if r.Agents[i].Name == name {
			r.Agents[i].Status = status
			return
		}
	}
}

func (r *Review) finish(err error) {
	now := time.Now().UTC()
	r.EndedAt = &now
	if err != nil {
		r.failRunningStages(err)
		r.Status = StatusError
		r.Error = err.Error()
		r.setAgent(agentName, StatusError)
		return
	}
	r.Status = StatusDone
	r.setAgent(agentName, StatusDone)
}

func (r *Review) failRunningStages(err error) {
	for i := range r.Stages {
		if r.Stages[i].Status == StatusRunning {
			r.endStage(r.Stages[i].Name, "", err)
		}
	}
}
