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

var reviewStageOrder = []string{StageFetch, StageCheckout, StageStore}

var agentStageOrder = []string{StageSession, StagePrompt, StageParse}

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
	Name                string  `json:"name"`
	Status              Status  `json:"status"`
	Stages              []Stage `json:"stages"`
	OpencodeSessionID   string  `json:"-"`
	OpencodeSessionPath string  `json:"opencodeSessionPath,omitempty"`
}

type Concern struct {
	Agent    string `json:"agent"`
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
	OpencodeSessionID   string `json:"-"`
	OpencodeSessionPath string `json:"opencodeSessionPath,omitempty"`

	Summary  string    `json:"summary,omitempty"`
	Concerns []Concern `json:"concerns"`
	Error    string    `json:"error,omitempty"`

	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

func newStages(names []string) []Stage {
	stages := make([]Stage, len(names))
	for i, name := range names {
		stages[i] = Stage{Name: name, Status: StatusPending}
	}
	return stages
}

func newReview(id, owner, repo string, prNumber int, agents []string) *Review {
	list := make([]Agent, len(agents))
	for i, name := range agents {
		list[i] = Agent{Name: name, Status: StatusPending, Stages: newStages(agentStageOrder)}
	}
	return &Review{
		ID:        id,
		Owner:     owner,
		Repo:      repo,
		PRNumber:  prNumber,
		Status:    StatusPending,
		Stages:    newStages(reviewStageOrder),
		Agents:    list,
		Concerns:  []Concern{},
		StartedAt: time.Now().UTC(),
	}
}

func (r *Review) clone() *Review {
	c := *r
	c.Stages = append(make([]Stage, 0, len(r.Stages)), r.Stages...)
	c.Concerns = append(make([]Concern, 0, len(r.Concerns)), r.Concerns...)
	c.Agents = make([]Agent, 0, len(r.Agents))
	for _, a := range r.Agents {
		a.Stages = append(make([]Stage, 0, len(a.Stages)), a.Stages...)
		c.Agents = append(c.Agents, a)
	}
	return &c
}

func (r *Review) agent(name string) *Agent {
	for i := range r.Agents {
		if r.Agents[i].Name == name {
			return &r.Agents[i]
		}
	}
	return nil
}

func (r *Review) stage(agentName, name string) *Stage {
	stages := r.Stages
	if agentName != "" {
		a := r.agent(agentName)
		if a == nil {
			return nil
		}
		stages = a.Stages
	}
	for i := range stages {
		if stages[i].Name == name {
			return &stages[i]
		}
	}
	return nil
}

func (r *Review) startStage(agentName, name string) {
	s := r.stage(agentName, name)
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.Status = StatusRunning
	s.StartedAt = &now
	if a := r.agent(agentName); a != nil && a.Status == StatusPending {
		a.Status = StatusRunning
	}
}

func (r *Review) endStage(agentName, name, detail string, err error) {
	s := r.stage(agentName, name)
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
		if a := r.agent(agentName); a != nil {
			a.Status = StatusError
		}
		return
	}
	s.Status = StatusDone
	if a := r.agent(agentName); a != nil && lastStage(a.Stages, name) {
		a.Status = StatusDone
	}
}

func lastStage(stages []Stage, name string) bool {
	return len(stages) > 0 && stages[len(stages)-1].Name == name
}

func (r *Review) setAgentSession(name, id, path string) {
	a := r.agent(name)
	if a == nil {
		return
	}
	a.OpencodeSessionID = id
	a.OpencodeSessionPath = path
}

func (r *Review) finish(err error) {
	now := time.Now().UTC()
	r.EndedAt = &now
	if err != nil {
		r.failRunningStages(err)
		r.Status = StatusError
		r.Error = err.Error()
		return
	}
	r.Status = StatusDone
}

func (r *Review) failRunningStages(cause error) {
	for i := range r.Stages {
		if r.Stages[i].Status == StatusRunning {
			r.endStage("", r.Stages[i].Name, "", cause)
		}
	}
	for i := range r.Agents {
		for j := range r.Agents[i].Stages {
			if r.Agents[i].Stages[j].Status == StatusRunning {
				r.endStage(r.Agents[i].Name, r.Agents[i].Stages[j].Name, "", cause)
			}
		}
		if r.Agents[i].Status == StatusRunning || r.Agents[i].Status == StatusPending {
			r.Agents[i].Status = StatusError
		}
	}
}
