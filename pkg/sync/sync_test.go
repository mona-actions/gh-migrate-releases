package sync

import (
	"testing"

	"github.com/google/go-github/v62/github"
	"github.com/mona-actions/gh-migrate-releases/internal/api"
)

// TestAssetDetection tests that assets are properly detected as existing or not
func TestAssetDetection(t *testing.T) {
	tests := []struct {
		name           string
		release        *github.RepositoryRelease
		assetName      string
		assetSize      int64
		expectedExists bool
	}{
		{
			name: "Asset exists with matching name and size",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{
						Name: github.String("test-asset.bin"),
						Size: github.Int(1024),
					},
				},
			},
			assetName:      "test-asset.bin",
			assetSize:      1024,
			expectedExists: true,
		},
		{
			name: "Asset does not exist - different name",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{
						Name: github.String("test-asset.bin"),
						Size: github.Int(1024),
					},
				},
			},
			assetName:      "different-asset.bin",
			assetSize:      1024,
			expectedExists: false,
		},
		{
			name: "Asset does not exist - different size",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{
						Name: github.String("test-asset.bin"),
						Size: github.Int(1024),
					},
				},
			},
			assetName:      "test-asset.bin",
			assetSize:      2048,
			expectedExists: false,
		},
		{
			name: "Release has no assets",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{},
			},
			assetName:      "test-asset.bin",
			assetSize:      1024,
			expectedExists: false,
		},
		{
			name:           "Release is nil",
			release:        nil,
			assetName:      "test-asset.bin",
			assetSize:      1024,
			expectedExists: false,
		},
		{
			name: "Multiple assets, target exists",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{
						Name: github.String("asset1.bin"),
						Size: github.Int(1024),
					},
					{
						Name: github.String("asset2.bin"),
						Size: github.Int(2048),
					},
					{
						Name: github.String("target-asset.bin"),
						Size: github.Int(4096),
					},
				},
			},
			assetName:      "target-asset.bin",
			assetSize:      4096,
			expectedExists: true,
		},
		{
			name: "Large file size (1GB)",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{
						Name: github.String("large_file_1.bin"),
						Size: github.Int(1073741824),
					},
				},
			},
			assetName:      "large_file_1.bin",
			assetSize:      1073741824,
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := api.AssetExists(tt.release, tt.assetName, tt.assetSize)
			if exists != tt.expectedExists {
				t.Errorf("AssetExists() = %v, want %v", exists, tt.expectedExists)
			}
		})
	}
}

// TestReleaseExistsLogic tests the logic for checking if releases exist
func TestReleaseExistsLogic(t *testing.T) {
	tests := []struct {
		name            string
		sourceRelease   *github.RepositoryRelease
		existingRelease *github.RepositoryRelease
		shouldMatch     bool
	}{
		{
			name: "Release exists with matching fields",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			shouldMatch: true,
		},
		{
			name: "Release exists but different name",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Different Release"),
				TargetCommitish: github.String("abc123"),
			},
			shouldMatch: false,
		},
		{
			name: "Release exists but different commit",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("def456"),
			},
			shouldMatch: false,
		},
		{
			name: "Release with empty commit hash",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v2.0.0"),
				Name:            github.String("Release v2.0.0"),
				TargetCommitish: github.String(""),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v2.0.0"),
				Name:            github.String("Release v2.0.0"),
				TargetCommitish: github.String(""),
			},
			shouldMatch: true,
		},
		{
			name: "Release with full SHA commit",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v0.0.3"),
				Name:            github.String("Release v0.0.3"),
				TargetCommitish: github.String("b45939ec0c57c712b5ed6ce966b3691b7dda8058"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v0.0.3"),
				Name:            github.String("Release v0.0.3"),
				TargetCommitish: github.String("b45939ec0c57c712b5ed6ce966b3691b7dda8058"),
			},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the matching logic
			nameMatches := tt.existingRelease.GetName() == tt.sourceRelease.GetName()
			commitMatches := tt.existingRelease.GetTargetCommitish() == tt.sourceRelease.GetTargetCommitish()

			matches := nameMatches && commitMatches

			if matches != tt.shouldMatch {
				t.Errorf("Release matching logic = %v, want %v", matches, tt.shouldMatch)
			}
		})
	}
}

// TestLatestReleaseTracking tests that the correct release is tracked as latest
func TestLatestReleaseTracking(t *testing.T) {
	// Create mock releases
	releases := []*github.RepositoryRelease{
		{
			ID:      github.Int64(1),
			TagName: github.String("v1.0.0"),
			Name:    github.String("Release v1.0.0"),
		},
		{
			ID:      github.Int64(2),
			TagName: github.String("v2.0.0"),
			Name:    github.String("Release v2.0.0"),
		},
		{
			ID:      github.Int64(3), // This is the latest
			TagName: github.String("v3.0.0"),
			Name:    github.String("Release v3.0.0"),
		},
	}

	latestReleaseID := int64(3)
	var trackedLatestID int64

	// Simulate the logic in migrateRepositoryReleases
	for _, release := range releases {
		if release.GetID() == latestReleaseID {
			// This would be the new release ID in the target repo
			// In real scenario, this would be newRelease.GetID()
			trackedLatestID = release.GetID()
		}
	}

	if trackedLatestID != latestReleaseID {
		t.Errorf("Latest release tracking failed: got %d, want %d", trackedLatestID, latestReleaseID)
	}
}

