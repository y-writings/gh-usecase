package prlist

const graphQLQuery = `query ($owner: String!, $name: String!, $state: [PullRequestState!], $first: Int!, $after: String) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: $state, first: $first, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
        nodes {
          number
          createdAt
          state
          mergedAt
          changedFiles
          reviewDecision
          comments {
            totalCount
          }
          author {
            login
          }
          reviewRequests(first: 20) {
            nodes {
              requestedReviewer {
                ... on User {
                  login
                }
                ... on Team {
                  slug
                }
                ... on Bot {
                  login
                }
                ... on Mannequin {
                  login
                }
              }
            }
          }
          reviews(first: 20) {
            nodes {
              author {
                login
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }`
