# Goリアーキテクチャ引き継ぎ書

日付: 2026-05-30

## 目的

この文書は、`gh-usecase` をBun実装からGo実装へ移行するための引き継ぎ書である。
元の議論を知らないエンジニアやAI agentが、なぜGoへ移行するのか、何を維持すべきか、npm/TypeScript wrapperをどう設計すべきか、GitHub情報取得をGoでどう実装すべきかを理解できることを目的とする。

この文書は実装計画そのものではない。
実装計画を作る前提資料であり、設計判断、境界、移行順序、検証条件を固定する。

## 結論

`gh-usecase` は、`esbuild` と同系統の構成へ移行する。

```txt
利用者
  -> npm / npx / TypeScript import
    -> Node/TypeScript wrapper
      -> platform別Go binary
        -> go-gh/v2 pkg/api GraphQLClient
          -> GitHub GraphQL API
```

重要な判断は2つある。

- npmを「実装runtime」ではなく「配布チャネル」と扱う。
- Go版のGitHub情報取得は `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` で最初から直接行う。

利用者はGitHub Actionsの `run:`、既存GitHub Actionsの内部処理、ローカルshell、Node/TypeScript codeからこのtoolを使えるようにする。
利用者にBunやGoのインストールを要求しない。
Go版では外部 `gh` executableのinstallも必須にしない。

## 現行リポジトリの状態

確認時点の実装状態は以下である。

- RuntimeはBun。
- `package.json` のpackage名は `gh-pr-counter`。
- `package.json` の `bin` は `gh-pr-counter: src/index.ts`。
- command entrypointは `src/index.ts`。
- command実装は `src/cli/pr-count`、`src/cli/pr-list`、`src/cli/pr-detail` にある。
- 共通のGraphQL実行境界は `src/core/command-runner.ts`。
- `runGraphQlCommand` は `gh api graphql -f query=... -F key=value` を組み立て、`Bun.spawnSync` で実行する。
- JSON parseとschema validationは `src/core/response-parser.ts` とZod schemaで行う。
- CLI argument parsingは `src/core/cli-args.ts` のcustom parserで行う。
- CLI設計標準は `docs/cli-design-standards.md` にある。
- `pr-detail` のfield rationaleは `docs/pr-detail-field-rationale.md` にある。

直近commitは小さく、現在のコードベースはまだ移行前提で大きく分岐していない。

- `feat: add GitHub CLI TypeScript wrapper (#1)`
- `initial commit`

## 議論の経緯

最初の前提では、`gh-usecase` はBunで作られたGitHub情報取得wrapperだった。
Goで書き直す案について、チームはGo/Bunどちらにも同等に対応できるため、人員や工数ではなく技術適性で比較した。

純粋な薄いGraphQL wrapperとして見ると、Bun/TypeScript/Zodは非常に適している。
JSON処理、Zodによるruntime validation、TypeScriptのobject transformが簡潔であり、現行実装との親和性も高い。

一方で、利用形態がより具体化した。

- npmで配布したい。
- GitHub Actionsの `run:` から使いたい。
- 既存GitHub Actionsの中から外部commandとして使いたい。
- `gh-usecase` 自体を `uses:` で使うGitHub Actionにはしない。
- TypeScriptから `import` してprogrammaticに使う導線も欲しい。
- Go版ではGitHub情報取得もGo内部で完結させたい。

この前提では、Go実装をnpmで配る構成が強くなる。
代表的な前例として `esbuild` がある。
`esbuild` はGo実装をnative binaryとして持ち、npm packageからCLI/APIを提供する。

GitHub情報取得については、2026-05-30時点で `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` を採用する。
理由は、GitHub CLI互換の認証・host慣習に寄せながら、外部 `gh` processをspawnせずにGoからGraphQL APIを呼べるためである。

## ADR一覧

詳細な意思決定は `docs/adr/` を参照する。

- `docs/adr/0001-go-core-esbuild-style-npm-distribution.md`
- `docs/adr/0002-use-go-gh-graphql-client-from-initial-port.md`
- `docs/adr/0003-node-typescript-wrapper-over-go-binary.md`

ADRは以下の3つの論点を分けている。

- Go coreとnpm配布モデル。
- Go版のGitHub情報取得に `go-gh/v2/pkg/api.GraphQLClient` を使うこと。
- TypeScript import APIをGo binary wrapperとして提供すること。

