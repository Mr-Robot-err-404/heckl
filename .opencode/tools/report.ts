import { tool } from "@opencode-ai/plugin"

const concern = tool.schema.object({
  file: tool.schema.string().describe("Path exactly as it appears in the diff header."),
  line: tool.schema.number().int().describe("Line number in the file this concern is about."),
  side: tool.schema
    .enum(["additions", "deletions"])
    .describe(
      "additions if the line is added or unchanged context, deletions if it is a removed line.",
    ),
  anchor: tool.schema
    .string()
    .describe(
      "The exact source text of that line, copied verbatim from the diff without the leading +/-/space marker. This is what pins the concern to a location, so copy it precisely.",
    ),
  severity: tool.schema.enum(["low", "medium", "high"]),
  title: tool.schema.string(),
  body: tool.schema.string(),
})

export default tool({
  description:
    "Submit the finished review. Call this exactly once, as the final action. This is the only way to deliver a review — anything written as ordinary prose is discarded.",
  args: {
    summary: tool.schema
      .string()
      .describe("What this PR is trying to do, in one or two sentences. Plain and specific."),
    concerns: tool.schema
      .array(concern)
      .describe("Only concerns you are confident about. An empty array is a valid, complete review."),
  },
  async execute(args) {
    return `Review recorded: ${args.concerns.length} concern${args.concerns.length === 1 ? "" : "s"}.`
  },
})