// TestLatestReleaseTrackingWithMultipleReleases tests latest release tracking with more complex scenarios
func TestLatestReleaseTrackingWithMultipleReleases(t *testing.T) {
	tests := []struct {
		name            string
		releases        []*github.RepositoryRelease
		latestReleaseID int64
		expectedTracked int64
	}{
		{
			name: "Latest is first release",
			releases: []*github.RepositoryRelease{
				{ID: github.Int64(100), TagName: github.String("v1.0.0")},
				{ID: github.Int64(200), TagName: github.String("v0.9.0")},
				{ID: github.Int64(300), TagName: github.String("v0.8.0")},
			},
			latestReleaseID: 100,
			expectedTracked: 100,
		},
		{
			name: "Latest is middle release",
			releases: []*github.RepositoryRelease{
				{ID: github.Int64(100), TagName: github.String("v0.9.0")},
				{ID: github.Int64(200), TagName: github.String("v1.0.0")},
				{ID: github.Int64(300), TagName: github.String("v0.8.0")},
			},
			latestReleaseID: 200,
			expectedTracked: 200,
		},
		{
			name: "Latest is last release",
			releases: []*github.RepositoryRelease{
				{ID: github.Int64(100), TagName: github.String("v0.8.0")},
				{ID: github.Int64(200), TagName: github.String("v0.9.0")},
				{ID: github.Int64(300), TagName: github.String("v1.0.0")},
			},
			latestReleaseID: 300,
			expectedTracked: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var trackedLatestID int64

			for _, release := range tt.releases {
				if release.GetID() == tt.latestReleaseID {
					trackedLatestID = release.GetID()
				}
			}

			if trackedLatestID != tt.expectedTracked {
				t.Errorf("Latest release tracking failed: got %d, want %d", trackedLatestID, tt.expectedTracked)
			}
		})
	}
}

// TestAssetSkippingLogic tests that assets are properly skipped when they exist
func TestAssetSkippingLogic(t *testing.T) {
	targetRelease := &github.RepositoryRelease{
		Assets: []*github.ReleaseAsset{
			{
				Name: github.String("existing-asset-1.bin"),
				Size: github.Int(1073741824), // 1GB
			},
			{
				Name: github.String("existing-asset-2.bin"),
				Size: github.Int(1073741824),
			},
		},
	}

	sourceAssets := []*github.ReleaseAsset{
		{
			Name: github.String("existing-asset-1.bin"),
			Size: github.Int(1073741824), // Should be skipped
		},
		{
			Name: github.String("new-asset.bin"),
			Size: github.Int(1073741824), // Should be uploaded
		},
		{
			Name: github.String("existing-asset-2.bin"),
			Size: github.Int(1073741824), // Should be skipped
		},
	}

	skippedCount := 0
	uploadCount := 0

	for _, asset := range sourceAssets {
		if api.AssetExists(targetRelease, asset.GetName(), int64(asset.GetSize())) {
			skippedCount++
		} else {
			uploadCount++
		}
	}

	if skippedCount != 2 {
		t.Errorf("Expected 2 assets to be skipped, got %d", skippedCount)
	}

	if uploadCount != 1 {
		t.Errorf("Expected 1 asset to be uploaded, got %d", uploadCount)
	}
}

// TestAssetSkippingWithVariousSizes tests asset skipping with different file sizes
func TestAssetSkippingWithVariousSizes(t *testing.T) {
	tests := []struct {
		name             string
		existingAssets   []*github.ReleaseAsset
		sourceAssets     []*github.ReleaseAsset
		expectedSkipped  int
		expectedToUpload int
	}{
		{
			name: "All assets exist",
			existingAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
				{Name: github.String("file2.bin"), Size: github.Int(2048)},
			},
			sourceAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
				{Name: github.String("file2.bin"), Size: github.Int(2048)},
			},
			expectedSkipped:  2,
			expectedToUpload: 0,
		},
		{
			name:           "No assets exist",
			existingAssets: []*github.ReleaseAsset{},
			sourceAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
				{Name: github.String("file2.bin"), Size: github.Int(2048)},
			},
			expectedSkipped:  0,
			expectedToUpload: 2,
		},
		{
			name: "Mixed - some exist, some don't",
			existingAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
			},
			sourceAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
				{Name: github.String("file2.bin"), Size: github.Int(2048)},
				{Name: github.String("file3.bin"), Size: github.Int(4096)},
			},
			expectedSkipped:  1,
			expectedToUpload: 2,
		},
		{
			name: "Same names but different sizes",
			existingAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(1024)},
			},
			sourceAssets: []*github.ReleaseAsset{
				{Name: github.String("file1.bin"), Size: github.Int(2048)}, // Different size
			},
			expectedSkipped:  0,
			expectedToUpload: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetRelease := &github.RepositoryRelease{
				Assets: tt.existingAssets,
			}

			skippedCount := 0
			uploadCount := 0

			for _, asset := range tt.sourceAssets {
				if api.AssetExists(targetRelease, asset.GetName(), int64(asset.GetSize())) {
					skippedCount++
				} else {
					uploadCount++
				}
			}

			if skippedCount != tt.expectedSkipped {
				t.Errorf("Expected %d assets to be skipped, got %d", tt.expectedSkipped, skippedCount)
			}

			if uploadCount != tt.expectedToUpload {
				t.Errorf("Expected %d assets to be uploaded, got %d", tt.expectedToUpload, uploadCount)
			}
		})
	}
}
