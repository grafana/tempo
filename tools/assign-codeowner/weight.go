package main

import (
	"context"
	"strings"

	"github.com/google/go-github/v93/github"
)

// A pull request contributes 1..5 to its owner's load based on its diff size,
// so a large PR counts as more work than a one-liner. Lines under the first
// tier weigh 1; each tier crossed adds 1, capping at 5.
//
// Tiers are on non-vendored, non-generated lines:
//
//	<100   -> 1
//	100+   -> 2
//	250+   -> 3
//	500+   -> 4
//	2000+  -> 5
var weightTiers = []int{100, 250, 500, 2000}

// Paths whose diff lines never count toward a PR's weight: vendored code,
// generated protobufs, and compiled jsonnet. Keeping vendored/generated churn
// out prevents a single dependency bump from dominating the load balance.
var (
	weightIgnoredPrefixes = []string{"vendor/", "opentelemetry-proto/", "operations/jsonnet-compiled/"}
	weightIgnoredSuffixes = []string{".pb.go", ".y.go", ".gen.go"}
)

// sizeWeight maps a PR's counted line total to its 1..5 load weight.
func sizeWeight(lines int) int {
	weight := 1
	for _, tier := range weightTiers {
		if lines >= tier {
			weight++
		}
	}
	return weight
}

// isIgnoredWeightPath reports whether a changed file is vendored or generated
// and therefore excluded from the size weight.
func isIgnoredWeightPath(path string) bool {
	for _, prefix := range weightIgnoredPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, suffix := range weightIgnoredSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// weightOrDefault treats an unset weight as 1, so a PR whose size was not
// computed still counts as a single unit of load.
func weightOrDefault(weight int) int {
	if weight < 1 {
		return 1
	}
	return weight
}

// pullWeight fetches the PR's changed files and returns its size weight from
// the lines outside vendored/generated paths. It costs one paginated request
// per PR.
func pullWeight(ctx context.Context, client *github.Client, number int) (int, error) {
	files, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.CommitFile, *github.Response, error) {
		return client.PullRequests.ListFiles(ctx, repoOwner, repoName, number, opts)
	})
	if err != nil {
		return 0, err
	}
	lines := 0
	for _, file := range files {
		if isIgnoredWeightPath(file.GetFilename()) {
			continue
		}
		lines += file.GetAdditions() + file.GetDeletions()
	}
	return sizeWeight(lines), nil
}
