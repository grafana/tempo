// Command ghr is a tool to create a Github Release and upload your
// artifacts in parallel.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/google/go-github/github"
	"github.com/mitchellh/colorstring"
	"github.com/tcnksm/go-gitconfig"
)

const (
	// EnvGitHubToken is an environment var containing the GitHub API token
	EnvGitHubToken = "GITHUB_TOKEN"

	// EnvGitHubAPI is an environment var containing the GitHub API base endpoint.
	// This is used mainly by GitHub Enterprise users.
	EnvGitHubAPI = "GITHUB_API"

	// EnvDebug is an environment var to handle debug mode
	EnvDebug = "GHR_DEBUG"
)

// Exit codes are set to a value that represent an exit code for a particular error.
const (
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeParseFlagsError
	ExitCodeBadArgs
	ExitCodeInvalidURL
	ExitCodeTokenNotFound
	ExitCodeOwnerNotFound
	ExitCodeRepoNotFound
	ExitCodeReleaseError
)

const (
	defaultCheckTimeout = 2 * time.Second
	defaultBaseURL      = "https://api.github.com/"
	defaultParallel     = -1
)

// Debugf prints debug output when EnvDebug is set
func Debugf(format string, args ...interface{}) {
	if env := os.Getenv(EnvDebug); len(env) != 0 {
		log.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// PrintRedf prints red error message to console.
func PrintRedf(w io.Writer, format string, args ...interface{}) {
	format = fmt.Sprintf("[red]%s[reset]", format)
	fmt.Fprint(w,
		colorstring.Color(fmt.Sprintf(format, args...)))
}

// CLI is the main command line object
type CLI struct {
	// outStream and errStream correspond to stdout and stderr, respectively,
	// to take messages from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {

	var (
		owner string
		repo  string
		token string

		commitish  string
		name       string
		body       string
		draft      bool
		prerelease bool

		parallel int

		recreate bool
		replace  bool
		soft     bool

		stat    bool
		version bool
		debug   bool
	)

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprint(cli.errStream, helpText)
	}

	flags.StringVar(&owner, "username", "", "")
	flags.StringVar(&owner, "owner", "", "")
	flags.StringVar(&owner, "u", "", "")

	flags.StringVar(&repo, "repository", "", "")
	flags.StringVar(&repo, "r", "", "")

	flags.StringVar(&token, "token", os.Getenv(EnvGitHubToken), "")
	flags.StringVar(&token, "t", os.Getenv(EnvGitHubToken), "")

	flags.StringVar(&commitish, "commitish", "", "")
	flags.StringVar(&commitish, "c", "", "")

	flags.StringVar(&name, "name", "", "")
	flags.StringVar(&name, "n", "", "")

	flags.StringVar(&body, "body", "", "")
	flags.StringVar(&body, "b", "", "")

	flags.BoolVar(&draft, "draft", false, "")
	flags.BoolVar(&prerelease, "prerelease", false, "")

	flags.IntVar(&parallel, "parallel", defaultParallel, "")
	flags.IntVar(&parallel, "p", defaultParallel, "")

	flags.BoolVar(&recreate, "delete", false, "")
	flags.BoolVar(&recreate, "recreate", false, "")

	flags.BoolVar(&replace, "replace", false, "")

	flags.BoolVar(&soft, "soft", false, "")

	flags.BoolVar(&version, "version", false, "")
	flags.BoolVar(&version, "v", false, "")

	flags.BoolVar(&debug, "debug", false, "")

	// Deprecated
	flags.BoolVar(&stat, "stat", false, "")

	// Parse flags
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeParseFlagsError
	}

	if debug {
		os.Setenv(EnvDebug, "1")
		Debugf("Run as DEBUG mode")
	}

	// Show version and check latest version release
	if version {
		fmt.Fprintf(cli.outStream, OutputVersion())
		return ExitCodeOK
	}

	parsedArgs := flags.Args()
	Debugf("parsed args : %s", parsedArgs)
	var tag, path string
	switch len(parsedArgs) {
	case 1:
		tag, path = parsedArgs[0], ""
	case 2:
		tag, path = parsedArgs[0], parsedArgs[1]
	default:
		PrintRedf(cli.errStream,
			"Invalid number of arguments: you must set a git TAG and optionally a PATH.\n")
		return ExitCodeBadArgs
	}

	// Extract github repository owner username.
	// If it's not provided via command line flag, read it from .gitconfig
	// (github user or git user).
	if len(owner) == 0 {
		origin, err := gitconfig.OriginURL()
		if err == nil {
			owner = retrieveOwnerName(origin)
		}
		if len(owner) == 0 {
			owner, err = gitconfig.GithubUser()
			if err != nil {
				owner, err = gitconfig.Username()
			}

			if err != nil {
				PrintRedf(cli.errStream,
					"Failed to set up ghr: repository owner name not found\n")
				fmt.Fprintf(cli.errStream,
					"Please set it via `-u` option.\n\n"+
						"You can set default owner name in `github.username` or `user.name`\n"+
						"in `~/.gitconfig` file\n")
				return ExitCodeOwnerNotFound
			}
		}
	}
	Debugf("Owner: %s", owner)

	// Extract repository name from files.
	// If not provided, read it from .git/config file.
	if len(repo) == 0 {
		var err error
		repo, err = gitconfig.Repository()
		if err != nil {
			PrintRedf(cli.errStream,
				"Failed to set up ghr: repository name not found\n")
			fmt.Fprintf(cli.errStream,
				"ghr reads it from `.git/config` file. Change directory to \n"+
					"repository root directory or setup git repository.\n"+
					"Or set it via `-r` option.\n")
			return ExitCodeRepoNotFound
		}
	}
	Debugf("Repository: %s", repo)

	// If GitHub API token is not provided via command line flag
	// or env var then read it from .gitconfig file.
	if len(token) == 0 {
		var err error
		token, err = gitconfig.GithubToken()
		if err != nil {
			PrintRedf(cli.errStream, "Failed to set up ghr: token not found\n")
			fmt.Fprintf(cli.errStream,
				"To use ghr, you need a GitHub API token.\n"+
					"Please set it via `%s` env var or `-t` option.\n\n"+
					"If you don't have one, visit official doc (goo.gl/jSnoI)\n"+
					"and get it first.\n",
				EnvGitHubToken)
			return ExitCodeTokenNotFound
		}
	}
	Debugf("Github API Token: %s", maskString(token))

	// Set Base GitHub API URL. Base URL can also be provided via env var for use with GHE.
	baseURLStr := defaultBaseURL
	if urlStr := os.Getenv(EnvGitHubAPI); len(urlStr) != 0 {
		baseURLStr = urlStr
	}
	Debugf("Base GitHub API URL: %s", baseURLStr)

	if parallel <= 0 {
		parallel = runtime.NumCPU()
	}
	Debugf("Parallel factor: %d", parallel)

	localAssets, err := LocalAssets(path)
	if err != nil {
		PrintRedf(cli.errStream,
			"Failed to find assets from %s: %s\n", path, err)
		return ExitCodeError
	}
	Debugf("Number of file to upload: %d", len(localAssets))

	// Create a GitHub client
	gitHubClient, err := NewGitHubClient(owner, repo, token, baseURLStr)
	if err != nil {
		PrintRedf(cli.errStream, "Failed to construct GitHub client: %s\n", err)
		return ExitCodeError
	}

	ghr := GHR{
		GitHub:    gitHubClient,
		outStream: cli.outStream,
	}

	Debugf("Name: %s", name)

	// Prepare create release request
	req := &github.RepositoryRelease{
		Name:            github.String(name),
		TagName:         github.String(tag),
		Prerelease:      github.Bool(prerelease),
		Draft:           github.Bool(draft),
		TargetCommitish: github.String(commitish),
		Body:            github.String(body),
	}

	ctx := context.TODO()

	if soft {
		_, err := ghr.GitHub.GetRelease(ctx, *req.TagName)

		if err == nil {
			fmt.Fprintf(cli.outStream, "ghr aborted since tag `%s` already exists\n", *req.TagName)
			return ExitCodeOK
		}

		if err != nil && err != ErrReleaseNotFound {
			PrintRedf(cli.errStream, "Failed to get GitHub release: %s\n", err)
			return ExitCodeError
		}
	}

	release, err := ghr.CreateRelease(ctx, req, recreate)
	if err != nil {
		PrintRedf(cli.errStream, "Failed to create GitHub release page: %s\n", err)
		return ExitCodeError
	}

	if replace {
		err := ghr.DeleteAssets(ctx, *release.ID, localAssets, parallel)
		if err != nil {
			PrintRedf(cli.errStream, "Failed to delete existing assets: %s\n", err)
			return ExitCodeError
		}
	}

	// FIXME(tcnksm): More ideal way to change this
	// This is for Github enterprise
	if err := ghr.GitHub.SetUploadURL(*release.UploadURL); err != nil {
		fmt.Fprintf(cli.errStream, "Failed to set upload URL %s: %s\n", *release.UploadURL, err)
		return ExitCodeError
	}

	err = ghr.UploadAssets(ctx, *release.ID, localAssets, parallel)
	if err != nil {
		PrintRedf(cli.errStream, "Failed to upload one of assets: %s\n", err)
		return ExitCodeError
	}

	if !draft {
		_, err := ghr.GitHub.EditRelease(ctx, *release.ID, &github.RepositoryRelease{
			Draft: github.Bool(false),
		})
		if err != nil {
			PrintRedf(cli.errStream, "Failed to publish release: %s\n", err)
			return ExitCodeError
		}
	}

	return ExitCodeOK
}

