export type Repo = {
  ID: number
  Owner: string
  Name: string
  AddedAt: string
}

export type PR = {
  ID: number
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
  SyncedAt: string
}

export type PRFile = {
  ID: number
  PrID: number
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
