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

	"log"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v62/github"
	"github.com/mona-actions/gh-migrate-releases/internal/file"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Releases []Release

type Release struct {
	*github.RepositoryRelease
}

var tmpDir = "tmp"

func newGHGraphqlClient(token string) *githubv4.Client {
	hostname := viper.GetString("SOURCE_HOSTNAME")
	var client *githubv4.Client
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(httpClient.Transport)

	if err != nil {
		panic(err)
	}
	client = githubv4.NewClient(rateLimiter)

	// Trim any trailing slashes from the hostname
	hostname = strings.TrimSuffix(hostname, "/")

	// If hostname is received, create a new client with the hostname
	if hostname != "" {
		client = githubv4.NewEnterpriseClient(hostname+"/api/graphql", rateLimiter)
	}
	return client
}

func newGHRestClient(token string) *github.Client {
	hostname := viper.GetString("SOURCE_HOSTNAME")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(tc.Transport)

	if err != nil {
		panic(err)
	}

	client := github.NewClient(rateLimiter)

	hostname = strings.TrimSuffix(hostname, "/")

	if hostname != "" {
		client, err = github.NewClient(rateLimiter).WithEnterpriseURLs("https://"+hostname+"/api/v3", "https://"+hostname+"/api/uploads")
		if err != nil {
			panic(err)
		}
	}

	return client
}

func GetSourceRepositoryReleases() ([]*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("source_token"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	releases, _, err := client.Repositories.ListReleases(ctx, viper.Get("SOURCE_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), &github.ListOptions{})
	if err != nil {
		fmt.Println("Error getting releases: ", err)
		return nil, err
	}

	return releases, nil
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
		log.Printf("HTTP request failed: %s", resp.Status)
		return fmt.Errorf("HTTP request failed with status code %d, Message: %s", resp.StatusCode, resp.Body)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	newRelease, _, err := client.Repositories.CreateRelease(ctx, viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), release)
	if err != nil {
		//fmt.Println("Error creating release: ", err)
		return nil, err
	}

	return newRelease, nil
}

func UploadReleaseAsset(releaseID int64, asset *github.ReleaseAsset) error {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"))

	log.Println("Received: ", asset.GetName(), asset.GetLabel(), *asset.ContentType)
	opts := &github.UploadOptions{
		Name:      asset.GetName(),
		Label:     asset.GetLabel(),
		MediaType: *asset.ContentType,
	}

	log.Println("Options: ", opts)

	dirName := tmpDir
	fileName := dirName + "/" + asset.GetName()
	log.Println("filename: ", fileName)

	file, err := file.OpenFile(fileName)
	if err != nil {
		log.Println("Error opening file: ", fileName, err)
		return fmt.Errorf("error opening file: %v err: %v", file, err)
	}

	defer file.Close()

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	log.Println("Targeting: ", viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), releaseID)
	newAsset, resp, err := client.Repositories.UploadReleaseAsset(ctx, viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), releaseID, opts, file)
	if err != nil {
		//fmt.Println("Error uploading asset to release: ", releaseID, "err:", err)
		return fmt.Errorf("error uploading asset to release: %v err: %v", releaseID, err)
	}
	log.Printf("response: %v, new asset: %v, err: %v", resp, newAsset, err)

	err = os.Remove(asset.GetName())
	if err != nil {
		return fmt.Errorf("error deleting asset from local storage: %v err: %v", asset.Name, err)
	}

	return nil
}

