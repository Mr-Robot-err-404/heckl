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
