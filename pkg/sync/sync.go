package sync

import (
	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/mona-actions/gh-migrate-releases/internal/mapping"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

func SyncReleases() {
	// Get all releases from source repository
	fetchReleasesSpinner, _ := pterm.DefaultSpinner.Start("Fetching releases from repository...")
	releases, err := api.GetSourceRepositoryReleases()
	if err != nil {
		pterm.Error.Printf("Error getting releases: %v", err)
		fetchReleasesSpinner.Fail()
	}
	fetchReleasesSpinner.Success()

	// Create releases in target repository
	createReleasesSpinner, _ := pterm.DefaultSpinner.Start("Creating releases in target repository...")

	//loop through each release and create it in the target repository
	for _, release := range releases {
		createReleasesSpinner.UpdateText("Creating release: " + release.GetName())

		// Modify release body to map new handles and map old urls to new urls
		release.Body, err = mapping.ModifyReleaseBody(release.Body, viper.GetString("MAPPING_FILE"))
		if err != nil {
			pterm.Error.Printf("Error modifying release body: %v", err)
		}

		// Create release api call
		newRelease, err := api.CreateRelease(release)
		if err != nil {
			createReleasesSpinner.Fail()
			pterm.Fatal.Printf("Error creating release: %v", err)
		}
		createReleasesSpinner.UpdateText("Downloading assets...")
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
