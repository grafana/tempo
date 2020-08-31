package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Songmu/retry"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// ErrReleaseNotFound contains the error for when a release is not found
var (
	ErrReleaseNotFound = errors.New("release is not found")
)

// GitHub contains the functions necessary for interacting with GitHub release
// objects
type GitHub interface {
	CreateRelease(ctx context.Context, req *github.RepositoryRelease) (*github.RepositoryRelease, error)
	GetRelease(ctx context.Context, tag string) (*github.RepositoryRelease, error)
	EditRelease(ctx context.Context, releaseID int64, req *github.RepositoryRelease) (*github.RepositoryRelease, error)
	DeleteRelease(ctx context.Context, releaseID int64) error
	DeleteTag(ctx context.Context, tag string) error

	UploadAsset(ctx context.Context, releaseID int64, filename string) (*github.ReleaseAsset, error)
	DeleteAsset(ctx context.Context, assetID int64) error
	ListAssets(ctx context.Context, releaseID int64) ([]*github.ReleaseAsset, error)

	SetUploadURL(urlStr string) error
}

// GitHubClient is the client for interacting with the GitHub API
type GitHubClient struct {
	Owner, Repo string
	*github.Client
}

// NewGitHubClient creates and initializes a new GitHubClient
func NewGitHubClient(owner, repo, token, urlStr string) (GitHub, error) {
	if len(owner) == 0 {
		return nil, errors.New("missing GitHub repository owner")
	}

	if len(repo) == 0 {
		return nil, errors.New("missing GitHub repository name")
	}

	if len(token) == 0 {
		return nil, errors.New("missing GitHub API token")
	}

	if len(urlStr) == 0 {
		return nil, errors.New("missing GitHub API URL")
	}

	baseURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Github API URL")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(context.TODO(), ts)

	client := github.NewClient(tc)
	client.BaseURL = baseURL

	return &GitHubClient{
		Owner:  owner,
		Repo:   repo,
		Client: client,
	}, nil
}

// SetUploadURL constructs the upload URL for a release
func (c *GitHubClient) SetUploadURL(urlStr string) error {
	i := strings.Index(urlStr, "repos/")
	parsedURL, err := url.ParseRequestURI(urlStr[:i])
	if err != nil {
		return errors.Wrap(err, "failed to parse upload URL")
	}

	c.UploadURL = parsedURL
	return nil
}

// CreateRelease creates a new release object in the GitHub API
func (c *GitHubClient) CreateRelease(ctx context.Context, req *github.RepositoryRelease) (*github.RepositoryRelease, error) {

	release, res, err := c.Repositories.CreateRelease(context.TODO(), c.Owner, c.Repo, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a release")
	}

	if res.StatusCode != http.StatusCreated {
		return nil, errors.Errorf("create release: invalid status: %s", res.Status)
	}

	return release, nil
}

// GetRelease queries the GitHub API for a specified release object
func (c *GitHubClient) GetRelease(ctx context.Context, tag string) (*github.RepositoryRelease, error) {
	// Check Release whether already exists or not
	release, res, err := c.Repositories.GetReleaseByTag(context.TODO(), c.Owner, c.Repo, tag)
	if err != nil {
		if res == nil {
			return nil, errors.Wrapf(err, "failed to get release tag: %s", tag)
		}

		// TODO(tcnksm): Handle invalid token
		if res.StatusCode != http.StatusNotFound {
			return nil, errors.Wrapf(err,
				"get release tag: invalid status: %s", res.Status)
		}

		return nil, ErrReleaseNotFound
	}

	return release, nil
}

// EditRelease edit a release object within the GitHub API
func (c *GitHubClient) EditRelease(ctx context.Context, releaseID int64, req *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	var release *github.RepositoryRelease

	err := retry.Retry(3, 3*time.Second, func() error {
		var (
			res *github.Response
			err error
		)
		release, res, err = c.Repositories.EditRelease(context.TODO(), c.Owner, c.Repo, releaseID, req)
		if err != nil {
			return errors.Wrapf(err, "failed to edit release: %d", releaseID)
		}

		if res.StatusCode != http.StatusOK {
			return errors.Errorf("edit release: invalid status: %s", res.Status)
		}
		return nil
	})
	return release, err
}

// DeleteRelease deletes a release object within the GitHub API
func (c *GitHubClient) DeleteRelease(ctx context.Context, releaseID int64) error {
	res, err := c.Repositories.DeleteRelease(context.TODO(), c.Owner, c.Repo, releaseID)
	if err != nil {
		return errors.Wrap(err, "failed to delete release")
	}

	if res.StatusCode != http.StatusNoContent {
		return errors.Errorf("delete release: invalid status: %s", res.Status)
	}

	return nil
}

// DeleteTag deletes a tag from the GitHub API
func (c *GitHubClient) DeleteTag(ctx context.Context, tag string) error {
	ref := fmt.Sprintf("tags/%s", tag)
	res, err := c.Git.DeleteRef(context.TODO(), c.Owner, c.Repo, ref)
	if err != nil {
		return errors.Wrapf(err, "failed to delete tag: %s", ref)
	}

	if res.StatusCode != http.StatusNoContent {
		return errors.Errorf("delete tag: invalid status: %s", res.Status)
	}

	return nil
}

// UploadAsset uploads specified assets to a given release object
func (c *GitHubClient) UploadAsset(ctx context.Context, releaseID int64, filename string) (*github.ReleaseAsset, error) {

	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get abs path")
	}

	opts := &github.UploadOptions{
		// Use base name by default
		Name: filepath.Base(filename),
	}

	var asset *github.ReleaseAsset
	err = retry.Retry(3, 3*time.Second, func() error {
		var (
			res *github.Response
			err error
		)

		f, err := os.Open(filename)
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		defer f.Close()

		asset, res, err = c.Repositories.UploadReleaseAsset(context.TODO(), c.Owner, c.Repo, releaseID, opts, f)
		if err != nil {
			return errors.Wrapf(err, "failed to upload release asset: %s", filename)
		}

		switch res.StatusCode {
		case http.StatusCreated:
			return nil
		case 422:
			return errors.Errorf(
				"upload release asset: invalid status code: %s",
				"422 (this is probably because the asset already uploaded)")
		default:
			return errors.Errorf(
				"upload release asset: invalid status code: %s", res.Status)
		}
	})
	return asset, err
}

// DeleteAsset deletes assets from a given release object
func (c *GitHubClient) DeleteAsset(ctx context.Context, assetID int64) error {
	res, err := c.Repositories.DeleteReleaseAsset(context.TODO(), c.Owner, c.Repo, assetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete release asset")
	}

	if res.StatusCode != http.StatusNoContent {
		return errors.Errorf("delete release assets: invalid status code: %s", res.Status)
	}

	return nil
}

// ListAssets lists assets associated with a given release
func (c *GitHubClient) ListAssets(ctx context.Context, releaseID int64) ([]*github.ReleaseAsset, error) {
	result := []*github.ReleaseAsset{}
	page := 1

	for {
		assets, res, err := c.Repositories.ListReleaseAssets(context.TODO(), c.Owner, c.Repo, releaseID, &github.ListOptions{Page: page})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list assets")
		}

		if res.StatusCode != http.StatusOK {
			return nil, errors.Errorf("list release assets: invalid status code: %s", res.Status)
		}

		result = append(result, assets...)

		if res.NextPage <= page {
			break
		}

		page = res.NextPage
	}

	return result, nil
}