これらを分ける理由は、将来1つのlayerだけを差し替えられるようにするためである。
たとえば将来TypeScript wrapperのAPIを増やしても、GoのGraphQL API client境界は維持できる。

## 目標アーキテクチャ

目標は4層構造である。

```txt
Consumer surface
  CLI: npx @scope/gh-usecase pr-detail --owner ... --name ... --number ...
  TS:  import { prDetail } from "@scope/gh-usecase"

Node/npm wrapper
  platform binaryを解決する
  TypeScript input objectをCLI argsへ変換する
  TypeScript callではstdout JSONをparseする
  structured JS errorを返す

Go binary
  command registry、input validation、GraphQL API call、JSON decode、output transformを持つ
  成功時はJSONをstdoutへ出す
  失敗時はhuman-readable errorをstderrへ出す

GitHub API boundary
  go-gh/v2/pkg/api.GraphQLClientでGitHub GraphQL APIを呼ぶ
  GH_TOKEN / GITHUB_TOKEN / GH_HOST / GitHub CLI config慣習をgo-ghに委譲する
```

Go binaryがbehaviorのsource of truthである。
TypeScript packageはGraphQL query、GitHub API call、変換処理を複製しない。

## 利用者シナリオ

### GitHub Actions run step

想定workflow:

```yaml
permissions:
  contents: read
  pull-requests: read

steps:
  - name: Fetch PR detail
    run: npx -y @scope/gh-usecase pr-detail --owner octokit --name rest.js --number 1
    env:
      GH_TOKEN: ${{ github.token }}
```

この利用形態では、`oven-sh/setup-bun` を要求しない。
`actions/setup-go` も要求しない。
Go版では外部 `gh` CLIのinstallも要求しない。

### 既存GitHub Actionの内部から呼ぶ

すでに実装済みのActionが、shellやNode processからこのtoolを外部commandとして呼ぶ。
そのActionは `npx`、package dependency、またはTypeScript wrapper経由でGo binaryを実行できる。

このプロジェクト自体は `uses:` 用Actionとして設計しない。

### TypeScript import

想定API:

```ts
import { prDetail } from "@scope/gh-usecase";

const detail = await prDetail({
  owner: "octokit",
  name: "rest.js",
  number: 1,
});
```

このcallは内部的にGo binaryをspawnし、stdout JSONをparseする。
Go binary内部では `go-gh/v2/pkg/api.GraphQLClient` がGitHub GraphQL APIを呼ぶ。
TypeScript APIはNode専用としてdocumentする。
browser SDKとして見せない。

## Go版GitHub API client方針

Go版のGitHub情報取得には `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` を使う。

基本方針:

- `api.DefaultGraphQLClient()` または `api.NewGraphQLClient(api.ClientOptions{...})` を使う。
- GraphQL queryは現行Bun実装のraw query文字列を維持する。
- variablesは `map[string]interface{}` で渡す。
- API実行は `DoWithContext` に統一する。
- timeout/cancellationは `context.Context` で扱う。
- responseはGo structへdecodeする。
- GitHub Actionsでは `GH_TOKEN: ${{ github.token }}` を標準認証方式にする。

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

`go-gh/v2/pkg/api.GraphQLClient` を選ぶ理由:

- external `gh` processをspawnしない。
- GitHub CLI互換の認証・host慣習に寄せられる。
- 現行raw GraphQL queryを活かせる。
- `context.Context` でtimeout/cancellationを扱える。
- GraphQL errorとHTTP errorをGoのerrorとして扱える。
- REST補助が必要な場合も同じ `go-gh/v2/pkg/api` の `RESTClient` を使える。

採用しない方針:

- `gh` executableをspawnするcompatibility layerは作らない。
- `net/http` でGitHub auth/host/error handlingを完全自前実装しない。
- 初期移行で `github.com/shurcooL/githubv4` のstruct-based queryへ全面移行しない。
- REST中心の `github.com/google/go-github` を主軸にしない。

## 初期Go portのscope

初期Go portでは現行Bun commandとのbehavioral parityを最優先する。
ただし、GitHub API execution boundaryは `gh` processではなく `go-gh/v2/pkg/api.GraphQLClient` に置き換える。

対象command:

- `pr-count`
- `pr-list`
- `pr-detail`

維持するcommand contract:

