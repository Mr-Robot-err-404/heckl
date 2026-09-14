package server

import (
	"context"
	"fmt"
	"log/slog"
)

func (s *Server) SweepOpencodeSessions(ctx context.Context) (string, error) {
	rows, err := s.store.UnreachableOpencodeSessions(ctx)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "no unreachable sessions", nil
	}

	deleted := 0
	for _, row := range rows {
		if err := s.oc.DeleteSession(row.SessionID); err != nil {
			slog.Warn("sessions: delete failed", "session_id", row.SessionID, "err", err)
			continue
		}
		if err := s.store.ForgetOpencodeSession(ctx, row.SessionID); err != nil {
			slog.Error("sessions: forget failed", "session_id", row.SessionID, "err", err)
			continue
		}
		slog.Info("sessions: deleted unreachable session",
			"session_id", row.SessionID, "owner", row.Owner, "repo", row.Repo,
			"pr", row.PrNumber, "agent", row.Agent)
		deleted++
	}
	return fmt.Sprintf("deleted %d of %d unreachable sessions", deleted, len(rows)), nil
}
