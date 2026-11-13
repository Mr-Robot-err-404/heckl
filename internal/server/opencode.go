package server

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *Server) handleOpencodeSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	origin, err := url.Parse(r.URL.Query().Get("origin"))
	requestURL := &url.URL{Host: r.Host}
	if err != nil || origin.User != nil || origin.Hostname() != requestURL.Hostname() || origin.Port() != "4420" || (origin.Scheme != "http" && origin.Scheme != "https") || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		http.Error(w, "Invalid OpenCode origin", http.StatusBadRequest)
		return
	}
	id := r.PathValue("session")
	if !strings.HasPrefix(id, "ses") || strings.ContainsAny(id, "/?#") {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	cookies, err := s.oc.BrowserCookies(ctx, origin.Host)
	if err != nil {
		slog.Error("opencode: browser authentication failed", "err", err)
		http.Error(w, "Could not connect to OpenCode. Check the Heckl server logs.", http.StatusBadGateway)
		return
	}
	for _, cookie := range cookies {
		cookie.Secure = origin.Scheme == "https"
		http.SetCookie(w, cookie)
	}
	serverKey := base64.RawURLEncoding.EncodeToString([]byte(strings.TrimRight(origin.String(), "/")))
	origin.Path = "/server/" + serverKey + "/session/" + id
	http.Redirect(w, r, origin.String(), http.StatusSeeOther)
}
