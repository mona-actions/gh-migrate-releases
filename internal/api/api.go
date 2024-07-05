package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode"

	"log"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v62/github"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Releases []Release

type Release struct {
	*github.RepositoryRelease
}

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

func GetReleasesAssets(release *github.RepositoryRelease) error {
	client := newGHRestClient(viper.GetString("source_token"))

	log.Println("Getting assets for release: ", *release.Name, *release.TagName)
	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, viper.Get("SOURCE_ORGANIZATION").(string), viper.Get("REPOSITORY").(string), release.GetID(), &github.ListOptions{})
	log.Println("assets", assets)
	if err != nil {
		fmt.Println("Error getting release assets: ", err)
	}

	var assetsData = []map[string]string{}
	for _, asset := range assets {
		assetsData = append(assetsData, map[string]string{"Name": asset.GetName(), "URL": asset.GetBrowserDownloadURL()})
		log.Println("Asset: ", asset.GetName(), asset.GetBrowserDownloadURL(), assetsData)
	}

	return nil
}

func GetReleaseZip(release *github.RepositoryRelease) error {
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
	}

	fileName := fmt.Sprintf("%s-%s.zip", repo, tagName)

	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %s", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	// Get the data
	resp, err := http.DefaultClient.Do(req)
	log.Println("response", resp.StatusCode)
	if err != nil {
		log.Println("Error getting release zip: ", err)
		return err
	}
	defer resp.Body.Close()
	//log.Println("response", resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP request failed: %s", resp.Status)
		return fmt.Errorf("HTTP request failed with status code %d, Message: %s", resp.StatusCode, resp.Body)
	}

	if resp.Header.Get("Content-Type") != "application/zip" {
		return fmt.Errorf("expected Content-Type to be application/zip, got %s", resp.Header.Get("Content-Type"))
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func GetReleaseTarball(release *github.RepositoryRelease) error {
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
	}

	fileName := fmt.Sprintf("%s-%s.tar.gz", repo, tagName)

	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	// Get the data
	resp, err := http.DefaultClient.Do(req)
	log.Println("tarball response", resp)
	if err != nil {
		log.Println("Error getting release tarball: ", err)
		return err
	}
	defer resp.Body.Close()
	log.Println("response", resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP request failed: %s", resp.Status)
		return fmt.Errorf("HTTP request failed with status code %d, Message: %s", resp.StatusCode, resp.Body)
	}

	if resp.Header.Get("Content-Type") != "application/gzip" {
		log.Println("Content-Type", resp.Header.Get("Content-Type"))
		return fmt.Errorf("expected Content-Type to be application/gzip, got %s", resp.Header.Get("Content-Type"))
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func GetSourceOrganizationTeams() []map[string]string {
	client := newGHGraphqlClient(viper.GetString("SOURCE_TOKEN"))

	var query struct {
		Organization struct {
			Teams struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Edges []struct {
					Node struct {
						Id          string
						Name        string
						Description string
						Slug        string
						Privacy     string
						ParentTeam  struct {
							Id   string
							Slug string
						}
					}
				}
			} `graphql:"teams(first: $first, after: $after)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(viper.Get("SOURCE_ORGANIZATION").(string)),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var teams = []map[string]string{}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			panic(err)
		}

		for _, team := range query.Organization.Teams.Edges {
			teams = append(teams, map[string]string{
				"Id":             team.Node.Id,
				"Name":           team.Node.Name,
				"Slug":           team.Node.Slug,
				"Description":    team.Node.Description,
				"Privacy":        team.Node.Privacy,
				"ParentTeamId":   team.Node.ParentTeam.Id,
				"ParentTeamName": team.Node.ParentTeam.Slug,
			})
		}

		if !query.Organization.Teams.PageInfo.HasNextPage {
			break
		}

		variables["after"] = githubv4.NewString(query.Organization.Teams.PageInfo.EndCursor)
	}

	return teams
}

func GetTeamMemberships(team string) []map[string]string {
	client := newGHGraphqlClient(viper.GetString("source_token"))

	var query struct {
		Organization struct {
			Team struct {
				Members struct {
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
					Edges []struct {
						Node struct {
							Login string
							Email string
						}
						Role string
					}
				} `graphql:"members(first: $first, after: $after)"`
			} `graphql:"team(slug: $slug)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(viper.Get("SOURCE_ORGANIZATION").(string)),
		"slug":  githubv4.String(team),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var members = []map[string]string{}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			panic(err)
		}

		for _, member := range query.Organization.Team.Members.Edges {
			members = append(members, map[string]string{"Login": member.Node.Login, "Email": member.Node.Email, "Role": member.Role})
		}

		if !query.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}

		variables["after"] = githubv4.NewString(query.Organization.Team.Members.PageInfo.EndCursor)
	}

	return members
}

