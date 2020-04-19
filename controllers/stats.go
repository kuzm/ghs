package controllers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/horalstvo/ghs/external"
	"github.com/horalstvo/ghs/models"
	"github.com/horalstvo/ghs/util"
	"github.com/logrusorgru/aurora"
)

// GetStats returns pull request data
func GetStats(config models.StatsConfig) {
	ctx := context.Background()

	client := external.GetClient(ctx, config.ApiToken)

	repos := make([]*github.Repository, 0)

	if len(config.Org) > 0 {
		repos = external.GetTeamRepos(ctx, config.Org, config.Team, client)
	} else if len(config.Owner) > 0 {
		repos = external.GetOwnerRepos(ctx, config.Owner, client)
	} else {
		fmt.Println("Both organization and owner cannot be specified")
		os.Exit(1)
	}

	owner := config.Org
	if len(config.Org) == 0 {
		owner = config.Owner
	}

	prs := getPullRequests(ctx, owner, repos, client)

	prs = filterPullRequests(prs, config.Start, config.End)

	fmt.Printf("Number of PRs opened in the interval: %d\n", aurora.Blue(len(prs)))

	pullRequests := getDetails(ctx, owner, prs, client)

	fmt.Printf("Writing to CSV file '%v'...\n", config.File)
	file, err := os.Create(config.File)
	util.Check(err)
	defer file.Close()

	writer := bufio.NewWriter(file)

	fmt.Fprintf(writer, "Repo,Number,Created,FirstReviewedHrs,FirstApprovedHrs,SecondApprovedHrs,MergedHrs,Merged,ChangedFiles,Additions,Deletions\n")
	for _, pr := range pullRequests {
		dtFormat := "2006-01-02T15:04:05-0700"
		created := pr.Created.Format(dtFormat)
		merged := ""
		if !pr.Merged.IsZero() {
			merged = pr.Merged.Format(dtFormat)
		}

		firstReviewedHrs := ""
		if pr.FirstReviewedHrs >= 0 {
			firstReviewedHrs = strconv.Itoa(pr.FirstReviewedHrs)
		}
		firstApprovedHrs := ""
		if pr.FirstApprovedHrs >= 0 {
			firstApprovedHrs = strconv.Itoa(pr.FirstApprovedHrs)
		}
		secondApprovedHrs := ""
		if pr.SecondApprovedHrs >= 0 {
			secondApprovedHrs = strconv.Itoa(pr.SecondApprovedHrs)
		}
		mergedHrs := ""
		if pr.MergedHrs >= 0 {
			mergedHrs = strconv.Itoa(pr.MergedHrs)
		}
		fmt.Fprintf(writer, "%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v\n", pr.Repo, pr.Number, created,
			firstReviewedHrs, firstApprovedHrs, secondApprovedHrs, mergedHrs, merged, pr.ChangedFiles, pr.Additions, pr.Deletions)
	}

	writer.Flush()
	file.Sync()

	fmt.Println("Writing to CSV file completed")

}

func getPullRequests(ctx context.Context, owner string, repos []*github.Repository, client *github.Client) []*github.PullRequest {

	prsPerRepo := make([][]*github.PullRequest, len(repos))

	for i, repo := range repos {
		prsPerRepo[i] = external.GetPullRequests(ctx, owner, *repo.Name, client)
		fmt.Printf("Number of PRs returned for %s: %d\n", *repo.Name, aurora.Blue(len(prsPerRepo[i])))
	}

	prs := make([]*github.PullRequest, 0)
	for i := range repos {
		prs = append(prs, prsPerRepo[i]...)
	}
	return prs
}

func getPullRequest(ctx context.Context, owner string, repo string, number int,
	client *github.Client) *github.PullRequest {

	pr := external.GetPullRequest(ctx, owner, repo, number, client)
	return pr
}

