package export

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/pterm/pterm"
)

func CreateJSONs() {
	// Get all teams from source organization
	teamsSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching teams from organization...")
	releases, err := api.GetSourceRepositoryReleases()
	if err != nil {
		log.Fatalf("Error getting releases: %v", err)
	}
	log.Printf("Releases: %v", releases)
	for index, release := range releases {
		//err := api.DownloadReleasesAssets(release)
		// if err != nil {
		// 	log.Fatalf("Error downloading assets: %v", err)
		// }
		filename := "release-" + fmt.Sprint(index) + ".json"
		createJSON(release, filename)
		// err := api.DownloadReleaseZip(release)
		// if err != nil {
		// 	log.Fatalf("Error downloading zip: %v", err)
		// }

		// err = api.DownloadReleaseTarball(release)
		// if err != nil {
		// 	log.Fatalf("Error downloading tarball: %v", err)
		// }
	}
	teamsSpinnerSuccess.Success()

	// // Create team membership csv
	// createCSVMembershipsSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Creating team membership csv...")
	// //createCSV(teams.ExportTeamMemberships(), viper.GetString("OUTPUT_FILE")+"-team-membership.csv")
	// createCSVMembershipsSpinnerSuccess.Success()

	// // Create team repository csv
	// createCSVRepositoriesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Creating team repository csv...")
	// //createCSV(teams.ExportTeamRepositories(), viper.GetString("OUTPUT_FILE")+"-team-repositories.csv")
	// createCSVRepositoriesSpinnerSuccess.Success()

	// // Get all repositories from source organization
	// repositoriesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching repositories from organization...")
	// //repositories := repository.GetSourceOrganizationRepositories()
	// repositoriesSpinnerSuccess.Success()

	// // Create repository collaborator csv
	// createCSVCollaboratorsSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Creating repository collaborator csv...")
	// //createCSV(repositories.ExportRepositoryCollaborators(), viper.GetString("OUTPUT_FILE")+"-repository-collaborators.csv")
	// createCSVCollaboratorsSpinnerSuccess.Success()
}

func createJSON(data interface{}, filename string) {

	// Create a new file
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	// Create a new JSON encoder and write to the file
	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		log.Fatalf("Error encoding JSON to file: %v", err)
	}

}

// func createCSV(data [][]string, filename string) {
// 	// Create team membership csv
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer file.Close()

// 	// Initialize csv writer
// 	writer := csv.NewWriter(file)
// 	defer writer.Flush()

// 	// Write team memberships to csv

// 	for _, line := range data {
// 		writer.Write(line)
// 	}
// }
