# ADR 0001: Go Core With esbuild-Style npm Distribution

日付: 2026-05-30

ステータス: Proposed

## 文脈

`gh-usecase` は現在、Bun/TypeScript で実装された `gh api graphql` の薄いCLIラッパーである。
現行実装の主な責務は、CLI引数の解釈、GitHub CLIの実行、JSONのパース、応答shapeの検証、必要なデータ変換、JSON出力である。
Go版では、外部 `gh` executableへの依存を残さず、`github.com/cli/go-gh/v2/pkg/api.GraphQLClient` でGitHub GraphQL APIを直接呼ぶ。

次のアーキテクチャでは、以下の利用形態を支える必要がある。

- GitHub Actions の `run:` ステップから実行する。
- すでに実装済みの GitHub Actions の内部処理から外部コマンドとして呼ぶ。
- npm経由でインストールまたは実行する。例: `npx @scope/gh-usecase ...`。
- TypeScript から `import { prDetail } from "@scope/gh-usecase"` のようにプログラム利用する。

一方で、`gh-usecase` 自体を `uses: owner/action@v1` の形で使うGitHub Actionとして提供することは目的ではない。
この前提により、最適化対象は JavaScript Action の実行契約ではなく、npmで配布されるCLI/ライブラリとしての利用体験になる。

チームは Go と Bun のどちらにも同程度に対応でき、人員や工数は判断材料ではない。
したがって、この判断は、言語/runtime適性、配布モデル、エコシステムの成熟度、長期運用性に基づいて行う。

## 決定

`esbuild` と同系統の構成を採用する。

- 本番用のcoreとCLIはGoで実装する。
- 配布チャネルはnpmにする。
- platform別のGoバイナリをnpm packageまたはpackage artifactとして配布する。
- Node/TypeScript wrapperを用意し、実行環境に合うバイナリ解決とtyped import APIを提供する。
- GitHub情報取得はGo内部で `go-gh/v2/pkg/api.GraphQLClient` を使って行う。
- npmは配布手段であり、実装をJavaScript/Bunに限定する理由ではないと扱う。
- 移行中も公開CLIとJSON出力契約は安定させる。

目標構成は以下である。

```txt
TypeScript import API または npx CLI
  -> Node wrapper が platform binary を解決
    -> Go CLI が command を実行
      -> go-gh/v2 pkg/api GraphQLClient
        -> GitHub GraphQL API
```

Goバイナリを command behavior の source of truth にする。
TypeScript wrapperはGoバイナリのclient layerであり、GraphQL queryや変換処理の二重実装ではない。

## この決定が利用形態に合う理由

GitHub Actions の `run:` ステップでは、Node/npmと任意のnative binaryを自然に扱える。
Bunは標準の前提ではないため、Bunを使い続けるとworkflow側にBun setupが必要になりやすい。
利用者が `npx @scope/gh-usecase ...` を実行する場合、実装言語がGoであることを意識する必要はなく、GoやBunを追加インストールする必要もない状態が望ましい。

`esbuild` の配布モデルは、この構成が成立する実例である。

- native実装をnpm利用者向けにpackageできる。
- Node wrapperでCLIとprogrammatic APIを提供できる。
- platform-specific packageによりinstall結果を決定的にしやすい。
- native実装の性能と単純なruntime要件を維持できる。

`gh-usecase` において性能は主理由ではない。
主な利点は、配布性、Bun runtimeからの独立、外部 `gh` executableへのruntime依存削減、明確なAPI boundary、長期保守性である。

## 検討した代替案

### Bunのままnpm publishする

現行実装とZod schemaを維持できる。
しかし、standalone executableを別途配らない限り、利用者側にBun runtimeを要求する。
GitHub Actionsのworkflowでは `oven-sh/setup-bun` のようなsetup stepが必要になりやすく、主な利用形態に対して摩擦が大きい。

### GoではなくNode/TypeScriptへ移行する

TypeScript importは最も自然になる。
binary wrapper境界も不要になる。
ただし、今回の主用途は `run:` や既存Action内部からのCLI呼び出しであり、CLI配布・runtime独立性の観点ではGo案が強い。
TypeScript importが主用途に変わるなら再評価対象になる。

### GitHub Actionとしてpublishする

`uses: owner/action@v1` を主導線にするなら有効である。
しかし、ユーザーは `gh-usecase` 自体を `uses:` で使うことを想定していない。
このプロジェクトは、他のworkflowやActionから呼ばれるCLI/ライブラリpackageとして維持する。

### Go binaryのみを配り、TypeScript import APIを提供しない

CLI利用だけなら最も単純である。
しかし、TypeScriptからの `import` 利用要件を満たせない。
安定したnpm programmatic APIを提供するため、TypeScript wrapperは必要である。

## 結果

良い結果:

- 利用者はBunやGoを入れずに `npx` で実行できる。
- GitHub Actionsの通常の `run:` ステップから呼べる。
- 既存GitHub Actionsから外部コマンドとして呼べる。
- CLI behavior のsource of truthがGoバイナリに集約される。
- TypeScript importは薄いtyped wrapperとして提供でき、business logicの二重管理を避けられる。
- `gh` CLIをinstallしていない環境でも、tokenとnetwork accessがあればGitHub情報取得を実行できる。

悪い結果:

- TypeScript import呼び出しはin-process SDK呼び出しではなくsubprocess呼び出しになる。
- wrapper側でbinary resolution、error mapping、stdout parsing、cancellation、timeoutを実装する必要がある。
- platform-specific npm packagingによりrelease設計が複雑になる。
- GitHub API error mappingと認証設定は、Go CLIのerror contractとして明示的に設計する必要がある。

## 非目標

- `gh-usecase` 自体を `uses:` で利用するGitHub Actionにはしない。
- npm利用者にBunを要求しない。
- GraphQL実装をGoとTypeScriptに二重実装しない。
- browser-compatible JavaScript APIは提供しない。
- runtime移行のついでにcommand output semanticsを変えない。

## 後続で決めること

実装計画では、最終的なpackage名、release tooling、binary package layoutを決める。
推奨defaultは、main npm packageとplatform-specific optional dependenciesの組み合わせである。
これはnative npm packageで一般的な構成であり、このADRの意図に合う。
