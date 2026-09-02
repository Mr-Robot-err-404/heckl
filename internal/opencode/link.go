package opencode

import "encoding/base64"

func SessionPath(directory, sessionID string) string {
	if directory == "" || sessionID == "" {
		return ""
	}
	dir := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(directory))
	return "/" + dir + "/session/" + sessionID
}
