package api

import (
	"testing"
	"time"

	"github.com/google/go-github/v62/github"
)

func TestSortReleasesByCreationDate(t *testing.T) {
	// Create test data with known dates
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)

	// Create test releases in reverse chronological order (newest first)
	releases := []*github.RepositoryRelease{
		{
			ID:        github.Int64(3),
			TagName:   github.String("v3.0.0"),
			CreatedAt: &github.Timestamp{Time: time3}, // Newest
		},
		{
			ID:        github.Int64(2),
			TagName:   github.String("v2.0.0"),
			CreatedAt: &github.Timestamp{Time: time2}, // Middle
		},
		{
			ID:        github.Int64(1),
			TagName:   github.String("v1.0.0"),
			CreatedAt: &github.Timestamp{Time: time1}, // Oldest
		},
	}

	// Apply the same sorting logic from GetSourceRepositoryReleases
	sortReleasesByCreationDate(releases)

	// Verify they are now in ascending order (oldest first)
	if releases[0].GetID() != 1 {
		t.Errorf("Expected first release to have ID 1, got %d", releases[0].GetID())
	}
	if releases[1].GetID() != 2 {
		t.Errorf("Expected second release to have ID 2, got %d", releases[1].GetID())
	}
	if releases[2].GetID() != 3 {
		t.Errorf("Expected third release to have ID 3, got %d", releases[2].GetID())
	}

	// Verify the order by tag names
	expectedOrder := []string{"v1.0.0", "v2.0.0", "v3.0.0"}
	for i, expected := range expectedOrder {
		if releases[i].GetTagName() != expected {
			t.Errorf("Expected release %d to have tag %s, got %s", i, expected, releases[i].GetTagName())
		}
	}
}

func TestSortReleasesWithNilDates(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	// Create test releases with some nil dates
	releases := []*github.RepositoryRelease{
		{
			ID:        github.Int64(3),
			TagName:   github.String("v3.0.0"),
			CreatedAt: &github.Timestamp{Time: time2},
		},
		{
			ID:        github.Int64(2),
			TagName:   github.String("v2.0.0"),
			CreatedAt: nil, // Nil date should come first
		},
		{
			ID:        github.Int64(1),
			TagName:   github.String("v1.0.0"),
			CreatedAt: &github.Timestamp{Time: time1},
		},
	}

	// Apply the sorting logic
	sortReleasesByCreationDate(releases)

	// Verify that nil dates come first, then sorted by date
	if releases[0].GetID() != 2 {
		t.Errorf("Expected first release (nil date) to have ID 2, got %d", releases[0].GetID())
	}
	if releases[1].GetID() != 1 {
		t.Errorf("Expected second release to have ID 1, got %d", releases[1].GetID())
	}
	if releases[2].GetID() != 3 {
		t.Errorf("Expected third release to have ID 3, got %d", releases[2].GetID())
	}
}