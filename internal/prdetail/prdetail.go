package prdetail

import (
	"context"
	"sort"
	"time"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

func Execute(ctx context.Context, client githubapi.GraphQLClient, input Input) (Output, error) {
	if err := Validate(input); err != nil {
		return Output{}, err
	}

	filesFirst := 40
	if input.FilesFirst != nil {
		filesFirst = *input.FilesFirst
	}

	variables := map[string]interface{}{
		"owner":      input.Owner,
		"name":       input.Name,
		"number":     input.Number,
		"filesFirst": filesFirst,
	}

	var response graphQLOutput
	if err := client.DoWithContext(ctx, graphQLQuery, variables, &response); err != nil {
		return Output{}, err
	}

	return transformOutput(response), nil
}

func Validate(input Input) error {
	if input.Owner == "" {
		return validation.New("owner is required")
	}
	if input.Name == "" {
		return validation.New("name is required")
	}
	if input.Number <= 0 {
		return validation.New("number must be a positive integer")
	}

	filesFirst := 40
	if input.FilesFirst != nil {
		filesFirst = *input.FilesFirst
	}
	if filesFirst < 1 || filesFirst > 100 {
		return validation.New("filesFirst must be between 1 and 100")
	}
	return nil
}

func transformOutput(input graphQLOutput) Output {
	pr := input.Data.Repository.PullRequest
	if pr == nil {
		return Output{Data: Data{Repository: Repository{PullRequest: nil}}}
	}

	files := make([]File, 0, len(pr.Files.Nodes))
	excludedFiles := make([]ExcludedFile, 0)
	for _, file := range pr.Files.Nodes {
		if isLikelyGeneratedFile(file.Path) {
			excludedFiles = append(excludedFiles, ExcludedFile{Path: file.Path, Reason: "likely-generated"})
			continue
		}
		if isLikelyBinaryFile(file.Path) {
			excludedFiles = append(excludedFiles, ExcludedFile{Path: file.Path, Reason: "likely-binary"})
			continue
		}
		files = append(files, File{
			Path:       file.Path,
			ChangeType: file.ChangeType,
			Additions:  file.Additions,
			Deletions:  file.Deletions,
		})
	}

	reviews := make([]Review, 0, len(pr.Reviews.Nodes))
	for _, review := range pr.Reviews.Nodes {
		reviews = append(reviews, Review{
			AuthorLogin: loginOf(review.Author),
			State:       review.State,
			Body:        review.BodyText,
			SubmittedAt: review.SubmittedAt,
			CommitOid:   oidOf(review.Commit),
		})
	}

	reviewThreads := make([]ReviewThread, 0, len(pr.ReviewThreads.Nodes))
	for _, thread := range pr.ReviewThreads.Nodes {
		comments := make([]ReviewComment, 0, len(thread.Comments.Nodes))
		for _, comment := range thread.Comments.Nodes {
			comments = append(comments, ReviewComment{
				ID:                comment.ID,
				AuthorLogin:       loginOf(comment.Author),
				Body:              comment.BodyText,
				Path:              comment.Path,
				CreatedAt:         comment.CreatedAt,
				Line:              comment.Line,
				OriginalLine:      comment.OriginalLine,
				StartLine:         comment.StartLine,
				OriginalStartLine: comment.OriginalStartLine,
				Side:              comment.Side,
				StartSide:         comment.StartSide,
				CommitOid:         oidOf(comment.Commit),
				OriginalCommitOid: oidOf(comment.OriginalCommit),
			})
		}
		reviewThreads = append(reviewThreads, ReviewThread{IsResolved: thread.IsResolved, Comments: comments})
	}

	commits := make([]CommitOutput, 0, len(pr.Commits.Nodes))
	for _, node := range pr.Commits.Nodes {
		commits = append(commits, CommitOutput{
			Oid:             node.Commit.Oid,
			MessageHeadline: node.Commit.MessageHeadline,
			CommittedDate:   node.Commit.CommittedDate,
		})
	}

	reviewStartCommitOid, reviewStartConfidence := getReviewStartAnchor(pr)
	return Output{Data: Data{Repository: Repository{PullRequest: &PullRequest{
		Number:                pr.Number,
		Title:                 pr.Title,
		Description:           pr.BodyText,
		ReviewDecision:        pr.ReviewDecision,
		AuthorLogin:           loginOf(pr.Author),
		MergeCommitOid:        oidOf(pr.MergeCommit),
		ReviewStartCommitOid:  reviewStartCommitOid,
		ReviewStartConfidence: reviewStartConfidence,
		CodeDiff: CodeDiff{
			Stats:         CodeDiffStats{ChangedFiles: pr.ChangedFiles, Additions: pr.Additions, Deletions: pr.Deletions},
			Files:         files,
			ExcludedFiles: excludedFiles,
			FilePageInfo:  FilePageInfo{HasNextPage: pr.Files.PageInfo.HasNextPage, EndCursor: pr.Files.PageInfo.EndCursor, TotalCount: pr.Files.TotalCount},
			Strategy:      Strategy{BaseCommit: pr.BaseRefOid, HeadCommit: pr.HeadRefOid},
		},
		Conversations: Conversations{Reviews: reviews, ReviewThreads: reviewThreads},
		Commits:       commits,
	}}}}
}

func loginOf(actor *actor) *string {
	if actor == nil {
		return nil
	}
	return &actor.Login
}

func oidOf(node *oidNode) *string {
	if node == nil {
		return nil
	}
	return &node.Oid
}

type datedCommit struct {
	commit Commit
	ts     time.Time
}

type commitCandidate struct {
	ts  time.Time
	oid string
}

func getReviewStartAnchor(pr *graphQLPullRequest) (*string, string) {
	commitsByDate := make([]datedCommit, 0, len(pr.Commits.Nodes))
	commitSet := map[string]struct{}{}
	for _, node := range pr.Commits.Nodes {
		commitSet[node.Commit.Oid] = struct{}{}
		commitsByDate = append(commitsByDate, datedCommit{commit: node.Commit, ts: parseTime(node.Commit.CommittedDate)})
	}
	sort.Slice(commitsByDate, func(i, j int) bool { return commitsByDate[i].ts.Before(commitsByDate[j].ts) })

	candidates := make([]commitCandidate, 0)
	for _, review := range pr.Reviews.Nodes {
		if review.Commit == nil || review.SubmittedAt == nil {
			continue
		}
		candidates = append(candidates, commitCandidate{ts: parseTime(*review.SubmittedAt), oid: review.Commit.Oid})
	}
	for _, thread := range pr.ReviewThreads.Nodes {
		for _, comment := range thread.Comments.Nodes {
			oid := oidOf(comment.OriginalCommit)
			if oid == nil {
				oid = oidOf(comment.Commit)
			}
			if oid == nil {
				continue
			}
			candidates = append(candidates, commitCandidate{ts: parseTime(comment.CreatedAt), oid: *oid})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ts.Before(candidates[j].ts) })
	if len(candidates) > 0 {
		oid := candidates[0].oid
		confidence := "medium"
		if _, ok := commitSet[oid]; ok {
			confidence = "high"
		}
		return &oid, confidence
	}

	activityTimestamps := make([]time.Time, 0)
	for _, review := range pr.Reviews.Nodes {
		if review.SubmittedAt != nil {
			activityTimestamps = append(activityTimestamps, parseTime(*review.SubmittedAt))
		}
	}
	for _, thread := range pr.ReviewThreads.Nodes {
		for _, comment := range thread.Comments.Nodes {
			activityTimestamps = append(activityTimestamps, parseTime(comment.CreatedAt))
		}
	}
	if len(activityTimestamps) > 0 && len(commitsByDate) > 0 {
		earliestActivity := activityTimestamps[0]
		for _, ts := range activityTimestamps[1:] {
			if ts.Before(earliestActivity) {
				earliestActivity = ts
			}
		}

		var chosen *string
		for _, commit := range commitsByDate {
			if commit.ts.Before(earliestActivity) || commit.ts.Equal(earliestActivity) {
				oid := commit.commit.Oid
				chosen = &oid
			}
		}
		if chosen != nil {
			return chosen, "medium"
		}

		oid := commitsByDate[len(commitsByDate)-1].commit.Oid
		return &oid, "low"
	}

	return nil, "low"
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
