export type Repo = {
  ID: number
  Owner: string
  Name: string
  AddedAt: string
}

export type GitHubUser = {
  login: string
  avatar: string
}

export type PRReviewSummary = {
  prNumber: number
  status: "done" | "error"
  createdAt: string
  concernCount: number
  highCount: number
}

export type PR = {
  Owner: string
  Repo: string
  Number: number
  Title: string
  Body: string
  State: string
  Author: string
  HtmlUrl: string
  Draft: boolean
  CreatedAt: string
  UpdatedAt: string
  requestedReviewers?: GitHubUser[]
  approvals?: GitHubUser[]
  changesRequested?: GitHubUser[]
  viewerApproved: boolean
  viewerHasReviewed: boolean
  review?: PRReviewSummary
}

export type PRFile = {
  Sha: string
  Filename: string
  Status: string
  Additions: number
  Deletions: number
  Changes: number
  Patch: string
}

export type PRDetail = {
  pr: PR
  files: PRFile[]
}

export type Tab = "description" | "files" | "review"

export type ReviewStatus = "pending" | "running" | "done" | "error" | "cancelled"

export type ReviewStage = {
  name: string
  status: ReviewStatus
  startedAt?: string
  endedAt?: string
  durationMs: number
  detail?: string
  error?: string
}

export type ReviewAgent = {
  name: string
  status: ReviewStatus
  stages: ReviewStage[]
  error?: string
  summary?: string
  durationMs: number
  opencodeSessionPath?: string
}

export type Concern = {
  agent: string
  file: string
  line?: number
  side?: "additions" | "deletions"
  severity: "low" | "medium" | "high"
  title: string
  body: string
}

export type RankedConcern = Concern & { rank: number }

export type ConcernTarget = {
  file: string
  line: number
  side: "additions" | "deletions"
  rank: number
}

export type ReviewerNote = {
  id: number
  body: string
  state?: string
  file?: string
  line?: number
  side?: "additions" | "deletions"
  outdated?: boolean
  reply?: boolean
  url?: string
  createdAt: string
}

export type ReviewerThread = {
  user: GitHubUser
  bot?: boolean
  notes: ReviewerNote[]
}

export type ReviewHistoryRow = {
  id: number
  owner: string
  repo: string
  prNumber: number
  headSha: string
  summary: string
  status: "done" | "error"
  error?: string
  durationMs: number
  createdAt: string
  concernCount: number
  highCount: number
  mediumCount: number
  lowCount: number
  opencodeSessionPath?: string
}

export type ReviewHistoryPage = {
  sessions: ReviewHistoryRow[]
  hasMore: boolean
}

export type Review = {
  id: string
  owner: string
  repo: string
  prNumber: number
  headSha: string
  status: ReviewStatus
  stages: ReviewStage[]
  agents: ReviewAgent[]
  sessionId?: number
  opencodeSessionPath?: string
  summary?: string
  concerns: Concern[]
  error?: string
  startedAt: string
  endedAt?: string
}

export type AgentConfig = {
  name: string
  model: string
  prompt: string
  defaultModel: string
}

export type ModelOption = {
  ref: string
  name: string
  provider: string
}

export type AgentConfigPage = {
  agents: AgentConfig[]
  models: ModelOption[]
}

export type TmuxPick = {
  file: string
  line?: number
}

export type TmuxSession = {
  session: string
  attach: string
  opened: string[]
  skipped?: string[]
}

export type TmuxLiveSession = {
  session: string
  attach: string
  windows: number
  worktree: string
  headSha: string
}

export type DiffBlobFile = {
  name: string
  contents: string
}

export type DiffBlob = {
  oldFile: DiffBlobFile | null
  newFile: DiffBlobFile | null
}
