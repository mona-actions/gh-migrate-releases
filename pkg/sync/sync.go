package sync

import (
	"log"

	"github.com/mona-actions/gh-migrate-releases/internal/api"
	"github.com/pterm/pterm"
)

func SyncReleases() {
	// Get all teams from source organization
	//teamsSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching releases from repository...")
	releases, err := api.GetSourceRepositoryReleases()
	if err != nil {
		pterm.Error.Printf("Error getting releases: %v", err)
		//teamsSpinnerSuccess.Fail()
	}
	//teamsSpinnerSuccess.Success()

	// Create teams in target organization
	createReleasesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Creating releases in target repository...")
	for _, release := range releases {
		createReleasesSpinnerSuccess.UpdateText("Creating release: " + release.GetName())
		newRelease, err := api.CreateRelease(release)
		if err != nil {
			pterm.Error.Printf("Error creating release: %v", err)
		}
		createReleasesSpinnerSuccess.UpdateText("Downloading assets...")
		for _, asset := range release.Assets {
			createReleasesSpinnerSuccess.UpdateText("Downloading asset..." + asset.GetName())
			err := api.DownloadReleaseAssets(asset)
			if err != nil {
				pterm.Error.Printf("Error downloading assets: %v", err)
			}
			createReleasesSpinnerSuccess.UpdateText("Uploading assets..." + asset.GetName())
			log.Println("New Release ID: ", *newRelease.ID, newRelease.GetAssetsURL(), newRelease.GetUploadURL())
			err = api.UploadAssetViaURL(newRelease.GetUploadURL(), asset)
			if err != nil {
				pterm.Error.Printf("Error uploading assets: %v", err)
				createReleasesSpinnerSuccess.Fail()
			}
		}

	}
	createReleasesSpinnerSuccess.Success()
}

// func mapMembers(team team.Team) team.Team {
// 	for i, member := range team.Members {
// 		// Check if member handle is in mapping file
// 		target_handle, err := getTargetHandle(os.Getenv("GHMT_MAPPING_FILE"), member.Login)
// 		if err != nil {
// 			log.Println("Unable to read or open mapping file")
// 		}
// 		team.Members[i] = updateMemberHandle(member, member.Login, target_handle)
// 	}
// 	return team
// }

// func updateMemberHandle(member team.Member, source_handle string, target_handle string) team.Member {
// 	// Update member handles
// 	if member.Login == source_handle {
// 		member.Login = target_handle
// 	}
// 	return member
// }

// func getTargetHandle(filename string, source_handle string) (string, error) {
// 	// Open mapping file
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer file.Close()

// 	// Parse mapping file
// 	reader := csv.NewReader(file)
// 	reader.FieldsPerRecord = -1 // Allow variable number of fields per record
// 	records, err := reader.ReadAll()
// 	if err != nil {
// 		return "", err
// 	}

// 	// Find target value for source value
// 	for _, record := range records[1:] {
// 		if record[0] == source_handle {
// 			return record[1], nil
// 		}
// 	}

// 	return source_handle, nil
// }
