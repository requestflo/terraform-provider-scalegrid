# Contributing

## Commit messages drive releases

Releases are automated with [semantic-release](https://semantic-release.gitbook.io/)
and determined entirely by commit messages on `main`, following
[Conventional Commits](https://www.conventionalcommits.org/). Every push to
`main` runs the release workflow; the commit types since the last release decide
the next version.

| Type | Example | Effect |
|------|---------|--------|
| `fix:` | `fix: correct MySQL engine field` | patch release (`x.y.Z`) |
| `feat:` | `feat: add scalegrid_user resource` | minor release (`x.Y.0`) |
| `feat!:` or a `BREAKING CHANGE:` footer | `feat!: rename size values` | major release (`X.0.0`) |
| `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `style:` | `docs: clarify auth` | no release |

If a change set contains no `fix:`/`feat:`/breaking commit, no release is cut —
that is expected, not a failure.

## Pull requests

- Branch off `main` and open a PR.
- When merging, use **squash** and make the squash commit title a valid
  Conventional Commit — that title is what lands on `main` and what
  semantic-release reads.
- CI (`go build`, `go vet`, `gofmt`, `go test`) must pass.

## Local development

```sh
make build   # compile the provider
make test    # unit tests
make vet     # go vet
make fmt     # gofmt
```

See the README's "Releasing" section for the full pipeline and one-time
Terraform Registry setup.