- list-style commandは1回のcallで1 pageのみ返す。
- paginationは `after` と `first` によるuser-driven modelを維持する。
- filteringはGitHub GraphQLが直接supportするserver-side filteringのみ許可する。
- hidden aggregation loopを導入しない。
- output shapeは別途breaking change判断がない限り維持する。
- GraphQL executionはGo内部のGitHub API client boundaryに集約する。

`docs/cli-design-standards.md` はGo移行後も有効な設計基準として扱う。
Zod固有の記述は、Goでは以下の意図として読み替える。

- validationは素直に書く。
- magic numberはnamed constantにする。
- command registryはsingle sourceにする。
- parser boundaryを明確にする。
- command moduleはquery/filter assemblyとoutput shapingへ集中させる。

## 推奨Go package boundary

正確なfile layoutは実装計画で確定する。
ただし責務境界は以下を推奨する。

```txt
cmd/gh-usecase
  main package
  argv dispatch
  stdout/stderr/exit code handling

internal/cli
  command registry
  argument parsing
  usage text
  command-level input validation

internal/githubapi
  go-gh GraphQLClient construction
  context timeout/cancellation
  GraphQL execution
  GraphQL/HTTP error mapping

internal/graphql
  query strings
  variable assembly
  response structs shared by command packages where appropriate

internal/prcount
  pr-count command behavior

internal/prlist
  pr-list command behavior

internal/prdetail
  pr-detail command behavior
  output transformation
  generated/binary file exclusion heuristics

internal/jsonutil
  strict JSON/output helpers
  JSON stdout encoding helpers
```

実装では、premature abstractionよりも読みやすい責務境界を優先する。
plugin system、generic command framework、compatibility shimは、具体的必要性が出るまで導入しない。

## Go validation方針

現行Bun実装はZodでruntime validationを行っている。
Go移行で外部境界のvalidationを弱めてはいけない。

推奨方針:

- 既知のGraphQL responseはtyped structへdecodeする。
- nullable GraphQL fieldはpointerで表現する。
- enum-like fieldはcustom string typeとvalidation関数で表現する。
- API実行前にcommand inputをvalidateする。
- 可能であれば出力前にtransformed outputもvalidateする。
- GraphQL responseに `errors` がある場合はGo CLIのerror contractへ明示的にmapする。

重要な差分:

Goの `encoding/json` はunknown fieldをdefaultで無視する。
`go-gh` のdecode挙動と組み合わせて、現行Zod schemaの `.strict()` 相当が本当に必要な境界は別途testで固定する。
ただしGitHub GraphQL APIはqueryで要求したfieldのみ返すため、unknown field拒否は主にfixture decodeやoutput validation側で扱う。

## TypeScript wrapper contract

TypeScript wrapperはGo commandと1対1に対応するtyped functionを提供する。

推奨public API:

```ts
export type PrCountInput = {
  owner: string;
  name: string;
  state?: "OPEN" | "CLOSED" | "MERGED";
};

export type PrListInput = {
  owner: string;
  name: string;
  state?: "OPEN" | "CLOSED" | "MERGED";
  after?: string;
  first?: number;
};

export type PrDetailInput = {
  owner: string;
  name: string;
  number: number;
  filesFirst?: number;
};

export type RunOptions = {
  env?: NodeJS.ProcessEnv;
  cwd?: string;
  signal?: AbortSignal;
  timeoutMs?: number;
};

export function prCount(input: PrCountInput, options?: RunOptions): Promise<PrCountOutput>;
export function prList(input: PrListInput, options?: RunOptions): Promise<PrListOutput>;
export function prDetail(input: PrDetailInput, options?: RunOptions): Promise<PrDetailOutput>;
```

low-level runnerは具体的需要が出た場合のみ公開する。
公開する場合も、Go binaryをsource of truthとして扱い、TypeScript側でbehaviorを再実装しない。

## TypeScript wrapper error model

wrapperはstructured errorをthrowする。

推奨error type:

- `GhUsecaseCommandError`: Go binaryがnon-zero exitした場合。
- `GhUsecaseBinaryNotFoundError`: platform binaryが見つからない、またはunsupported platformの場合。
- `GhUsecaseOutputParseError`: command成功後のstdoutがvalid JSONでない場合。
- `GhUsecaseTimeoutError`: wrapper側timeoutが発生した場合。

`GhUsecaseCommandError` には以下を含める。

- `command`
- `args`
- `exitCode`
- `stdout`
- `stderr`
- `cause`

wrapperはhuman-readable stderrをparseしてbehavior判断しない。
exit codeとstdout JSON contractを判断軸にする。

