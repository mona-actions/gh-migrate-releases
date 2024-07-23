package sync

import (
	"fmt"
	"os"
	"strings"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/mona-actions/gh-migrate-releases/internal/files"
	"github.com/mona-actions/gh-migrate-releases/internal/mapping"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

func SyncReleases() {
	// Get all releases from source repository
	checkVars()

	var totalReleases, totalFailed int

	if viper.GetString("REPOSITORY_LIST") != "" {
		// Read repository list from file
		repositories, err := files.ReadRepositoryListFromFile(viper.GetString("REPOSITORY_LIST"))
		if err != nil {
			pterm.Error.Printf("Error reading repository list: %v", err)
			os.Exit(1)
		}

		// Loop through each repository in the list
		for _, repository := range repositories {

			releasesCount, failedReleases, err := migrateRepositoryReleases(repository)
			if err != nil {
				pterm.Error.Printf("Error migrating repository releases: %v", err)
			}

			totalReleases += releasesCount
			totalFailed += failedReleases

		}
	} else if viper.GetString("REPOSITORY") != "" {
		// Migrate releases from a single repository
		repository := viper.GetString("REPOSITORY")

		releasesCount, failedReleases, err := migrateRepositoryReleases(repository)
		if err != nil {
			pterm.Error.Printf("Error migrating repository releases: %v", err)
		}

		totalReleases += releasesCount
		totalFailed += failedReleases

	} else {
		pterm.Error.Println("Error: No repository or repository list specified")
		os.Exit(1)
	}

	if os.Getenv("CI") == "true" && os.Getenv("GITHUB_ACTIONS") == "true" {
		// Print in a README Table format the number of releases created
		message := fmt.Sprintf(
			"| No. of Releases | Succeeded | Failed |\n"+
				"| --------------- | --------- | ------ |\n"+
				"| %d | %d | %d |\n",
			totalReleases, totalReleases-totalFailed, totalFailed,
		)
		organization, repository, issueNumber, err := api.GetDatafromGitHubContext()
		if err != nil {
			pterm.Error.Printf("Error getting issue number: %v", err)
		}
		err = api.WriteToIssue(organization, repository, issueNumber, message)
		if err != nil {
			pterm.Error.Printf("Error writing releases table to issue: %v", err)
		}
	} else {
		pterm.Info.Printf("Total Releases: %d\n", totalReleases)
		pterm.Info.Printf("Succeeded: %d\n", totalReleases-totalFailed)
		pterm.Info.Printf("Failed: %d\n", totalFailed)

	}

}

func checkVars() {
	//check that repository and repository list are not sent at the same time
	if viper.GetString("REPOSITORY") != "" && viper.GetString("REPOSITORY_LIST") != "" {
		pterm.Error.Println("Error: Cannot specify both a repository and a repository list")
		os.Exit(1)
	} else if viper.GetString("REPOSITORY") != "" && viper.GetString("SOURCE_ORGANIZATION") == "" {
		pterm.Error.Println("Error: Source organization is required when specifying a repository")
		os.Exit(1)
	}
}

func migrateRepositoryReleases(repository string) (int, int, error) {
	var owner string
	// if repository includes owner, split it
	if strings.Contains(repository, "/") {
		repositoryParts := strings.Split(repository, "/")
		owner = repositoryParts[0]
		repository = repositoryParts[1]
	} else {
		owner = viper.GetString("SOURCE_ORGANIZATION")
	}

	fetchReleasesSpinner, _ := pterm.DefaultSpinner.Start("Fetching releases from repository: ", repository)
	releases, err := api.GetSourceRepositoryReleases(owner, repository)
	if err != nil {
		pterm.Fatal.Printf("Error: %v", err)
		fetchReleasesSpinner.Fail()
	}
	fetchReleasesSpinner.UpdateText(fmt.Sprintf(" %d Releases fetched successfully!", len(releases)))
	fetchReleasesSpinner.Success()

	// Create releases in target repository
	createReleasesSpinner, _ := pterm.DefaultSpinner.Start("Creating releases in target repository...", repository)
	var failed int
	releasesCount := len(releases)
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
		newRepository := repository
		newRelease, err := api.CreateRelease(newRepository, release)
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

	if failed > 0 {
		createReleasesSpinner.UpdateText("Some Releases failed to create")
		createReleasesSpinner.Fail()
		return releasesCount, failed, fmt.Errorf("some releases failed to create")
	} else {
		createReleasesSpinner.UpdateText("All Releases created successfully!")
		createReleasesSpinner.Success()
		return releasesCount, failed, nil
	}

}
