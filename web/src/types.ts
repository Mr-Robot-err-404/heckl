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

export type ReviewStatus = "pending" | "running" | "done" | "error"

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
  nonce: number
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