## CLI error model

Go CLIは単純で機械処理しやすいcontractを維持する。

- 成功時、stdoutにはJSON resultのみを出す。
- 成功時、exit codeは `0`。
- 失敗時、stderrにはhuman-readable error messageを出す。
- 失敗時、exit codeはnon-zero。
- machine-readable structured errorは、必要になった場合に `--error-format json` のような明示optionとして後から追加する。

stdoutをJSON専用に保つことは重要である。
TypeScript wrapperがstdoutをparseするため、成功時stdoutにlogを混ぜてはいけない。

## runtime dependencies

初期architectureのruntime dependencies:

- npm packageが提供するplatform-specific `gh-usecase` Go binary。
- npm package executionとTypeScript wrapper利用のためのNode.js。
- GitHub APIへ到達できるnetwork access。
- `GH_TOKEN`、`GITHUB_TOKEN`、または `go-gh` が参照できるGitHub CLI config上のcredential。

GitHub Actionsでは以下を渡す。

```yaml
env:
  GH_TOKEN: ${{ github.token }}
```

`--owner` と `--name` を明示するcommandでは、基本的に `actions/checkout` は不要である。
このtoolはlocal repository stateではなく、GitHub APIのremote service stateを読む。

## npm distribution model

推奨配布モデル:

- main package: `@scope/gh-usecase`。
- platform package: `@scope/gh-usecase-linux-x64`、`@scope/gh-usecase-darwin-arm64`、`@scope/gh-usecase-win32-x64` など。
- main packageはoptional dependenciesとしてplatform packageを持つ。
- main packageがruntimeで現在のplatformに合うbinaryをresolveする。
- main packageはCLI binとTypeScript import APIを提供する。

これは `esbuild` などのnative npm packageで一般的な構成である。

default designではinstall-time binary downloadを避ける。
install-time downloadはcorporate proxy、offline install、locked-down CIで詰まりやすい。

全platform binaryを1 packageへ同梱する案は単純だが、package sizeが大きくなる。
明示的にpackage sizeを許容する判断がない限り、platform optional dependenciesを優先する。

## build and release expectations

実装後は、利用者が使うplatform向けにbinaryを生成する。

最低推奨platform:

- Linux x64
- Linux arm64
- macOS x64
- macOS arm64

外部npm利用者を想定するならWindowsも含める。
社内利用でLinux/macOS runnerに限定されるなら、Windows対応は明示判断のうえで後回しにできる。

release automationでは以下を確認する。

- Go testsが通る。
- TypeScript wrapper testsが通る。
- 各platform package artifactのCLIが `--help` を実行できる。
- main npm packageが現在platformのbinaryを解決できる。
- package contentsにpublish不要なsource fileやsecretが含まれていない。

## migration strategy

推奨移行順序:

1. 現行Bun behaviorをtestまたはgolden fixtureで固定する。
2. `go-gh/v2/pkg/api.GraphQLClient` を使うGitHub API boundaryをGoで実装する。
3. `pr-count` をportし、JSON parityを確認する。
4. `pr-list` をportし、pagination contract parityを確認する。
5. `pr-detail` をportし、field transformation parityを確認する。
6. local Go binaryをbuildし、manual callで動作確認する。
7. local binaryをresolveして実行するnpm wrapper CLIを追加する。
8. Go binary上のTypeScript import APIを追加する。
9. platform-specific package layoutを追加する。
10. release automationを追加する。
11. npm/Go parityが証明された後にBun entrypointを削除またはdeprecatedにする。

移行開始時点でBun実装を削除しない。
Go parityが証明されるまではBun実装をreferenceとして残す。

## verification requirements

Go port完了を主張する前に、以下を確認する。

- `pr-count` が同じinputを受け、同じoutput shapeを返す。
- `pr-list` が同じinputを受け、同じoutput shapeを返す。
- `pr-detail` が同じinputを受け、同じoutput shapeを返す。
- GraphQL API errorがGo CLIのnon-zero exitになる。
- HTTP/auth/rate-limit系errorがGo CLI error messageに反映される。
- 成功時stdoutがvalid JSONであり、logを含まない。
- TypeScript wrapperが成功stdoutをparseできる。
- TypeScript wrapperがCLI failure時にstructured errorをthrowする。
- GitHub Actionsで `GH_TOKEN: ${{ github.token }}` を渡して動く。
- npm利用者にBun runtimeを要求しない。
- npm利用者に外部 `gh` executableを要求しない。

