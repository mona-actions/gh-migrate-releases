package api

import (
	"context"
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

func GetSourceRepositoryReleases() ([]*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("source_token"), viper.GetString("source_hostname"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	var allReleases []*github.RepositoryRelease
	opts := &github.ListOptions{PerPage: 100}

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, viper.Get("SOURCE_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), opts)
		if err != nil {
			return allReleases, fmt.Errorf("error getting releases: %v", err)
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReleases, nil

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

func CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"), "")

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	newRelease, _, err := client.Repositories.CreateRelease(ctx, viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), release)
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

func WriteToIssue(issueNumber int, comment string) error {
	//client := newGHRestClient(viper.GetString("TARGET_TOKEN"), "")

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	fmt.Printf("context: %v", ctx)
	fmt.Printf("comment: %v", comment)
	//_, _, err := client.Issues.CreateComment(ctx, viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), issueNumber, &github.IssueComment{Body: &comment})
	// if err != nil {
	// 	return err
	// }

	return nil
}