func getDetails(ctx context.Context, owner string, prs []*github.PullRequest,
	client *github.Client) []models.PullRequest {

	fmt.Printf("Getting details for %v pull requests...\n", len(prs))

	pullRequests := make([]models.PullRequest, len(prs))

	for i, pr := range prs {
		pullRequests[i] = getPullRequestDetails(ctx, owner, pr, client)

		// Throttle number of sequential requests to GitHub API
		if (i+1)%25 == 0 {
			fmt.Printf("%v pull requests were processed\n", i+1)
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("Done")

	sort.Slice(pullRequests, func(i, j int) bool {
		return pullRequests[i].Created.Before(pullRequests[j].Created)
	})

	return pullRequests
}

func getPullRequestDetails(ctx context.Context, owner string, pr *github.PullRequest,
	client *github.Client) models.PullRequest {

	prDetails := getPullRequest(ctx, owner, *pr.Base.Repo.Name, *pr.Number, client)
	reviews := getReviews(ctx, owner, *pr.Base.Repo.Name, *pr.Number, client)

	firstReviewedHrs := -1
	firstApprovedHrs := -1
	secondApprovedHrs := -1
	mergedHrs := -1

	merged := time.Time{}

	changedFiles := -1
	additions := -1
	deletions := -1

	if len(reviews) > 0 {
		firstReview := reviews[0]
		approvals := getApprovals(reviews)
		firstReviewedHrs = util.WorkHours(*pr.CreatedAt, firstReview.Submitted)

		if len(approvals) > 0 {
			firstApproval := approvals[0]
			firstApprovedHrs = util.WorkHours(*pr.CreatedAt, firstApproval.Submitted)
		}

		if len(approvals) > 1 {
			secondApproval := approvals[1]
			secondApprovedHrs = util.WorkHours(*pr.CreatedAt, secondApproval.Submitted)
		}
	}

	if prDetails.MergedAt != nil {
		merged = *pr.MergedAt
		mergedHrs = util.WorkHours(*pr.CreatedAt, *prDetails.MergedAt)
	}

	if prDetails.ChangedFiles != nil {
		changedFiles = *prDetails.ChangedFiles
	}

	if prDetails.Additions != nil {
		additions = *prDetails.Additions
	}

	if prDetails.Deletions != nil {
		deletions = *prDetails.Deletions
	}

	pullRequest := models.PullRequest{
		Repo:    *pr.Base.Repo.Name,
		Number:  *pr.Number,
		Created: *pr.CreatedAt,
		Merged:  merged,

		FirstReviewedHrs:  firstReviewedHrs,
		FirstApprovedHrs:  firstApprovedHrs,
		SecondApprovedHrs: secondApprovedHrs,
		MergedHrs:         mergedHrs,

		ChangedFiles: changedFiles,
		Additions:    additions,
		Deletions:    deletions,
	}

	return pullRequest
}

func getColored(hours int, percentile float64) aurora.Value {
	if float64(hours) >= percentile {
		return aurora.Red(hours)
	}
	return aurora.Gray(hours)
}

func getApprovals(reviews []models.Review) []models.Review {
	approvals := make([]models.Review, 0)
	for _, rev := range reviews {
		if rev.Status == "APPROVED" {
			approvals = append(approvals, rev)
		}
	}
	return approvals
}

func getReviews(ctx context.Context, org string, repo string, number int, client *github.Client) []models.Review {
	rawReviews := external.GetReviews(ctx, org, repo, number, client)
	reviews := make([]models.Review, 0)
	for _, rev := range rawReviews {
		if rev.SubmittedAt != nil {
			reviews = append(reviews, models.Review{
				Author:    *rev.User.Login,
				Status:    *rev.State,
				Submitted: *rev.SubmittedAt,
			})
		} else {
			fmt.Printf("Skipping %s:%s:%d - nil for submittedAt.\n%v\n", org, repo, number, rev)
		}
	}
	return reviews
}

func filterPullRequests(prs []*github.PullRequest, startDays int, endDays int) []*github.PullRequest {
	from := time.Now().AddDate(0, 0, startDays)
	to := time.Now().AddDate(0, 0, endDays)
	filteredPrs := filter(prs, func(request *github.PullRequest) bool {
		return request.CreatedAt.After(from) && request.CreatedAt.Before(to)
	})
	return filteredPrs
}

func filter(prs []*github.PullRequest, fn func(*github.PullRequest) bool) []*github.PullRequest {
	filtered := make([]*github.PullRequest, 0)
	for _, pr := range prs {
		if fn(pr) {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}