func UploadAssetViaURL(uploadURL string, asset *github.ReleaseAsset) error {
	log.Println("Received: ", asset.GetName(), asset.GetLabel(), *asset.ContentType)
	opts := &github.UploadOptions{
		Name:      asset.GetName(),
		Label:     asset.GetLabel(),
		MediaType: *asset.ContentType,
	}

	//log.Println("Options: ", opts)

	dirName := tmpDir
	fileName := dirName + "/" + asset.GetName()
	log.Println("filename: ", fileName)

	file, err := file.OpenFile(fileName)
	if err != nil {
		log.Println("Error opening file: ", fileName, err)
		return fmt.Errorf("error opening file: %v err: %v", file, err)
	}

	stat, err := file.Stat()
	if err != nil {
		log.Println("Error getting file stats: ", fileName, err)
	}

	mediaType := mime.TypeByExtension(filepath.Ext(file.Name()))
	if opts.MediaType != "" {
		mediaType = asset.GetContentType()
	}

	log.Println("Targeting: ", viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), uploadURL)

	uploadURL = strings.TrimSuffix(uploadURL, "{?name,label}")

	params := url.Values{}
	params.Add("name", asset.GetName())
	params.Add("label", asset.GetLabel())

	uploadURLWithParams := fmt.Sprintf("%s?%s", uploadURL, params.Encode())

	log.Println("uploadURLWithParams: ", uploadURLWithParams)

	req, err := http.NewRequest("POST", uploadURLWithParams, file)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

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

	if resp.StatusCode != http.StatusOK {
		//fmt.Println("Error uploading asset to release: ", releaseID, "err:", err)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body: ", err)
		}
		bodyString := string(bodyBytes)
		log.Println("Response: ", bodyString)
		return fmt.Errorf("error uploading asset to release: %v err: %v", uploadURL, resp.Body)
	}
	log.Printf("response: %v, err: %v", resp, err)

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("error deleting asset from local storage: %v err: %v", asset.Name, err)
	}

	return nil
}

func NewUploadRequest(urlStr string, reader io.Reader, size int64, mediaType string) (*http.Request, error) {

	req, err := http.NewRequest("POST", urlStr, reader)
	if err != nil {
		return nil, err
	}

	req.ContentLength = size

	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	return req, nil
}

// func UploadAssetViaURL(uploadURL string, asset *github.ReleaseAsset) error {
// 	log.Println("Received: ", asset.GetName(), asset.GetLabel(), *asset.ContentType)
// 	opts := &github.UploadOptions{
// 		Name:      asset.GetName(),
// 		Label:     asset.GetLabel(),
// 		MediaType: *asset.ContentType,
// 	}

// 	log.Println("Options: ", opts)

// 	dirName := tmpDir
// 	fileName := dirName + "/" + asset.GetName()
// 	log.Println("filename: ", fileName)

// 	fileData, err := os.ReadFile(fileName)
// 	if err != nil {
// 		log.Println("Error opening file: ", fileName, err)
// 		return fmt.Errorf("error opening file: %v err: %v", fileName, err)
// 	}

// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)

// 	filePart, err := writer.CreateFormFile("file", filepath.Base(fileName))
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	_, err = filePart.Write(fileData)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	err = writer.Close()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Println("Targeting: ", viper.Get("TARGET_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), uploadURL)

// 	req, err := http.NewRequest("POST", uploadURL, body)
// 	if err != nil {
// 		return fmt.Errorf("error creating request: %v", err)
// 	}

// 	req.Header.Set("Accept", "application/vnd.github+json")
// 	req.Header.Set("Authorization", "Bearer "+viper.Get("TARGET_TOKEN").(string))
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("error uploading asset to release: %v err: %v", uploadURL, err)
// 	}

// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		//fmt.Println("Error uploading asset to release: ", releaseID, "err:", err)
// 		bodyBytes, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			log.Println("Error reading response body: ", err)
// 		}
// 		bodyString := string(bodyBytes)
// 		log.Println("Response: ", bodyString)
// 		return fmt.Errorf("error uploading asset to release: %v err: %v", uploadURL, resp.Body)
// 	}
// 	log.Printf("response: %v, err: %v", resp, err)

// 	err = os.Remove(fileName)
// 	if err != nil {
// 		return fmt.Errorf("error deleting asset from local storage: %v err: %v", asset.Name, err)
// 	}

// 	return nil
// }