## testing recommendations

testはlayerごとに分ける。

- Goのargument parsingとvalidationをunit testする。
- GoのGraphQL variables constructionをunit testする。
- Goのoutput transformationをfixture JSONでunit testする。
- `go-gh` clientのtransportを差し替えられる境界を用意し、HTTP fixtureでGraphQL success/errorをtestする。
- auth token解決はunit testでは明示tokenを使い、GitHub CLI configへの依存を避ける。
- TypeScript wrapperはfake binaryまたはfixture binaryでspawn、parse、timeout、error behaviorをtestする。
- real GitHub APIを叩くend-to-end testは、networkとcredentialが必要なためoptionalまたはmanual gateにする。

HTTP fixtureによるGraphQL boundary testは重要である。
GitHubへの実通信や本物のcredentialなしに、API success、GraphQL errors、HTTP errors、timeoutを決定的に検証できる。

## compatibility rules

移行では、明示的なbreaking changeとして記録しない限り公開behaviorを変えない。

維持対象:

- command name。
- flag name。
- default value。
- JSON output property name。
- nullability semantics。
- pagination behavior。
- filtering behavior。
- exit code convention。

package名やbinary名を改善したい場合でも、npm package namingとCLI command compatibilityは分けて扱う。
現行package名は `gh-pr-counter` だが、議論上のproject名は `gh-usecase` である。
publish前に最終的なnpm package名とbinary名を実装計画で確定する。

## risks

### TypeScript wrapper subprocess

TypeScript callはGo binaryをspawnする。
初期architectureでは許容する。
理由は、支配的なcostがGitHub API latencyであり、process startup costが主問題になりにくいためである。

高頻度local usageが重要になった場合は、batchingまたはlong-lived service modeを後続検討する。

### go-gh auth/config assumptions

GitHub Actionsでは `GH_TOKEN` を明示して使うため、GitHub CLI configに依存しない。
ローカル利用で保存済みcredentialを使う場合は、`go-gh` が参照できるGitHub CLI configが存在する必要がある。
READMEでは、CIでは `GH_TOKEN` を明示すること、ローカルでは `GH_TOKEN` または既存GitHub CLI auth configを使えることを説明する。

### GoとTypeScriptのtype drift

Go structsとTypeScript typesはdriftする可能性がある。
対策は、JSON schemaからTypeScript typesを生成する、shared fixturesを持つ、Go-produced outputをTypeScript wrapper testでparseする、などである。
初期実装では、十分なoutput fixtureがあるならhandwritten TypeScript typesでもよい。

### native npm package complexity

platform optional dependenciesによりrelease complexityが増える。
それでも、Bun runtime要求やinstall-time binary downloadよりはdefaultとして望ましい。

### go-gh dependency lifecycle

`go-gh/v2` はGitHub CLI ecosystemに沿った有力な選択肢だが、依存moduleとしてversion更新とsecurity advisory監視が必要である。
`go list -m -u all`、Dependabot、release note確認などで継続管理する。

## future API option

将来、`go-gh/v2/pkg/api.GraphQLClient` では足りない要件が出た場合、以下を再評価する。

- `go-gh/v2/pkg/api.ClientOptions` の明示設定で解決できるか。
- `net/http` による完全自前GraphQL clientが必要か。
- GitHub App installation tokenなど、`GH_TOKEN` 以外の認証が必要か。
- GitHub Enterprise hostやcustom API endpoint要件が増えるか。
- rate limitやretry policyを独自実装する必要があるか。

ただし初期Go portでは `go-gh/v2/pkg/api.GraphQLClient` を採用し、別clientへの抽象化を過剰に作らない。

## future agents向けguardrails

この文書から実装を始めるagentは、以下を守る。

- command behavior変更前に `docs/cli-design-standards.md` を読む。
- `pr-detail` field変更前に `docs/pr-detail-field-rationale.md` を読む。
- Goをbehavioral source of truthにする。
- TypeScript wrapperは薄く保つ。
- Go版GitHub情報取得は `go-gh/v2/pkg/api.GraphQLClient` で行う。
- 外部 `gh` executableをruntime dependencyに戻さない。
- npm利用者向けruntime dependencyにBunを追加しない。
- このprojectを `uses:` GitHub Actionにしない。
- hidden paginationやclient-side filteringを導入しない。
- testまたはfixture comparisonで証明するまでparityを主張しない。
