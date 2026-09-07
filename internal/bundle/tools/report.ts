const concern = {
  type: "object",
  properties: {
    file: {
      type: "string",
      description: "Path exactly as it appears in the diff header.",
    },
    line: {
      type: "integer",
      description: "Line number in the file this concern is about.",
    },
    side: {
      type: "string",
      enum: ["additions", "deletions"],
      description:
        "additions if the line is added or unchanged context, deletions if it is a removed line.",
    },
    anchor: {
      type: "string",
      description:
        "The exact source text of that line, copied verbatim from the diff without the leading +/-/space marker. This is what pins the concern to a location, so copy it precisely.",
    },
    severity: { type: "string", enum: ["low", "medium", "high"] },
    title: { type: "string" },
    body: { type: "string" },
  },
  required: ["file", "line", "side", "anchor", "severity", "title", "body"],
  additionalProperties: false,
}

export default {
  description:
    "Submit the finished review. Call this exactly once, as the final action. This is the only way to deliver a review - anything written as ordinary prose is discarded.",
  args: {
    summary: {
      type: "string",
      description:
        "What this PR is trying to do, in one or two sentences. Plain and specific.",
    },
    concerns: {
      type: "array",
      items: concern,
      description:
        "Only concerns you are confident about. An empty array is a valid, complete review.",
    },
  },
  async execute(args: { summary?: string; concerns?: unknown[] }) {
    const count = Array.isArray(args?.concerns) ? args.concerns.length : 0
    return `Review recorded: ${count} concern${count === 1 ? "" : "s"}.`
  },
}
