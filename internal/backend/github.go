package backend

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
)

type GitHubClient struct {
	client *github.Client
}

func NewGitHubClient(token string, baseURL ...string) *GitHubClient {
	client := github.NewClient(nil).WithAuthToken(token)

	if len(baseURL) > 0 && baseURL[0] != "" {
		// For testing with httptest
		client, _ = client.WithEnterpriseURLs(baseURL[0]+"/", baseURL[0]+"/")
	}

	return &GitHubClient{client: client}
}

func (g *GitHubClient) ListOpenPRs(owner, repo string) ([]PR, error) {
	return g.ListOpenPRsWithContext(context.Background(), owner, repo)
}

func (g *GitHubClient) ListOpenPRsWithContext(ctx context.Context, owner, repo string) ([]PR, error) {
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	ghPRs, _, err := g.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	prs := make([]PR, len(ghPRs))
	for i, p := range ghPRs {
		labels := make([]string, len(p.Labels))
		for j, l := range p.Labels {
			labels[j] = l.GetName()
		}

		prs[i] = PR{
			Number:     p.GetNumber(),
			Title:      p.GetTitle(),
			URL:        p.GetHTMLURL(),
			Branch:     p.GetHead().GetRef(),
			Base:       p.GetBase().GetRef(),
			State:      p.GetState(),
			CreatedAt:  p.GetCreatedAt().Time,
			Repo:       owner + "/" + repo,
			Labels:     labels,
			UpdateType: classifyUpdateType(labels, p.GetTitle()),
		}
	}
	return prs, nil
}

func (g *GitHubClient) GetPRMergeability(owner, repo string, number int) (mergeable bool, checksPass bool, err error) {
	return g.GetPRMergeabilityWithContext(context.Background(), owner, repo, number)
}

func (g *GitHubClient) GetPRMergeabilityWithContext(ctx context.Context, owner, repo string, number int) (mergeable bool, checksPass bool, err error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, false, err
	}
	mergeable = pr.GetMergeable()

	// Check combined status
	statuses, _, err := g.client.Repositories.GetCombinedStatus(ctx, owner, repo, pr.GetHead().GetRef(), nil)
	if err != nil {
		return mergeable, false, nil // non-fatal
	}
	checksPass = statuses.GetState() == "success" || statuses.GetTotalCount() == 0

	return mergeable, checksPass, nil
}

func (g *GitHubClient) MergePR(owner, repo string, number int) error {
	return g.MergePRWithContext(context.Background(), owner, repo, number)
}

func (g *GitHubClient) MergePRWithContext(ctx context.Context, owner, repo string, number int) error {
	_, _, err := g.client.PullRequests.Merge(ctx, owner, repo, number, "", &github.PullRequestOptions{
		MergeMethod: "merge",
	})
	return err
}

func (g *GitHubClient) ClosePR(owner, repo string, number int, branch string) error {
	return g.ClosePRWithContext(context.Background(), owner, repo, number, branch)
}

func (g *GitHubClient) ClosePRWithContext(ctx context.Context, owner, repo string, number int, branch string) error {
	state := "closed"
	_, _, err := g.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		State: &state,
	})
	if err != nil {
		return err
	}

	// Delete branch — best effort
	g.client.Git.DeleteRef(ctx, owner, repo, "heads/"+branch)
	return nil
}

func (g *GitHubClient) ListOwnerRepos(owner string) ([]string, error) {
	return g.ListOwnerReposWithContext(context.Background(), owner)
}

func (g *GitHubClient) ListOwnerReposWithContext(ctx context.Context, owner string) ([]string, error) {
	opts := &github.RepositoryListByUserOptions{
		Type:        "sources",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []string
	for {
		repos, resp, err := g.client.Repositories.ListByUser(ctx, owner, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			if !r.GetArchived() {
				allRepos = append(allRepos, r.GetName())
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos, nil
}

// classifyUpdateType determines if a Renovate PR is major/minor/patch.
func classifyUpdateType(labels []string, title string) string {
	for _, l := range labels {
		switch {
		case strings.Contains(l, "major"):
			return "major"
		case strings.Contains(l, "minor"):
			return "minor"
		case strings.Contains(l, "patch"):
			return "patch"
		case strings.Contains(l, "digest"):
			return "digest"
		case strings.Contains(l, "pin"):
			return "pin"
		}
	}

	titleLower := strings.ToLower(title)
	if strings.Contains(titleLower, "(major)") {
		return "major"
	}
	if strings.Contains(titleLower, "(minor)") {
		return "minor"
	}
	if strings.Contains(titleLower, "(patch)") {
		return "patch"
	}
	return ""
}

// IsSafeToMerge returns true if a PR is minor/patch, mergeable, and checks pass.
func IsSafeToMerge(pr PR) bool {
	return (pr.UpdateType == "minor" || pr.UpdateType == "patch") &&
		pr.Mergeable && pr.ChecksPass
}

// RelativeTime formats a time as "2m ago", "3d ago", etc.
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// OpenInBrowser opens a URL in the default browser.
func OpenInBrowser(url string) *http.Request {
	// This is a placeholder — the actual browser opening is handled by the TUI.
	return nil
}
