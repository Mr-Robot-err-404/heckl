package server

import "testing"

func TestNotificationPRState(t *testing.T) {
	tests := []struct {
		state  string
		draft  bool
		merged bool
		want   string
	}{
		{state: "open", want: "open"},
		{state: "open", draft: true, want: "draft"},
		{state: "closed", want: "closed"},
		{state: "closed", merged: true, want: "merged"},
	}

	for _, test := range tests {
		if got := notificationPRState(test.state, test.draft, test.merged); got != test.want {
			t.Errorf("notificationPRState(%q, %t, %t) = %q, want %q", test.state, test.draft, test.merged, got, test.want)
		}
	}
}
