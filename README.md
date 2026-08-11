```text
██╗  ██╗██╗   ██╗██████╗  █████╗
██║ ██╔╝██║   ██║██╔══██╗██╔══██╗
█████╔╝ ██║   ██║██████╔╝███████║
██╔═██╗ ██║   ██║██╔══██╗██╔══██║
██║  ██╗╚██████╔╝██║  ██║██║  ██║
╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝

                         installable commands from one trusted storehouse
```

Kura is a Go command storehouse for developers who want to discover and install reviewed host tools from one offline, distributable binary.

<!-- BEGIN KIT-MANAGED README BADGES -->
[![Last commit](https://img.shields.io/github/last-commit/jamesonstone/kura)](https://github.com/jamesonstone/kura/commits) [![Open issues](https://img.shields.io/github/issues/jamesonstone/kura)](https://github.com/jamesonstone/kura/issues) [![Pull requests](https://img.shields.io/github/issues-pr/jamesonstone/kura)](https://github.com/jamesonstone/kura/pulls) [![CI](https://github.com/jamesonstone/kura/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesonstone/kura/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/jamesonstone/kura)](https://github.com/jamesonstone/kura/releases)
<!-- END KIT-MANAGED README BADGES -->

## Install Kura

With Go 1.25 or newer:

```sh
go install github.com/jamesonstone/kura/cmd/kura@latest
```

Prebuilt macOS, Linux, and Windows archives are also published on the [GitHub Releases page](https://github.com/jamesonstone/kura/releases).

Make sure the directory containing `kura` is on `PATH`, then start the selector:

```sh
kura
```

Use the arrow keys or `j`/`k` to move, Space to select any number of tools, and Enter to install them. Kura shows every resulting path and warns when the selected executable directory is not on `PATH`.

## Available tools

### `git wt`

Kura includes the safe, project-oriented Git worktree command extracted from Kit. Selecting it installs:

- `git-wt` beside the discoverable Kura executable, or in the user-local bin directory when Kura was launched by path;
- `git-wt.1` in the user-local `man1` directory on macOS and Linux.

Git automatically discovers executables named `git-<command>` on `PATH`, so the installed tool is invoked as:

```sh
git wt help
git wt list
git wt issue 123
git wt sync --dry-run
```

The extracted command preserves Kit's conservative worktree, environment-link, removal, and synchronization behavior. Kura does not remove or alter Kit's current copy.

## Non-interactive use

```sh
kura list
kura install git-wt
kura install --all
kura install --bin-dir "$HOME/bin" --man-dir "$HOME/share/man/man1" git-wt
kura version
```

`KURA_BIN_DIR`, `KURA_MAN_DIR`, and `KURA_STATE_DIR` provide equivalent environment overrides. An existing unowned or locally modified destination is refused. `--force` is the explicit override when replacement is intentional.

## Installation safety

Kura is offline and user-scoped. It does not fetch scripts, elevate privileges, edit shell configuration, or install system-wide files by itself.

Before changing the host, Kura resolves every selected artifact and checks the complete destination set. It records the tool and digest of installed files so future versions can safely update unchanged Kura-owned artifacts. Writes are staged in their destination directories and committed with rollback if an artifact or ownership-state update fails.

## Adding a generic script

The embedded catalog lives at `internal/catalog/assets/catalog.json`. A generic script needs only:

1. the reviewed script beneath `internal/catalog/assets/scripts/`;
2. a catalog tool entry whose artifact uses `source: "embedded"`, `destination: "bin"`, a safe filename, and executable mode `493` (`0755`).

Platform filters can limit an artifact to `darwin`, `linux`, or `windows`. Compiled aliases such as `git-wt` additionally require a dispatcher in the Kura binary; ordinary embedded scripts do not.

## Development

```sh
make check
make test-race
make lint
make release-check
make snapshot
```

The pull-request workflow runs formatting, vet, full and race-enabled tests, lint, source-size enforcement, and a cross-platform GoReleaser snapshot. Tags matching `v*` publish Kura archives and checksums through GoReleaser.

## Maintainers

Maintained with 🪖 and ❤️ by [Jameson](https://github.com/jamesonstone) (`jamesonstone`).
