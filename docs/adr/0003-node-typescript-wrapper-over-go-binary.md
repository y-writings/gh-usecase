# ADR 0003: Provide a Node/TypeScript Wrapper Over the Go Binary

日付: 2026-05-30

ステータス: Proposed

## 文脈

このpackageは、CLIとしてもTypeScript importとしても使える必要がある。
ADR 0001でGo coreをnpm経由で配る方針にしたため、TypeScript import APIは何らかの形でGo実装を呼ぶ必要がある。

候補となるintegration styleは以下である。

- TypeScriptでlogicを再実装する。
- GoをWebAssemblyへcompileする。
- native addon境界でGoを露出する。
- Go CLI binaryをspawnし、JSON stdoutをparseする。

現時点のproductはCLI指向のGitHub情報取得wrapperである。
Go binary内部では `go-gh/v2/pkg/api.GraphQLClient` を使ってGitHub GraphQL APIを直接呼ぶ。
期待される呼び出しはGitHub API待ちが支配的であり、TypeScript wrapperからGo binaryをspawnするoverheadは初期architectureでは許容できる。

## 決定

TypeScript importは、Goバイナリ上のtyped wrapperとして提供する。

wrapperの責務は以下である。

- platform-specific Go binaryを解決する。
- typed input objectをCLI argumentsへ変換する。
- cleanなenvironmentでbinaryをspawnする。
- 成功時はJSON stdoutをparseする。
- non-zero exitまたはinvalid JSONの場合はstructured JavaScript errorをthrowする。
- timeoutとabort signalをsupportする。
- GraphQL query text、GitHub API呼び出し、transformation logic、output shapingをTypeScript側に複製しない。

目標API例:

```ts
import { prDetail, prList, prCount } from "@scope/gh-usecase";

const detail = await prDetail({
  owner: "octokit",
  name: "rest.js",
  number: 1,
});
```

このwrapperはtyped CLI clientである。
browser SDKでもin-process GitHub API clientでもない。

## 結果

良い結果:

- CLIとTypeScript import behaviorが同じGo source of truthを共有する。
- npm利用者は自然なTypeScript interfaceを使える。
- CLI stderrを変えずに、wrapper側でrichなJavaScript errorを提供できる。
- TypeScript側にGraphQL schemaやtransformation logicを並行管理する必要がない。

悪い結果:

- programmatic callでもsubprocess executionが発生する。
- error mappingではstderrとexit codeを保持する必要がある。
- 高頻度呼び出しでは、将来的にbatchingやlong-lived service modeが必要になる可能性がある。
- wrapperはchild process executionとnative binaryに依存するためNode-onlyである。

## Error Contract

wrapperはcommand failureに対して専用error typeをthrowする。
errorには以下を含める。

- `command`: `pr-detail` などのcommand名。
- `args`: sanitized argument list。
- `exitCode`: 取得可能なprocess exit code。
- `stdout`: diagnostic用にcaptureしたstdout。
- `stderr`: diagnostic用にcaptureしたstderr。
- `cause`: process start failureなどのlower-level Node error。

Goバイナリが成功exitしたにもかかわらずstdoutが期待するJSONでない場合、wrapperは別のparse errorをthrowする。
これはGitHub API failureではなく、packaging、compatibility、またはGo CLI output contractのbugを示す。

## Compatibility Rule

TypeScript wrapperを第二のbehavior実装にしない。
command behaviorを変える場合はGo実装を変更し、必要に応じてTypeScript typesとwrapper argument mappingを更新する。
