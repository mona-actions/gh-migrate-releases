package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v62/github"
	"github.com/mona-actions/gh-migrate-releases/internal/files"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Releases []Release

type Release struct {
	*github.RepositoryRelease
}

var tmpDir = "tmp"

func newGHRestClient(token string, hostname string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(tc.Transport)

	if err != nil {
		panic(err)
	}

	client := github.NewClient(rateLimiter)

	if hostname != "" {
		hostname = strings.TrimSuffix(hostname, "/")
		client, err = github.NewClient(rateLimiter).WithEnterpriseURLs("https://"+hostname+"/api/v3", "https://"+hostname+"/api/uploads")
		if err != nil {
			panic(err)
		}
	}

	return client
}

func GetSourceRepositoryReleases(owner string, repository string) ([]*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("source_token"), viper.GetString("source_hostname"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	var allReleases []*github.RepositoryRelease
	opts := &github.ListOptions{PerPage: 100}

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repository, opts)
		if err != nil {
			return allReleases, fmt.Errorf("unable to get releases: %v", err)
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReleases, nil

}

func GetSourceRepositoryLatestRelease(owner string, repository string) (*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("source_token"), viper.GetString("source_hostname"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	release, resp, err := client.Repositories.GetLatestRelease(ctx, owner, repository)

	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no releases found for repository %s/%s", owner, repository)
		}
		return nil, fmt.Errorf("unable to get latest release: %v", err)
	}

	return release, nil
}

// AssetExists checks if an asset with the same name and size already exists in a release
func AssetExists(release *github.RepositoryRelease, assetName string, assetSize int64) bool {
	if release == nil || release.Assets == nil {
		return false
	}

	for _, existingAsset := range release.Assets {
		if existingAsset.GetName() == assetName && int64(existingAsset.GetSize()) == assetSize {
			return true
		}
	}

	return false
}

// GetReleaseByTag retrieves a release from the target repository by its tag name
func GetReleaseByTag(owner string, repository string, tagName string) (*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"), viper.GetString("TARGET_HOSTNAME"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	release, resp, err := client.Repositories.GetReleaseByTag(ctx, owner, repository, tagName)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("release not found for tag %s", tagName)
		}
		return nil, fmt.Errorf("unable to get release by tag: %v", err)
	}

	return release, nil
}

// ReleaseExists checks if a release with matching tag_name, name, and target_commitish already exists
func ReleaseExists(owner string, repository string, release *github.RepositoryRelease) (*github.RepositoryRelease, bool) {
	if release == nil || release.TagName == nil {
		return nil, false
	}

	existingRelease, err := GetReleaseByTag(owner, repository, release.GetTagName())
	if err != nil {
		return nil, false
	}

	// Check if name and target_commitish match
	nameMatches := existingRelease.GetName() == release.GetName()
	commitMatches := existingRelease.GetTargetCommitish() == release.GetTargetCommitish()

	if nameMatches && commitMatches {
		return existingRelease, true
	}

	return existingRelease, false
}

func DownloadReleaseAssets(asset *github.ReleaseAsset) error {

	token := viper.Get("SOURCE_TOKEN").(string)

	// Download the asset

	url := asset.GetBrowserDownloadURL()
	dirName := tmpDir
	fileName := dirName + "/" + asset.GetName()

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return err
	}

	err = DownloadFileFromURL(url, fileName, token)
	if err != nil {
		return err
	}
	return nil
}

func DownloadReleaseZip(release *github.RepositoryRelease) error {
	token := viper.Get("SOURCE_TOKEN").(string)
	repo := viper.Get("REPOSITORY").(string)
	if release.TagName == nil {
		return errors.New("TagName is nil")
	}
	tag := *release.TagName
	var tagName string

	url := *release.ZipballURL

	if len(tag) > 1 && tag[0] == 'v' && unicode.IsDigit(rune(tag[1])) {
		tagName = strings.TrimPrefix(tag, "v")
	} else {
		tagName = tag
	}

	fileName := fmt.Sprintf("%s-%s.zip", repo, tagName)

	err := DownloadFileFromURL(url, fileName, token)
	if err != nil {
		return err
	}

	return nil
}

