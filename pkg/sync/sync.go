package sync

import (
	"strings"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/mona-actions/gh-migrate-releases/internal/mapping"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

func SyncReleases() {
	// Get all releases from source repository
	fetchReleasesSpinner, _ := pterm.DefaultSpinner.Start("Fetching releases from repository: ", viper.GetString("REPOSITORY"))
	releases, err := api.GetSourceRepositoryReleases()
	if err != nil {
		pterm.Error.Printf("Error getting releases: %v", err)
		fetchReleasesSpinner.Fail()
	}
	fetchReleasesSpinner.UpdateText("Releases fetched successfully!")
	fetchReleasesSpinner.Success()

	// Create releases in target repository
	createReleasesSpinner, _ := pterm.DefaultSpinner.Start("Creating releases in target repository...", viper.GetString("REPOSITORY"))

	//loop through each release and create it in the target repository
	for _, release := range releases {
		createReleasesSpinner.UpdateText("Creating release: " + release.GetName())

		// Modify release body to map new handles and map old urls to new urls
		release, err := mapping.AddSourceTimeStamps(release)
		if err != nil {
			pterm.Warning.Printf("Error adding source timestamps: %v", err)
		}
		release.Body, err = mapping.ModifyReleaseBody(release.Body, viper.GetString("MAPPING_FILE"))
		if err != nil {
			pterm.Warning.Printf("Error modifying release body: %v", err)
		}
		// Create release api call
		newRelease, err := api.CreateRelease(release)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				pterm.Info.Printf("Release already exists: %v... skipping", release.GetName())
				continue
			} else {
				createReleasesSpinner.Fail()
				pterm.Fatal.Printf("Error creating release: %v", err)
			}
		}
		// Download assets from source repository and upload to target repository
		for _, asset := range release.Assets {

			err := api.DownloadReleaseAssets(asset)
			createReleasesSpinner.UpdateText("Downloading asset..." + asset.GetName())
			if err != nil {
				pterm.Error.Printf("Error downloading assets: %v", err)
			}
			createReleasesSpinner.UpdateText("Uploading assets..." + asset.GetName())

			err = api.UploadAssetViaURL(newRelease.GetUploadURL(), asset)
			if err != nil {
				pterm.Error.Printf("Error uploading assets: %v", err)
				createReleasesSpinner.Fail()
			}
		}

	}
	createReleasesSpinner.UpdateText("All Releases created successfully!")
	createReleasesSpinner.Success()
}
