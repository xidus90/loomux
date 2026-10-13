# Installing loomux

[Overview](README.md) · [First-time setup](getting-started.md) →

loomux is one Go binary with no Python or Node runtime. Windows is the tested platform; release binaries exist for Linux and macOS as well, but several parts work only on Windows. This page covers what you need, how to install and verify a release binary, how it updates itself, what is different on Linux and macOS, how to build from source, and how to check the version. Once loomux is installed, [First-time setup](getting-started.md) takes it into your project.

## Requirements

- **Windows (x64)** is the tested platform. See [Linux and macOS](#linux-and-macos) for the rest.
- **[GitHub CLI](https://cli.github.com/)** (`gh`), logged in with `gh auth login`. It does the download and, later, the self-update.
- **Git** 2.31 or newer.
- **Go** 1.26 or newer, only for building from source (the module's toolchain is go1.27.0).

## Windows quickstart

Run this in PowerShell. It downloads the newest stable release with `gh`, checks it against `SHA256SUMS` and installs it as `loomux.exe`:

```powershell
$bin = "$env:LOCALAPPDATA\loomux\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
gh release download --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin --clobber
$exe = Get-ChildItem $bin -Filter 'loomux_*_windows_amd64.exe' | Select-Object -First 1
if (-not $exe) { throw 'download failed: see the gh error above' }
$want = ((Select-String -Path "$bin\SHA256SUMS" -Pattern $exe.Name -SimpleMatch).Line -split '\s+')[0]
if ((Get-FileHash $exe.FullName -Algorithm SHA256).Hash -eq $want) { Move-Item -Force $exe.FullName "$bin\loomux.exe"; Remove-Item "$bin\SHA256SUMS" } else { throw "checksum mismatch: $($exe.Name)" }
```

Without a tag, `gh release download` takes the newest stable release. The comparison with `-eq` ignores case, because `Get-FileHash` prints upper case and `SHA256SUMS` holds lower case.

Then run this in any session, in your own project (`cd <project>`):

```powershell
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" status
```

If the binary already stands there, `init` does not download a second one. The full paths are needed because nothing puts the folder on your `PATH`. If you want that, add it to your user `PATH` once:

```powershell
[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ";$env:LOCALAPPDATA\loomux\bin", 'User')
```

This takes effect in a new session. The one-liner writes the user `Path` as a plain string (REG_SZ) and so expands any `%VAR%` entries in it; if you want to keep such entries, add the folder through "Edit environment variables" instead. Next comes [First-time setup](getting-started.md).

## Verifying a download

Download a binary from the [Releases page](https://github.com/xidus90/loomux/releases): `loomux_<version>_<os>_<arch>` for `windows/amd64` (`.exe`), `linux/amd64`, `linux/arm64`, `darwin/amd64` and `darwin/arm64`. Verify it against `SHA256SUMS` from the same release (the Windows quickstart does this for you). On Linux, macOS or in Git Bash:

```sh
sha256sum --check --ignore-missing SHA256SUMS
```

Beside them every release carries `NOTICE.md`, listed in `SHA256SUMS` as well: the licence of every third-party part the binary carries — Go's standard library, each linked module, each tree-sitter grammar it links, and the German word-frequency table under CC BY-SA 4.0 with its sources.

## The machine-wide binary

The MCP bridge and `loomux serve` run from one binary per machine, `%LOCALAPPDATA%\loomux\bin\loomux.exe`, taken from a release and never from a checkout. The quickstart above installs it; point the MCP entry at it:

```powershell
claude mcp add loomux -s user -- "$env:LOCALAPPDATA\loomux\bin\loomux.exe" mcp --channel local
```

`loomux init` takes over the first steps: its part `binary` puts the newest release at the same place when none stands there, and its part `mcp-json` writes the project's `.mcp.json` when the user scope has no server `loomux` (see the [CLI reference](cli-reference.md#11-project-setup-loomux-init)).

## Updates and channels

From then on `serve` keeps the binary current: a minute after it starts and daily after that it takes the highest release of the binary's own channel through `gh`, checks it against `SHA256SUMS` and its `--version`, and swaps the file. The next bridge replaces the running service. `loomux upgrade` does the same by hand. What the last pass found is in `update.json` in the state directory; session start warns when `serve` runs from anywhere else or the pass failed. Windows only for now.

A machine follows the beta channel while the file `channel` in its state directory holds `beta`: it takes every newer release, beta or stable. Without the marker it takes only stable releases. `loomux upgrade --beta` takes the newest release including betas and sets the marker; `--stable` takes the newest stable release, even if it is older than the running beta, and clears it; `--version <x.y.z>` installs exactly that tag, a beta or a downgrade included. A downgrade does not hold: `serve` lifts it within 24 hours. A binary of the version count before 1.0.0 without the marker takes the releases of that count and the stable ones, never a new beta, and ranks below every later release.

## Linux and macOS

Release binaries exist for `linux/amd64`, `linux/arm64`, `darwin/amd64` and `darwin/arm64`, and they are built from the same code. They are not tested there; the tested platform is Windows. Download the one for your system from the [Releases page](https://github.com/xidus90/loomux/releases) and verify it as described above.

Only on Windows:

- the machine-wide binary, and with it the part `binary` of `loomux init`;
- the self-update through `serve` and `loomux upgrade`;
- the worktree mirrors (junctions).

`loomux init` is not meant for Linux or macOS. Its part `binary` fails, and `.mcp.json`, the hook entries and the git hooks are not written but reported as failed, because they would call `${LOCALAPPDATA}/loomux/bin/loomux.exe`. If you use it there anyway, set the module `hooks` to `none` and, in the module `base`, choose `each` and deselect the parts `binary` and `mcp-json`. This works only interactively: `base` has no flag, so `--yes` cannot do it.

What runs there is the CLI itself: `check`, `graph`, `brain` and `serve`.

## Building from source

```sh
git clone https://github.com/xidus90/loomux.git
cd loomux
```

On Windows, for work on loomux itself:

```powershell
go run ./cmd/loomux init --yes   # builds bin/loomux.exe and arms the git hooks
```

On Linux and macOS:

```sh
go build -o loomux ./cmd/loomux
```

`init` is not meant for Linux or macOS (see [Linux and macOS](#linux-and-macos)). `go install github.com/xidus90/loomux/cmd/loomux@latest` does not work, because `go.mod` has a `replace` directive.

## Checking the version

```sh
loomux --version
# Output: loomux 1.0.0 (a stable release), loomux 1.1.0-beta.1 (a beta)
# or loomux 0.0.0-dev (built from source)
```

If the folder is not on your `PATH`, call the binary by its full path: `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" --version`.

[Overview](README.md) · [First-time setup](getting-started.md) →
