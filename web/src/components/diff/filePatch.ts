// Extract the raw patch for one file, including its git header and hunk lines.
export function filePatch(patch: string, name: string): string | null {
  const headers = [...patch.matchAll(/^diff --git /gm)]
  for (let i = 0; i < headers.length; i++) {
    const start = headers[i].index
    const end = headers[i + 1]?.index ?? patch.length
    const section = patch.slice(start, end).trimEnd()
    const hunk = section.search(/^@@ /m)
    const header = hunk === -1 ? section : section.slice(0, hunk)
    const newName = header.match(/^\+\+\+ b\/(.*)$/m)?.[1]
      ?? header.match(/^rename to (.*)$/m)?.[1]
      ?? header.match(/^copy to (.*)$/m)?.[1]
    const oldName = header.match(/^--- a\/(.*)$/m)?.[1]
    const gitPaths = header.match(/^diff --git a\/(.*) b\/(.*)$/m)
    if (newName === name || (header.includes("+++ /dev/null") && oldName === name) ||
      (!newName && gitPaths?.[2] === name)) {
      return `${section}\n`
    }
  }
  return null
}