var ownerNameReg = regexp.MustCompile(`([-a-zA-Z0-9]+)/[^/]+$`)

func retrieveOwnerName(repoURL string) string {
	matched := ownerNameReg.FindStringSubmatch(repoURL)
	if len(matched) < 2 {
		return ""
	}
	return matched[1]
}

// maskString is used to mask a string which should not be displayed
// directly, like the auth token
func maskString(s string) string {
	if len(s) < 5 {
		return "**** (masked)"
	}

	return s[:5] + "**** (masked)"
}

var helpText = `Usage: ghr [options...] TAG [PATH]

ghr is a tool to create Release on Github and upload your
artifacts to it. ghr parallelizes upload of multiple artifacts.

You must specify TAG (e.g., v1.0.0) and an optional PATH to local artifacts.
If PATH is directory, ghr globs all files in the directory and
upload it. If PATH is a file then, upload only it.

And you also must provide GitHub API token which has enough permission
(For a private repository you need the 'repo' scope and for a public
repository need 'public_repo' scope). You can get token from GitHub's
account setting page.

You can use ghr on GitHub Enterprise. Set base URL via GITHUB_API
environment variable.

Options:

-username, -owner, -u
	Github repository owner name. By default, ghr extracts it from global
	gitconfig value.

-repository, -r
	GitHub repository name. By default, ghr extracts repository name from
	current directory's .git/config.

-token, -t
	GitHub API Token. By default, ghr reads it from 'GITHUB_TOKEN' env var.

-commitish, -c
	Set target commitish, branch or commit SHA

-name, -n
	GitHub release title. By default the tag is used.

-body, -b
	Set text describing the contents of the release

-draft
	Release as draft (Unpublish)

-prerelease
	Create prerelease

-parallel=-1
	Parallelization factor. This option limits amount of parallelism of
	uploading. By default, ghr uses number of logic CPU.

-delete, -recreate
	Recreate release if it already exists. If want to upload to same release
	and replace use '-replace'.

-replace
	Replace artifacts if it is already uploaded. ghr thinks it's same when
	local artifact base name and uploaded file name are same.

-soft
	Stop uploading if the repository already has release with the specified
	tag.

-version, -v
	Print ghr version and exit

-debug
	Enable debug output
`
