import { isLikelyBinaryFile, isLikelyGeneratedFile } from './file-utils';
import type { PrDetailGraphQlOutput, PrDetailOutput } from './schema';

type PrDetailPayload = NonNullable<PrDetailOutput['data']['repository']['pullRequest']>;

type ReviewStartConfidence = PrDetailPayload['reviewStartConfidence'];

function getReviewStartAnchor(
  pullRequest: NonNullable<PrDetailGraphQlOutput['data']['repository']['pullRequest']>,
): {
  reviewStartCommitOid: string | null;
  reviewStartConfidence: ReviewStartConfidence;
} {
  const commitsByDate = [...pullRequest.commits.nodes]
    .map(({ commit }) => commit)
    .sort(
      (left, right) =>
        new Date(left.committedDate).getTime() - new Date(right.committedDate).getTime(),
    );
  const commitSet = new Set(commitsByDate.map((commit) => commit.oid));

  const timestampedCommitCandidates: Array<{ ts: number; oid: string }> = [];

  for (const review of pullRequest.reviews.nodes) {
    if (!review.commit?.oid || !review.submittedAt) {
      continue;
    }

    timestampedCommitCandidates.push({
      ts: new Date(review.submittedAt).getTime(),
      oid: review.commit.oid,
    });
  }

  for (const thread of pullRequest.reviewThreads.nodes) {
    for (const comment of thread.comments.nodes) {
      const oid = comment.originalCommit?.oid ?? comment.commit?.oid;
      if (!oid) {
        continue;
      }

      timestampedCommitCandidates.push({
        ts: new Date(comment.createdAt).getTime(),
        oid,
      });
    }
  }

  const sortedTimestampedCommitCandidates = [...timestampedCommitCandidates].sort(
    (left, right) => left.ts - right.ts,
  );

  const directCandidate = sortedTimestampedCommitCandidates[0];
  if (directCandidate) {
    return {
      reviewStartCommitOid: directCandidate.oid,
      reviewStartConfidence: commitSet.has(directCandidate.oid) ? 'high' : 'medium',
    };
  }

  const activityTimestamps: number[] = [];
  for (const review of pullRequest.reviews.nodes) {
    if (review.submittedAt) {
      activityTimestamps.push(new Date(review.submittedAt).getTime());
    }
  }
  for (const thread of pullRequest.reviewThreads.nodes) {
    for (const comment of thread.comments.nodes) {
      activityTimestamps.push(new Date(comment.createdAt).getTime());
    }
  }

  if (activityTimestamps.length > 0 && commitsByDate.length > 0) {
    const earliestActivity = Math.min(...activityTimestamps);
    let chosen: string | null = null;
    for (const commit of commitsByDate) {
      const commitTs = new Date(commit.committedDate).getTime();
      if (commitTs <= earliestActivity) {
        chosen = commit.oid;
      }
    }

    if (chosen) {
      return {
        reviewStartCommitOid: chosen,
        reviewStartConfidence: 'medium',
      };
    }

    return {
      reviewStartCommitOid: commitsByDate[commitsByDate.length - 1]?.oid ?? null,
      reviewStartConfidence: 'low',
    };
  }

  return {
    reviewStartCommitOid: null,
    reviewStartConfidence: 'low',
  };
}

export function transformOutput(graphQlOutput: PrDetailGraphQlOutput): PrDetailOutput {
  const pullRequest = graphQlOutput.data.repository.pullRequest;
  if (!pullRequest) {
    return {
      data: {
        repository: {
          pullRequest: null,
        },
      },
    };
  }

  const includedFiles: PrDetailPayload['codeDiff']['files'] = [];
  const excludedFiles: PrDetailPayload['codeDiff']['excludedFiles'] = [];

  for (const file of pullRequest.files.nodes) {
    if (isLikelyGeneratedFile(file.path)) {
      excludedFiles.push({ path: file.path, reason: 'likely-generated' });
      continue;
    }

    if (isLikelyBinaryFile(file.path)) {
      excludedFiles.push({ path: file.path, reason: 'likely-binary' });
      continue;
    }

    includedFiles.push({
      path: file.path,
      changeType: file.changeType,
      additions: file.additions,
      deletions: file.deletions,
    });
  }

  const reviewThreads = pullRequest.reviewThreads.nodes.map((thread) => ({
    isResolved: thread.isResolved,
    comments: thread.comments.nodes.map((comment) => ({
      id: comment.id,
      authorLogin: comment.author?.login ?? null,
      body: comment.bodyText,
      path: comment.path,
      createdAt: comment.createdAt,
      line: comment.line,
      originalLine: comment.originalLine,
      startLine: comment.startLine,
      originalStartLine: comment.originalStartLine,
      side: comment.side,
      startSide: comment.startSide,
      commitOid: comment.commit?.oid ?? null,
      originalCommitOid: comment.originalCommit?.oid ?? null,
    })),
  }));

  const reviewStartAnchor = getReviewStartAnchor(pullRequest);

  return {
    data: {
      repository: {
        pullRequest: {
          number: pullRequest.number,
          title: pullRequest.title,
          description: pullRequest.bodyText,
          reviewDecision: pullRequest.reviewDecision,
          authorLogin: pullRequest.author?.login ?? null,
          mergeCommitOid: pullRequest.mergeCommit?.oid ?? null,
          reviewStartCommitOid: reviewStartAnchor.reviewStartCommitOid,
          reviewStartConfidence: reviewStartAnchor.reviewStartConfidence,
          codeDiff: {
            stats: {
              changedFiles: pullRequest.changedFiles,
              additions: pullRequest.additions,
              deletions: pullRequest.deletions,
            },
            files: includedFiles,
            excludedFiles,
            filePageInfo: {
              hasNextPage: pullRequest.files.pageInfo.hasNextPage,
              endCursor: pullRequest.files.pageInfo.endCursor,
              totalCount: pullRequest.files.totalCount,
            },
            strategy: {
              baseCommit: pullRequest.baseRefOid,
              headCommit: pullRequest.headRefOid,
            },
          },
          conversations: {
            reviews: pullRequest.reviews.nodes.map((review) => ({
              authorLogin: review.author?.login ?? null,
              state: review.state,
              body: review.bodyText,
              submittedAt: review.submittedAt,
              commitOid: review.commit?.oid ?? null,
            })),
            reviewThreads,
          },
          commits: pullRequest.commits.nodes.map(({ commit }) => ({
            oid: commit.oid,
            messageHeadline: commit.messageHeadline,
            committedDate: commit.committedDate,
          })),
        },
      },
    },
  };
}
