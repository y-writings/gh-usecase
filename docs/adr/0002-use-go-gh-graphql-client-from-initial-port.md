# ADR 0002: Use go-gh GraphQLClient From the Initial Go Port

日付: 2026-05-30

ステータス: Proposed

## 文脈

現行Bun実装は、GitHub APIを直接呼ばずに `gh api graphql` を外部コマンドとして呼び出している。
この方式は、GitHub CLIの認証・host解決・GraphQL変数渡しに依存できるため、Bun実装では薄いwrapperとして合理的だった。

Goへ移行する目的は、単にBunをGoに置き換えることではない。
npm配布、GitHub Actions `run:` 利用、TypeScript wrapper利用を前提に、runtime依存を減らし、Go内部でGitHub情報取得を完結させることも目的である。

2026-05-30時点では、GitHub CLI互換の認証・host慣習を保ちながらGoからGitHub APIを呼ぶ最適な選択肢として `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` を採用する。

## 決定

初期Go portから `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` を使ってGitHub GraphQL APIを直接呼ぶ。

Go CLIは `gh` executableをspawnしない。
GraphQL queryは現行Bun実装のraw query文字列を維持し、variablesを `map[string]interface{}` として渡す。
実行には `DoWithContext` を使い、timeoutとcancellationを `context.Context` で扱う。

基本形:

```go
client, err := api.DefaultGraphQLClient()
if err != nil {
    return err
}

var response PrDetailGraphQLResponse
err = client.DoWithContext(ctx, query, map[string]interface{}{
    "owner": owner,
    "name": name,
    "number": number,
}, &response)
```

`go-gh/v2/pkg/api` はGitHub CLIと同じ慣習に沿って、`GH_TOKEN`、`GITHUB_TOKEN`、`GH_HOST`、保存済みGitHub CLI credentialなどを扱える。
GitHub Actionsでは `GH_TOKEN: ${{ github.token }}` を渡す運用を標準とする。

## この方式を採用する理由

`gh api graphql` をspawnし続けると、Go binaryからさらに `gh` CLIを呼ぶ二重境界になる。
これは初期移行の差分を小さくする利点はあるが、npm配布されるGo CLIとしてはruntime dependencyが増える。

`go-gh/v2/pkg/api.GraphQLClient` は、以下の条件を同時に満たす。

- `gh` executableをruntime dependencyにしない。
- GitHub CLI互換の認証・host慣習に寄せられる。
- 現行のraw GraphQL query文字列を活かせる。
- `context.Context` でtimeout/cancellationを扱える。
- GraphQL errorとHTTP errorをGoのerrorとして扱える。
- REST APIが必要になった場合も同じ `go-gh/v2/pkg/api` の `RESTClient` を使える。

このprojectでは、GitHub GraphQL queryのshapeと出力変換を維持することが重要である。
Goのstruct-based GraphQL query builderへ寄せるより、raw queryを維持して `DoWithContext` へ渡す方が、現行実装からのbehavioral parityを取りやすい。

## 検討した代替案

### `gh api graphql` をGoからspawnする

現行Bun実装との境界は最も近い。
しかし、Go binaryのruntime dependencyとして `gh` CLIが必要になり、TypeScript wrapperから見ると `Node -> Go binary -> gh CLI -> GitHub API` の二重subprocessになる。
今回のnpm配布・GitHub Actions利用・runtime依存削減の目的には合わない。

### `net/http` でGitHub GraphQL APIを直接呼ぶ

依存は最小になる。
一方で、`GH_TOKEN` / `GITHUB_TOKEN` / `GH_HOST` / GitHub Enterprise / 保存済みGitHub CLI credential / default headers / error mapping を自前で扱う必要がある。
GitHub CLI互換の利用体験を保つには実装負担が大きい。

### `github.com/shurcooL/githubv4` を使う

Go structからGraphQL queryを組み立てる用途では有力である。
ただし、現行実装はすでにraw GraphQL query文字列を持っている。
query表現を同時に変えると、Go移行とGraphQL query再設計が混ざり、parity確認が難しくなる。

### `github.com/google/go-github` を主軸にする

REST API中心なら有力である。
しかし現行commandはGraphQL主体であり、PR詳細取得ではGraphQLのshapeを明示的に制御している。
このprojectの主軸clientとしては `go-gh/v2/pkg/api.GraphQLClient` の方が合う。

## 結果

良い結果:

- npm利用者に `gh` CLIのinstallを要求しない。
- GitHub Actionsでは `GH_TOKEN` を渡すだけで利用しやすい。
- Go binary単体でGitHub GraphQL API通信まで完結する。
- 現行raw GraphQL queryを維持できる。
- `context.Context` によるtimeout/cancellationを実装できる。
- 将来REST補助が必要になっても同じ `go-gh/v2/pkg/api` ecosystemを使える。

悪い結果:

- GitHub CLI processのstderrやexit codeに依存した現行error semanticsは再現しない。
- GraphQL/HTTP error mappingをGo CLIのerror contractとして設計する必要がある。
- 保存済み `gh auth login` credentialに依存する場合は、`go-gh` が読むGitHub CLI configの存在が前提になる。
- `go-gh/v2` のAPI仕様とsecurity advisoryを継続監視する必要がある。

## 実装上の方針

- `api.DefaultGraphQLClient()` または `api.NewGraphQLClient(api.ClientOptions{...})` を使う。
- commandごとにraw query文字列とvariables mapを組み立てる。
- API実行は `DoWithContext` に統一する。
- responseはGo structへdecodeする。
- nullable fieldはpointerで表現する。
- enum-like fieldはcustom typeとvalidationで扱う。
- 成功時stdoutはJSON resultのみを出す。
- 失敗時stderrにはhuman-readable errorを出す。
- GitHub Actions利用例では `GH_TOKEN: ${{ github.token }}` と必要最小限の `permissions` を案内する。

## 非目標

- `gh` executableをspawnするcompatibility layerは作らない。
- GraphQL queryをstruct-based builderへ全面移行しない。
- GitHub API authを完全自前実装しない。
- Go移行と同時にcommand output shapeを変更しない。
