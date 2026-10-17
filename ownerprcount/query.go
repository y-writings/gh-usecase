package ownerprcount

const graphQLQuery = `query ($owner: String!, $first: Int!, $after: String) {
  repositoryOwner(login: $owner) {
    repositories(first: $first, after: $after, ownerAffiliations: OWNER, orderBy: {field: NAME, direction: ASC}) {
      nodes {
        nameWithOwner
        pullRequests(states: [OPEN]) {
          totalCount
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}`
