# Nix Flake Design

## Goal

Add a minimal Nix flake for building and running the `gh-usecase` Go CLI, using `y-writings/xapi-usecase` as the reference implementation.

## Scope

In scope:

- Add `flake.nix`.
- Expose the CLI through `packages.default` and `apps.default`.
- Support `x86_64-linux`, `aarch64-linux`, `x86_64-darwin`, and `aarch64-darwin`.
- Build the Go command at `./cmd/gh-usecase` with `pkgs.buildGoModule`.

Out of scope:

- `devShells`.
- `checks`.
- GitHub Actions changes.
- Release automation.

## Architecture

The flake will mirror the reference `xapi-usecase` structure:

- Input: `nixpkgs` from `github:NixOS/nixpkgs/nixos-unstable`.
- System helper: a local `forEachSystem` using `nixpkgs.lib.genAttrs`.
- Package: `pkgs.buildGoModule` with `pname = "gh-usecase"` and `version = "0.0.0"`.
- App: a flake app pointing to the package binary at `bin/gh-usecase`.

The package source will be limited to files required for the Go build: `go.mod`, `go.sum`, `cmd`, and `internal`. This keeps the Nix source clean without implying extra package inputs.

## Build Behavior

The package will set:

- `subPackages = [ "cmd/gh-usecase" ]`.
- `vendorHash` set to the fixed-output hash for the module dependency tree, because `gh-usecase` has external Go dependencies in `go.sum`. This intentionally differs from the reference repository, which can use `vendorHash = null` because it has no external module dependencies.
- `ldflags = [ "-s" "-w" ]`, matching the reference and the Docker build's stripped binary intent.

Package metadata will describe the CLI and set `mainProgram = "gh-usecase"`.

## Testing

Verification should run at least:

```sh
nix build .
```

If available and not blocked by environment constraints, also run:

```sh
nix run . -- --help
```

No test-first production-code loop is required because this change adds packaging configuration rather than application behavior.
