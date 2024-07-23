package sync

import (
	"fmt"
	"os"
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
	fetchReleasesSpinner.UpdateText(fmt.Sprintf(" %d Releases fetched successfully!", len(releases)))
	fetchReleasesSpinner.Success()

	// Create releases in target repository
	createReleasesSpinner, _ := pterm.DefaultSpinner.Start("Creating releases in target repository...", viper.GetString("REPOSITORY"))
	var failed int
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
				failed++
				createReleasesSpinner.Fail()
				pterm.Warning.Printf("Error creating release: %v", err)
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

	if os.Getenv("CI") == "true" && os.Getenv("GITHUB_ACTIONS") == "true" {
		// Print in a README Table format the number of releases created
		message := fmt.Sprintf(
			"```\n"+
				"| No. of Releases | Succeeded | Failed |\n"+
				"| --------------- | --------- | ------ |\n"+
				"| %d | %d | %d |\n"+
				"```",
			len(releases), len(releases)-failed, failed,
		)
		organization, repository, issueNumber, err := api.GetDatafromGitHubContext()
		if err != nil {
			pterm.Error.Printf("Error getting issue number: %v", err)
		}
		err = api.WriteToIssue(organization, repository, issueNumber, message)
		if err != nil {
			pterm.Error.Printf("Error writing releases table to issue: %v", err)
		}
	}

	createReleasesSpinner.UpdateText("All Releases created successfully!")
	createReleasesSpinner.Success()
}
