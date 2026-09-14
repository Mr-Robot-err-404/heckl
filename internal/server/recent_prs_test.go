package server

import (
	"reflect"
	"testing"
)

func TestNormalizeRecentPRFilter(t *testing.T) {
	filter, err := normalizeRecentPRFilter(recentPRFilter{
		Organizations: []string{"zeta", " alpha ", "zeta"},
		Repositories:  []string{"zeta/two", "alpha/one", "zeta/two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alpha", "zeta"}; !reflect.DeepEqual(filter.Organizations, want) {
		t.Fatalf("organizations = %v, want %v", filter.Organizations, want)
	}
	if want := []string{"alpha/one", "zeta/two"}; !reflect.DeepEqual(filter.Repositories, want) {
		t.Fatalf("repositories = %v, want %v", filter.Repositories, want)
	}
}

func TestNormalizeRecentPRFilterRejectsMalformedValues(t *testing.T) {
	tests := []recentPRFilter{
		{Organizations: []string{"owner/repo"}},
		{Repositories: []string{"repo"}},
		{Repositories: []string{"owner/"}},
	}
	for _, filter := range tests {
		if _, err := normalizeRecentPRFilter(filter); err == nil {
			t.Fatalf("normalizeRecentPRFilter(%+v) succeeded", filter)
		}
	}
}
