# Contributing to ferry

Thanks for the interest. This is a small Go CLI and contributions are welcome — bug reports, fixes, docs, and new features all land the same way: through a pull request from a fork.

## Ground rules

- Be kind in issues and reviews.
- Keep PRs focused. One change per PR.
- New behavior gets a test. Bug fixes get a regression test.
- ferry handles secrets — when in doubt, err on the side of safety (refuse, don't overwrite, never log key bytes).

## Development setup

You'll need **Go 1.22+**.

```bash
gh repo fork djleonskennedy/ferry --clone --remote
cd ferry
go test ./...                 # unit + e2e, should pass on a fresh clone
make build                    # builds bin/ferry
```

The Makefile targets you'll use most:

| target | what it does |
|---|---|
| `make build` | binary at `bin/ferry` with version-stamped ldflags |
| `make test` | `go test ./...` |
| `make vet` | `go vet ./...` |
| `make install-local` | builds and drops `ferry` on your `PATH` (see `scripts/install.sh`) |
| `make snapshot` | local cross-platform build via GoReleaser (requires `goreleaser`) |

## Repository layout

```
cmd/ferry/           entry point + cobra commands
internal/            packages: paths, config, scan, crypto, archive,
                     manifest, safety, backup, snapshot, cliutil
test/e2e/            black-box tests that drive the root command
.github/workflows/   CI (test + snapshot build) and Release (tags v*)
.goreleaser.yaml     cross-compile config
scripts/install.sh   prod-style local install + PATH wiring
```

Read `internal/snapshot/create.go` and `internal/snapshot/apply.go` first — they're the heart of the tool.

## The PR workflow

`main` is protected:

- direct pushes are blocked
- a PR is required
- CI (`test` job) must be green
- one approving review is required (admins may bypass for emergencies)

So the flow is always:

1. **Fork** the repo on GitHub.
2. **Branch** off `main` in your fork: `git checkout -b my-fix`.
3. **Code + test** locally. Run `go vet ./... && go test ./...`. Add tests for what you changed.
4. **Commit** with a short imperative subject (and a body if the *why* is non-obvious).
5. **Push** your branch to your fork.
6. **Open a PR** against `djleonskennedy/ferry:main`. Describe what changed and why.
7. **CI runs**. Fix anything red.
8. **Review**. Address feedback by pushing more commits to the same branch — stale approvals are dismissed automatically on new pushes.
9. **Merge**. A maintainer merges (squash by default).

If your change is large or speculative, open an issue first to discuss the approach before writing the code. Saves time on both sides.

## Commit messages

Short imperative subject line, ≤ 70 chars. Body optional but appreciated for the *why*.

```
Refuse plaintext snapshot when encryption.required = true

Previously --plain silently produced an unencrypted tarball even when
the project config required encryption. Now snapshot returns ErrAbort
and exits 3.
```

No required prefix (no `feat:` / `fix:` conventional-commits enforcement, at least for now).

## Code style

- `gofmt` / `gofumpt` clean. Run `go vet ./...` before pushing — CI will too.
- Default to no comments. Add one only when the *why* is non-obvious (a hidden constraint, a subtle invariant, a workaround for a specific bug).
- New errors that affect the CLI exit code go through `internal/cliutil`'s typed errors.
- New filesystem locations go through `internal/paths` — never hardcode `~/.ferry` anywhere else.
- Never log raw key bytes. Don't print decrypted file contents to stdout.

## Reporting bugs

Open an issue at <https://github.com/djleonskennedy/ferry/issues> with:

- ferry version (`ferry --version`)
- OS + arch
- exact command you ran
- what you expected, what happened
- minimal reproduction if you can

If the bug involves data loss or could leak secrets, mark the issue accordingly or contact the maintainer privately before publishing details.

## Security disclosure

If you find a vulnerability — especially one that could leak plaintext secrets, weaken encryption, or trick `apply` into overwriting files outside the project root — please **do not** open a public issue. Email the maintainer (see the author info in commit metadata) with the details, a proof of concept if you have one, and a suggested fix if you have one.

## License

By contributing you agree your work is released under the project's [MIT License](LICENSE).
