package opencode

func SessionPath(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	return "/session/" + sessionID
}
