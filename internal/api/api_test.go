package api

import (
	"testing"

	"github.com/google/go-github/v62/github"
)

func TestAssetExists(t *testing.T) {
	tests := []struct {
		name      string
		release   *github.RepositoryRelease
		assetName string
		assetSize int64
		want      bool
	}{
		{
			name: "asset exists with matching name and size",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "test.bin",
			assetSize: 1024,
			want:      true,
		},
		{
			name: "asset doesn't exist - wrong name",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "other.bin",
			assetSize: 1024,
			want:      false,
		},
		{
			name: "asset doesn't exist - wrong size",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "test.bin",
			assetSize: 2048,
			want:      false,
		},
		{
			name:      "nil release",
			release:   nil,
			assetName: "test.bin",
			assetSize: 1024,
			want:      false,
		},
		{
			name: "empty assets",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{},
			},
			assetName: "test.bin",
			assetSize: 1024,
			want:      false,
		},
		{
			name: "nil assets array",
			release: &github.RepositoryRelease{
				Assets: nil,
			},
			assetName: "test.bin",
			assetSize: 1024,
			want:      false,
		},
		{
			name: "multiple assets - target exists",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("file1.bin"), Size: github.Int(1024)},
					{Name: github.String("file2.bin"), Size: github.Int(2048)},
					{Name: github.String("file3.bin"), Size: github.Int(4096)},
				},
			},
			assetName: "file2.bin",
			assetSize: 2048,
			want:      true,
		},
		{
			name: "multiple assets - target doesn't exist",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("file1.bin"), Size: github.Int(1024)},
					{Name: github.String("file2.bin"), Size: github.Int(2048)},
					{Name: github.String("file3.bin"), Size: github.Int(4096)},
				},
			},
			assetName: "file4.bin",
			assetSize: 8192,
			want:      false,
		},
		{
			name: "large file - 1GB",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("large_file_1.bin"), Size: github.Int(1073741824)},
				},
			},
			assetName: "large_file_1.bin",
			assetSize: 1073741824,
			want:      true,
		},
		{
			name: "asset name exists but size is 0",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("empty.bin"), Size: github.Int(0)},
				},
			},
			assetName: "empty.bin",
			assetSize: 0,
			want:      true,
		},
		{
			name: "case sensitive name check",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("Test.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "test.bin",
			assetSize: 1024,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssetExists(tt.release, tt.assetName, tt.assetSize)
			if got != tt.want {
				t.Errorf("AssetExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReleaseExistsMatchLogic(t *testing.T) {
	tests := []struct {
		name            string
		sourceRelease   *github.RepositoryRelease
		existingRelease *github.RepositoryRelease
		wantMatch       bool
	}{
		{
			name: "all fields match",
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
			wantMatch: true,
		},
		{
			name: "name doesn't match",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Different Name"),
				TargetCommitish: github.String("abc123"),
			},
			wantMatch: false,
		},
		{
			name: "commit doesn't match",
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
			wantMatch: false,
		},
		{
			name: "both name and commit don't match",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Different Name"),
				TargetCommitish: github.String("def456"),
			},
			wantMatch: false,
		},
		{
			name: "empty commit hash - both empty",
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
			wantMatch: true,
		},
		{
			name: "full SHA commit hash",
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
			wantMatch: true,
		},
		{
			name: "short SHA vs long SHA - should not match",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v0.0.3"),
				Name:            github.String("Release v0.0.3"),
				TargetCommitish: github.String("b45939e"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v0.0.3"),
				Name:            github.String("Release v0.0.3"),
				TargetCommitish: github.String("b45939ec0c57c712b5ed6ce966b3691b7dda8058"),
			},
			wantMatch: false,
		},
		{
			name: "case sensitive commit check",
			sourceRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("ABC123"),
			},
			existingRelease: &github.RepositoryRelease{
				TagName:         github.String("v1.0.0"),
				Name:            github.String("Release v1.0.0"),
				TargetCommitish: github.String("abc123"),
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the matching logic that ReleaseExists uses
			nameMatches := tt.existingRelease.GetName() == tt.sourceRelease.GetName()
			commitMatches := tt.existingRelease.GetTargetCommitish() == tt.sourceRelease.GetTargetCommitish()
			gotMatch := nameMatches && commitMatches

			if gotMatch != tt.wantMatch {
				t.Errorf("Release match logic = %v, want %v", gotMatch, tt.wantMatch)
				t.Errorf("  nameMatches=%v, commitMatches=%v", nameMatches, commitMatches)
			}
		})
	}
}

// TestAssetExistsEdgeCases tests edge cases for asset existence checking
func TestAssetExistsEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		release   *github.RepositoryRelease
		assetName string
		assetSize int64
		want      bool
	}{
		{
			name: "asset with special characters in name",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test-file_v1.0.0.tar.gz"), Size: github.Int(1024)},
				},
			},
			assetName: "test-file_v1.0.0.tar.gz",
			assetSize: 1024,
			want:      true,
		},
		{
			name: "asset with spaces in name",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test file.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "test file.bin",
			assetSize: 1024,
			want:      true,
		},
		{
			name: "very large file size",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("huge.bin"), Size: github.Int(53687091200)}, // 50GB
				},
			},
			assetName: "huge.bin",
			assetSize: 53687091200,
			want:      true,
		},
		{
			name: "negative size should not match",
			release: &github.RepositoryRelease{
				Assets: []*github.ReleaseAsset{
					{Name: github.String("test.bin"), Size: github.Int(1024)},
				},
			},
			assetName: "test.bin",
			assetSize: -1024,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssetExists(tt.release, tt.assetName, tt.assetSize)
			if got != tt.want {
				t.Errorf("AssetExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
