package mapping

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/spf13/viper"
)

func loadHandleMap(filePath string) (map[string]string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	handleMap := make(map[string]string)
	for _, record := range records {
		handleMap[record[0]] = record[1]
	}

	return handleMap, nil
}

func ModifyReleaseBody(releaseBody *string, filePath string) (*string, error) {
	// Modify release body to map new handles and map old urls to new urls

	updatedReleaseBody := *releaseBody

	if viper.GetString("SOURCE_HOSTNAME") != "" {
		// Replace source hostname with GHEC hostname github.com
		updatedReleaseBody = strings.ReplaceAll(updatedReleaseBody, viper.GetString("SOURCE_HOSTNAME"), "github.com")
	}

	// Replace source organization with target organization
	updatedReleaseBody = strings.ReplaceAll(updatedReleaseBody, viper.GetString("SOURCE_ORGANIZATION"), viper.GetString("TARGET_ORGANIZATION"))

	// Load handle map from file
	handleMap, err := loadHandleMap(filePath)
	if err != nil {
		return releaseBody, err //return the original release body if an error occurs
	}

	// Replace old handles with new handles
	for source, target := range handleMap {
		updatedReleaseBody = strings.ReplaceAll(updatedReleaseBody, source, target)
	}

	return &updatedReleaseBody, nil
}

func AddSourceTimeStamps(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}

	releaseBody := ""
	if release.Body != nil {
		releaseBody = *release.Body
	}

	var createdAt, publishedAt string
	now := time.Now().Format("January 2, 2006 at 15:04:05 CST")

	if release.CreatedAt != nil {
		createdAt = release.CreatedAt.Format("January 2, 2006 at 15:04:05 CST")
	} else {
		createdAt = now
	}

	if release.PublishedAt != nil {
		publishedAt = release.PublishedAt.Format("January 2, 2006 at 15:04:05 CST")
	} else {
		publishedAt = now
	}

	// Add source timestamps to release body
	releaseBody = releaseBody + "\n\n" + ">Release Originally Created on: " + createdAt + "\n" + "> Release Originally Published on: " + publishedAt

	release.Body = &releaseBody

	return release, nil
}
