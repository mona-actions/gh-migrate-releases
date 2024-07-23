package export

import (
	"fmt"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/mona-actions/gh-migrate-releases/internal/files"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

func CreateJSONs() {
	// Get all teams from source organization
	fetchReleasesSpinner, _ := pterm.DefaultSpinner.Start("Fetching releases from repository...")
	repository := viper.GetString("REPOSITORY")
	owner := viper.GetString("SOURCE_ORGANIZATION")
	releases, err := api.GetSourceRepositoryReleases(owner, repository)
	if err != nil {
		pterm.Fatal.Printf("Error getting releases: %v", err)
	}
	for index, release := range releases {
		filename := "release-" + fmt.Sprint(index) + ".json"
		err := files.CreateJSON(release, filename)
		if err != nil {
			pterm.Fatal.Printf("Error creating JSON: %v", err)
			fetchReleasesSpinner.Fail()
		}
		fetchReleasesSpinner.Success()
	}
}
