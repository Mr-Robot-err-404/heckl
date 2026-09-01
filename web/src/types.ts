export type Repo = {
  ID: number
  Owner: string
  Name: string
  AddedAt: string
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

export type Tab = "description" | "review"

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
}

export type Concern = {
  file: string
  line?: number
  severity: "low" | "medium" | "high"
  title: string
  body: string
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
  summary?: string
  concerns: Concern[]
  error?: string
  startedAt: string
  endedAt?: string
}
