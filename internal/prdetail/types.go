package prdetail

type Input struct {
	Owner      string
	Name       string
	Number     int
	FilesFirst *int
}

type graphQLOutput struct {
	Data graphQLData `json:"data"`
}

type graphQLData struct {
	Repository graphQLRepository `json:"repository"`
}

type graphQLRepository struct {
	PullRequest *graphQLPullRequest `json:"pullRequest"`
}

type graphQLPullRequest struct {
	Number         int                  `json:"number"`
	Title          string               `json:"title"`
	BodyText       string               `json:"bodyText"`
	ReviewDecision *string              `json:"reviewDecision"`
	Author         *actor               `json:"author"`
	MergeCommit    *oidNode             `json:"mergeCommit"`
	BaseRefOid     string               `json:"baseRefOid"`
	HeadRefOid     string               `json:"headRefOid"`
	Additions      int                  `json:"additions"`
	Deletions      int                  `json:"deletions"`
	ChangedFiles   int                  `json:"changedFiles"`
	Reviews        graphQLReviews       `json:"reviews"`
	ReviewThreads  graphQLReviewThreads `json:"reviewThreads"`
	Files          graphQLFiles         `json:"files"`
	Commits        graphQLCommits       `json:"commits"`
}

type actor struct {
	Login string `json:"login"`
}

type oidNode struct {
	Oid string `json:"oid"`
}

type graphQLReviews struct {
	Nodes []graphQLReview `json:"nodes"`
}

type graphQLReview struct {
	ID          string   `json:"id"`
	Author      *actor   `json:"author"`
	State       string   `json:"state"`
	BodyText    string   `json:"bodyText"`
	SubmittedAt *string  `json:"submittedAt"`
	Commit      *oidNode `json:"commit"`
}

type graphQLReviewThreads struct {
	Nodes []graphQLReviewThread `json:"nodes"`
}

type graphQLReviewThread struct {
	IsResolved bool                  `json:"isResolved"`
	Comments   graphQLReviewComments `json:"comments"`
}

type graphQLReviewComments struct {
	Nodes []graphQLReviewComment `json:"nodes"`
}

type graphQLReviewComment struct {
	ID                string   `json:"id"`
	Author            *actor   `json:"author"`
	BodyText          string   `json:"bodyText"`
	Path              *string  `json:"path"`
	CreatedAt         string   `json:"createdAt"`
	Line              *int     `json:"line"`
	OriginalLine      *int     `json:"originalLine"`
	StartLine         *int     `json:"startLine"`
	OriginalStartLine *int     `json:"originalStartLine"`
	Side              *string  `json:"side"`
	StartSide         *string  `json:"startSide"`
	Commit            *oidNode `json:"commit"`
	OriginalCommit    *oidNode `json:"originalCommit"`
}

type graphQLFiles struct {
	TotalCount int           `json:"totalCount"`
	Nodes      []graphQLFile `json:"nodes"`
	PageInfo   PageInfo      `json:"pageInfo"`
}

type graphQLFile struct {
	Path       string `json:"path"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	ChangeType string `json:"changeType"`
}

type graphQLCommits struct {
	Nodes []graphQLCommitNode `json:"nodes"`
}

type graphQLCommitNode struct {
	Commit Commit `json:"commit"`
}

type Output struct {
	Data Data `json:"data"`
}

type Data struct {
	Repository Repository `json:"repository"`
}

type Repository struct {
	PullRequest *PullRequest `json:"pullRequest"`
}

type PullRequest struct {
	Number                int            `json:"number"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	ReviewDecision        *string        `json:"reviewDecision"`
	AuthorLogin           *string        `json:"authorLogin"`
	MergeCommitOid        *string        `json:"mergeCommitOid"`
	ReviewStartCommitOid  *string        `json:"reviewStartCommitOid"`
	ReviewStartConfidence string         `json:"reviewStartConfidence"`
	CodeDiff              CodeDiff       `json:"codeDiff"`
	Conversations         Conversations  `json:"conversations"`
	Commits               []CommitOutput `json:"commits"`
}

type CodeDiff struct {
	Stats         CodeDiffStats  `json:"stats"`
	Files         []File         `json:"files"`
	ExcludedFiles []ExcludedFile `json:"excludedFiles"`
	FilePageInfo  FilePageInfo   `json:"filePageInfo"`
	Strategy      Strategy       `json:"strategy"`
}

type CodeDiffStats struct {
	ChangedFiles int `json:"changedFiles"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
}

type File struct {
	Path       string `json:"path"`
	ChangeType string `json:"changeType"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
}

type ExcludedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type FilePageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
	TotalCount  int     `json:"totalCount"`
}

type PageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}

type Strategy struct {
	BaseCommit string `json:"baseCommit"`
	HeadCommit string `json:"headCommit"`
}

type Conversations struct {
	Reviews       []Review       `json:"reviews"`
	ReviewThreads []ReviewThread `json:"reviewThreads"`
}

type Review struct {
	AuthorLogin *string `json:"authorLogin"`
	State       string  `json:"state"`
	Body        string  `json:"body"`
	SubmittedAt *string `json:"submittedAt"`
	CommitOid   *string `json:"commitOid"`
}

type ReviewThread struct {
	IsResolved bool            `json:"isResolved"`
	Comments   []ReviewComment `json:"comments"`
}

type ReviewComment struct {
	ID                string  `json:"id"`
	AuthorLogin       *string `json:"authorLogin"`
	Body              string  `json:"body"`
	Path              *string `json:"path"`
	CreatedAt         string  `json:"createdAt"`
	Line              *int    `json:"line"`
	OriginalLine      *int    `json:"originalLine"`
	StartLine         *int    `json:"startLine"`
	OriginalStartLine *int    `json:"originalStartLine"`
	Side              *string `json:"side"`
	StartSide         *string `json:"startSide"`
	CommitOid         *string `json:"commitOid"`
	OriginalCommitOid *string `json:"originalCommitOid"`
}

type CommitOutput struct {
	Oid             string `json:"oid"`
	MessageHeadline string `json:"messageHeadline"`
	CommittedDate   string `json:"committedDate"`
}

type Commit struct {
	Oid             string `json:"oid"`
	MessageHeadline string `json:"messageHeadline"`
	CommittedDate   string `json:"committedDate"`
}
