package export

import (
	"fmt"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/mona-actions/gh-migrate-releases/internal/files"
	"github.com/pterm/pterm"
)

func CreateJSONs() {
	// Get all teams from source organization
	fetchReleasesSpinner, _ := pterm.DefaultSpinner.Start("Fetching teams from organization...")
	releases, err := api.GetSourceRepositoryReleases()
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
