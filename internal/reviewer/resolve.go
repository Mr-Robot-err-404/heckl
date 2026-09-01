package reviewer

import (
	"strconv"
	"strings"
)

const (
	SideAdditions = "additions"
	SideDeletions = "deletions"
)

type diffLine struct {
	Side   string
	Number int
	Text   string
}

type diffIndex map[string][]diffLine

func parseDiffIndex(diff string) diffIndex {
	index := diffIndex{}

	var file string
	var oldLine, newLine int

	for _, raw := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(raw, "diff --git "):
			file = ""
			oldLine, newLine = 0, 0
		case strings.HasPrefix(raw, "+++ "):
			file = trimDiffPath(strings.TrimPrefix(raw, "+++ "))
			if file != "" {
				if _, ok := index[file]; !ok {
					index[file] = nil
				}
			}
		case strings.HasPrefix(raw, "@@"):
			oldLine, newLine = parseHunkHeader(raw)
		case file == "" || oldLine == 0 && newLine == 0:
			continue
		case strings.HasPrefix(raw, "+"):
			index[file] = append(index[file], diffLine{SideAdditions, newLine, raw[1:]})
			newLine++
		case strings.HasPrefix(raw, "-"):
			index[file] = append(index[file], diffLine{SideDeletions, oldLine, raw[1:]})
			oldLine++
		case strings.HasPrefix(raw, " "):
			index[file] = append(index[file], diffLine{SideAdditions, newLine, raw[1:]})
			oldLine++
			newLine++
		}
	}

	return index
}

func trimDiffPath(path string) string {
	path = strings.TrimSpace(path)
	if i := strings.IndexByte(path, '\t'); i != -1 {
		path = path[:i]
	}
	if path == "/dev/null" {
		return ""
	}
	if len(path) > 2 && (path[1] == '/') {
		return path[2:]
	}
	return path
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	fields := strings.Fields(header)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return 0, 0
	}
	old, okOld := parseLineStart(fields[1][1:])
	next, okNew := parseLineStart(fields[2][1:])
	if !okOld || !okNew {
		return 0, 0
	}
	return old, next
}

func parseLineStart(spec string) (int, bool) {
	if i := strings.IndexByte(spec, ','); i != -1 {
		spec = spec[:i]
	}
	n, err := strconv.Atoi(spec)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func (idx diffIndex) lookupFile(name string) (string, []diffLine, bool) {
	name = trimDiffPath(name)
	if lines, ok := idx[name]; ok {
		return name, lines, true
	}
	for candidate, lines := range idx {
		if strings.HasSuffix(candidate, "/"+name) || strings.HasSuffix(name, "/"+candidate) {
			return candidate, lines, true
		}
	}
	return name, nil, false
}

func (idx diffIndex) resolve(c Concern) Concern {
	name, lines, ok := idx.lookupFile(c.File)
	c.File = name
	if !ok {
		return fileLevel(c)
	}

	if anchor := strings.TrimSpace(c.Anchor); len(anchor) >= minAnchorLen {
		if match, found := bestAnchorMatch(lines, anchor, c.Side, c.Line); found {
			line := match.Number
			c.Line = &line
			c.Side = match.Side
			return c
		}
	}

	if c.Line != nil {
		for _, l := range lines {
			if l.Number == *c.Line && (c.Side == "" || c.Side == l.Side) {
				c.Side = l.Side
				return c
			}
		}
	}

	return fileLevel(c)
}

type anchorScore int

const (
	scoreNone anchorScore = iota
	scoreCoversAnchor
	scoreContainsAnchor
	scoreExact
)

const minAnchorLen = 4

func bestAnchorMatch(lines []diffLine, anchor, side string, claimed *int) (diffLine, bool) {
	var best diffLine
	bestScore := scoreNone

	for _, l := range lines {
		if side != "" && side != l.Side {
			continue
		}
		score := scoreAnchor(strings.TrimSpace(l.Text), anchor)
		switch {
		case score == scoreNone || score < bestScore:
			continue
		case score > bestScore:
			best, bestScore = l, score
		case claimed != nil && distance(l.Number, *claimed) < distance(best.Number, *claimed):
			best = l
		}
	}

	return best, bestScore != scoreNone
}

func scoreAnchor(text, anchor string) anchorScore {
	switch {
	case text == "":
		return scoreNone
	case text == anchor:
		return scoreExact
	case strings.Contains(text, anchor):
		return scoreContainsAnchor
	case len(text)*2 >= len(anchor) && strings.Contains(anchor, text):
		return scoreCoversAnchor
	}
	return scoreNone
}

func distance(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func fileLevel(c Concern) Concern {
	c.Line = nil
	c.Side = ""
	return c
}