func GetTeamRepositories(team string) []map[string]string {
	client := newGHGraphqlClient(viper.GetString("source_token"))

	var query struct {
		Organization struct {
			Team struct {
				Repositories struct {
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
					Edges []struct {
						Permission string
						Node       struct {
							Name string
						}
					}
				} `graphql:"repositories(first: $first, after: $after)"`
			} `graphql:"team(slug: $slug)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(viper.Get("SOURCE_ORGANIZATION").(string)),
		"slug":  githubv4.String(team),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var repositories = []map[string]string{}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			panic(err)
		}

		for _, repo := range query.Organization.Team.Repositories.Edges {
			repositories = append(repositories, map[string]string{"Name": repo.Node.Name, "Permission": repo.Permission})
		}

		if !query.Organization.Team.Repositories.PageInfo.HasNextPage {
			break
		}

		variables["after"] = githubv4.NewString(query.Organization.Team.Repositories.PageInfo.EndCursor)
	}

	return repositories
}

func GetSourceOrganizationRepositories() []map[string]string {
	client := newGHGraphqlClient(viper.GetString("source_token"))

	var query struct {
		Organization struct {
			Repositories struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Edges []struct {
					Node struct {
						Name string
					}
				}
			} `graphql:"repositories(first: $first, after: $after)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(viper.Get("SOURCE_ORGANIZATION").(string)),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var repositories = []map[string]string{}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			panic(err)
		}

		for _, repo := range query.Organization.Repositories.Edges {
			repositories = append(repositories, map[string]string{"Name": repo.Node.Name})
		}

		if !query.Organization.Repositories.PageInfo.HasNextPage {
			break
		}

		variables["after"] = githubv4.NewString(query.Organization.Repositories.PageInfo.EndCursor)
	}

	return repositories
}

func GetRepositoryCollaborators(repository string) []map[string]string {
	client := newGHGraphqlClient(viper.GetString("source_token"))

	var query struct {
		Repository struct {
			Collaborators struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Edges []struct {
					Permission string
					Node       struct {
						Login string
						Email string
					}
				}
			} `graphql:"collaborators(first: $first, after: $after)"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(viper.Get("SOURCE_ORGANIZATION").(string)),
		"name":  githubv4.String(repository),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var collaborators = []map[string]string{}
	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			panic(err)
		}

		for _, collaborator := range query.Repository.Collaborators.Edges {
			collaborators = append(collaborators, map[string]string{"Login": collaborator.Node.Login, "Email": collaborator.Node.Email, "Permission": collaborator.Permission})
		}

		if !query.Repository.Collaborators.PageInfo.HasNextPage {
			break
		}

		variables["after"] = githubv4.NewString(query.Repository.Collaborators.PageInfo.EndCursor)
	}

	return collaborators
}

func AddTeamRepository(slug string, repo string, permission string) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"))

	fmt.Println("Adding repository to team: ", slug, repo, permission)

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	_, err := client.Teams.AddTeamRepoBySlug(ctx, viper.Get("TARGET_ORGANIZATION").(string), slug, viper.Get("TARGET_ORGANIZATION").(string), repo, &github.TeamAddTeamRepoOptions{Permission: permission})

	if err != nil {
		if strings.Contains(err.Error(), "422 Validation Failed") {
			fmt.Println("Error adding repository to team: ", slug, repo, permission)
		} else if strings.Contains(err.Error(), "404 Not Found") {
			fmt.Println("Error adding repository to team, repository not found: ", slug, repo, permission)
		} else {
			fmt.Println("adding repository to team: ", slug, repo, permission, "Unknown error", err, err.Error())
		}
	}
}

func AddTeamMember(slug string, member string, role string) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"))

	role = strings.ToLower(role) // lowercase to match github api
	fmt.Println("Adding member to team: ", slug, member, role)

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	_, _, err := client.Teams.AddTeamMembershipBySlug(ctx, viper.Get("TARGET_ORGANIZATION").(string), slug, member, &github.TeamAddTeamMembershipOptions{Role: role})
	if err != nil {
		fmt.Println("Error adding member to team: ", slug, member, err)
	}
}

func GetTeamId(TeamName string) (int64, error) {
	client := newGHRestClient(viper.GetString("TARGET_TOKEN"))

	ctx := context.WithValue(context.Background(), github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
	team, _, err := client.Teams.GetTeamBySlug(ctx, viper.Get("TARGET_ORGANIZATION").(string), TeamName)
	if err != nil {
		fmt.Println("Error getting parent team ID: ", TeamName)
		return 0, err
	}
	return *team.ID, nil
}
