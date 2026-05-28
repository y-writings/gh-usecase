# pr-detail 最小項目定義（5つの分析目的対応）

このドキュメントは、以下5目的を達成するための `pr-detail` 必要最低限項目を定義する。
一般的なPRメトリクス収集は対象外とし、目的に直接効かない項目は削除対象とする。

## 分析目的

1. 過去に「誰が・誰に・どのような指摘をしたか」のトレーサビリティ
2. リポジトリ全体で多い指摘の傾向・パターン
3. レビュー前後のコード品質の変化
4. 最もコード品質を向上させたレビュアーの特定
5. 最もコード品質を向上させたレビューコメントの特定

## 必要最低限の項目（採用）

### A. PR識別・品質比較アンカー

- `data.repository.pullRequest.number`
  - 全分析結果をPR単位で再結合する主キー。
- `data.repository.pullRequest.authorLogin`
  - 目的1の「誰に」を確定するために必須。
- `data.repository.pullRequest.reviewDecision`
  - 目的3/4/5で最終結果との関係を評価するために必要。
- `data.repository.pullRequest.reviewStartCommitOid`
  - 目的3/5の「レビュー開始時点」比較基準。
- `data.repository.pullRequest.reviewStartConfidence`
  - `reviewStartCommitOid` の推定信頼度。分析時の重み付けに必要。
- `data.repository.pullRequest.mergeCommitOid`
  - 目的3/5の「マージ時点」比較基準。

### B. 差分の対象と規模

- `data.repository.pullRequest.codeDiff.files[].path`
  - 指摘対象ファイル特定の中心軸（目的1/2/5）。
- `data.repository.pullRequest.codeDiff.files[].changeType`
  - 変更タイプ別の傾向分析（目的2）。
- `data.repository.pullRequest.codeDiff.files[].additions`
- `data.repository.pullRequest.codeDiff.files[].deletions`
  - 変更規模と指摘密度・品質変化の相関分析（目的2/3）。

### C. レビュー単位のイベント

- `data.repository.pullRequest.conversations.reviews[].authorLogin` (追加)
  - 目的1/4で「誰が指摘したか」を確定するために必須。
- `data.repository.pullRequest.conversations.reviews[].state`
  - 指摘強度（承認/差し戻し等）の区分（目的2/3/4/5）。
- `data.repository.pullRequest.conversations.reviews[].body`
  - 指摘種別・内容分類（目的1/2/5）。
- `data.repository.pullRequest.conversations.reviews[].submittedAt`
  - 時系列分析（目的1/3）。
- `data.repository.pullRequest.conversations.reviews[].commitOid`
  - レビュー時点コードスナップショット（目的3/5）。

### D. コメント単位のイベント

- `data.repository.pullRequest.conversations.reviewThreads[].isResolved`
  - 指摘解決有無の観測（目的3/5）。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].id` (追加)
  - 目的5でコメント効果を安定再識別するための主キー。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].authorLogin` (追加)
  - 目的1/4/5でコメント貢献者を確定。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].body`
  - 指摘内容の中核データ（目的1/2/5）。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].path`
  - 指摘対象ファイル（目的1/2/5）。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].createdAt`
  - コメント時系列（目的1/3）。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].line`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].originalLine`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].startLine`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].originalStartLine`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].side`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].startSide`
  - 行単位アンカー再現（目的1/5）。
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].commitOid`
- `data.repository.pullRequest.conversations.reviewThreads[].comments[].originalCommitOid`
  - コメント文脈のコミット固定（目的3/5）。

## 削除対象（確実に不要、または存在価値が低い）

以下は上記5目的の達成に対して直接寄与しないため、最小構成から除外する。

### 明確に不要（最小構成では削除）

- `data.repository.pullRequest.title`
  - 文脈補助には使えるが、5目的の達成に必須ではない。
- `data.repository.pullRequest.description`
  - 同上。コメント本文と差分情報で代替可能。
- `data.repository.pullRequest.codeDiff.stats.*`
  - `codeDiff.files[]` から再計算可能。
- `data.repository.pullRequest.codeDiff.excludedFiles.*`
  - 透明性/説明用。分析本体には非必須。
- `data.repository.pullRequest.codeDiff.filePageInfo.*`
  - 取得制御用メタデータであり、分析指標には非必須。
- `data.repository.pullRequest.commits[].messageHeadline`
  - 品質分析・指摘効果測定への直接寄与が低い。

### 価値はあるが最小構成では外す

- `data.repository.pullRequest.codeDiff.strategy.baseCommit`
- `data.repository.pullRequest.codeDiff.strategy.headCommit`
  - 後追いで詳細差分を再取得する運用には有用。
  - ただし本ドキュメントの「必要最低限」からは除外する。

## ハッシュ取得方針（レビュー開始 / マージ時点）

`pr-detail` では、以下を分析上の正規アンカーとして扱う。

1. レビュー開始ハッシュ: `reviewStartCommitOid`
   - 優先: レビュー/レビューコメント最初の活動にひもづく `commitOid` / `originalCommitOid` を採用。
   - 補助: `reviewStartConfidence` で推定品質を管理。
   - 目的: 「PR作成時点」ではなく「レビュー開始時点」を固定する。

2. マージ時点ハッシュ: `mergeCommitOid`
   - 取得元: PRの `mergeCommit.oid`。
   - 取り扱い: マージ済みPRのみ確定値として扱う。

3. 代替取得手段（API側の補強）
   - レビュー開始: REST `pulls/{number}/comments` の最古コメント `original_commit_id` を補助利用可能。
   - マージ時点: REST `pulls/{number}` の `merge_commit_sha` を補助利用可能。

## 現行実装との整合状況

以下3項目は `pr-detail` 実装に反映済み。

- `conversations.reviews[].authorLogin`
- `conversations.reviewThreads[].comments[].authorLogin`
- `conversations.reviewThreads[].comments[].id`

## 注意事項（データ欠落リスク）

- `reviews(first: 100)` / `reviewThreads(first: 100)` / `comments(first: 100)` は上限がある。
- 大規模PRでは取りこぼしが起きるため、目的1/4/5の厳密分析ではページネーション対応を推奨。
