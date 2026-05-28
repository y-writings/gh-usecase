export const GRAPHQL_QUERY = `query ($owner: String!, $name: String!, $number: Int!, $filesFirst: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      number
      title
      bodyText
      reviewDecision
      author {
        login
      }
      mergeCommit {
        oid
      }
      baseRefOid
      headRefOid
      additions
      deletions
      changedFiles
      reviews(first: 100) {
        nodes {
          id
          author {
            login
          }
          state
          bodyText
          submittedAt
          commit {
            oid
          }
        }
      }
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 100) {
            nodes {
              id
              author {
                login
              }
              bodyText
              path
              createdAt
              line
              originalLine
              startLine
              originalStartLine
              side
              startSide
              commit {
                oid
              }
              originalCommit {
                oid
              }
            }
          }
        }
      }
      files(first: $filesFirst) {
        totalCount
        nodes {
          path
          additions
          deletions
          changeType
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
      commits(first: 100) {
        nodes {
          commit {
            oid
            messageHeadline
            committedDate
          }
        }
      }
    }
  }
}`;