func DownloadReleaseTarball(release *github.RepositoryRelease) error {
	token := viper.Get("SOURCE_TOKEN").(string)
	repo := viper.Get("REPOSITORY").(string)
	if release.TagName == nil {
		return errors.New("TagName is nil")
	}
	tag := *release.TagName
	var tagName string

	url := *release.TarballURL

	if len(tag) > 1 && tag[0] == 'v' && unicode.IsDigit(rune(tag[1])) {
		tagName = strings.TrimPrefix(tag, "v")
	} else {
		tagName = tag
	}

	fileName := fmt.Sprintf("%s-%s.tar.gz", repo, tagName)

	err := DownloadFileFromURL(url, fileName, token)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFileFromURL(url, fileName, token string) error {
	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(fmt.Errorf("error creating request: %s", err))
	}

	req.Header.Add("Authorization", "Bearer "+token)

	// Get the data
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error getting file: %v  err:%v", fileName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status code %d, Message: %s", resp.StatusCode, resp.Body)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func CreateRelease(repository string, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"), viper.GetString("TARGET_HOSTNAME"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	newRelease, _, err := client.Repositories.CreateRelease(ctx, viper.Get("TARGET_ORGANIZATION").(string), repository, release)
	if err != nil {
		if strings.Contains(err.Error(), "already_exists") {
			return nil, fmt.Errorf("release already exists: %v", release.GetName())
		} else {
			return nil, err
		}
	}

	return newRelease, nil
}

func UploadAssetViaURL(uploadURL string, asset *github.ReleaseAsset) error {

	dirName := tmpDir
	fileName := dirName + "/" + asset.GetName()

	// Open the file
	file, err := files.OpenFile(fileName)
	if err != nil {
		return fmt.Errorf("error opening file: %v err: %v", file, err)
	}

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file size of %v err: %v ", fileName, err)
	}

	// Get the media type
	mediaType := mime.TypeByExtension(filepath.Ext(file.Name()))
	if *asset.ContentType != "" {
		mediaType = asset.GetContentType()
	}

	uploadURL = strings.TrimSuffix(uploadURL, "{?name,label}")

	// Add the name and label to the URL
	params := url.Values{}
	params.Add("name", asset.GetName())
	params.Add("label", asset.GetLabel())

	uploadURLWithParams := fmt.Sprintf("%s?%s", uploadURL, params.Encode())

	// Create the request
	req, err := http.NewRequest("POST", uploadURLWithParams, file)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set the headers
	req.ContentLength = stat.Size()
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+viper.Get("TARGET_TOKEN").(string))
	req.Header.Set("Content-Type", mediaType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading asset to release: %v err: %v", uploadURL, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error uploading asset to release: %v err: %v", uploadURL, resp.Body)
	}

	err = files.RemoveFile(fileName)
	if err != nil {
		return fmt.Errorf("error deleting asset from local storage: %v err: %v", asset.Name, err)
	}

	return nil
}

func WriteToIssue(owner string, repository string, issueNumber int, comment string) error {

	client := newGHRestClient(viper.GetString("TARGET_TOKEN"), viper.GetString("TARGET_HOSTNAME"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	_, _, err := client.Issues.CreateComment(ctx, owner, repository, issueNumber, &github.IssueComment{Body: &comment})
	if err != nil {
		return err
	}

	return nil
}

func GetDatafromGitHubContext() (string, string, int, error) {
	githubContext := os.Getenv("GITHUB_CONTEXT")
	if githubContext == "" {
		return "", "", 0, fmt.Errorf("GITHUB_CONTEXT is not set or empty")
	}

	var issueEvent github.IssueEvent

	err := json.Unmarshal([]byte(githubContext), &issueEvent)
	if err != nil {
		return "", "", 0, fmt.Errorf("error unmarshalling GITHUB_CONTEXT: %v", err)
	}
	organization := *issueEvent.Repository.Owner.Login
	repository := *issueEvent.Repository.Name
	issueNumber := *issueEvent.Issue.Number

	return organization, repository, issueNumber, nil
}

func SetLatestRelease(owner string, repository string, releaseID int64) error {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"), viper.GetString("TARGET_HOSTNAME"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	_, _, err := client.Repositories.EditRelease(ctx, owner, repository, releaseID, &github.RepositoryRelease{
		MakeLatest: github.String("true"),
	})
	if err != nil {
		return fmt.Errorf("error making release latest: %v", err)
	}

	return nil
}
