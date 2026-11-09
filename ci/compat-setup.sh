#!/bin/sh
# Installs the runtime and the addons of one compatibility entry:
#   ci/compat-setup.sh <loomux binary> <entry key> <root>
# Godot lands in <root>/godot (and GODOT_BIN is exported to later steps), each
# addon's own folder in <root>/addons/<Name>, Python through uv. Go is set up
# by the workflow, which rebuilds loomux with it when rebuild=true: a Go above
# the toolchain of go.mod is one loomux was never built with. The description is the JSON of `dev compat run --describe`;
# it is flat and written by loomux, so sed reads it without a JSON tool.
set -eu
bin=${1:?usage: ci/compat-setup.sh <loomux binary> <entry key> <root>}
key=${2:?entry key}
root=${3:?root}
mkdir -p "$root"
os=${RUNNER_OS:-Linux}

# PROBE, the JSON of an entry that is not in the list, is described and run
# in place of the key.
if [ -n "${PROBE:-}" ]; then
	printf '%s' "$PROBE" >"$root/probe.json"
	desc=$("$bin" dev compat run --entry-json "$root/probe.json" --describe)
else
	desc=$("$bin" dev compat run "$key" --describe)
fi
field() { printf '%s\n' "$1" | sed -n "s/.*\"$2\":\"\([^\"]*\)\".*/\1/p" | head -n 1; }
runtime=$(field "$desc" runtime)
version=$(field "$desc" version)
install=$(field "$desc" install)

# above_toolchain tells whether major.minor $1 lies above the toolchain of
# go.mod (its go line when it names none).
above_toolchain() {
	mod=$(dirname "$0")/../go.mod
	tc=$(sed -n 's/^toolchain go//p' "$mod")
	[ -n "$tc" ] || tc=$(sed -n 's/^go //p' "$mod")
	awk -v a="$1" -v b="$tc" 'BEGIN {
		split(a, x, "."); split(b, y, ".")
		exit !(x[1] + 0 > y[1] + 0 || (x[1] + 0 == y[1] + 0 && x[2] + 0 > y[2] + 0))
	}'
}
rebuild=false
if [ "$runtime" = go ] && above_toolchain "$version"; then
	rebuild=true
fi
if [ -n "${GITHUB_OUTPUT:-}" ]; then
	printf '%s\n' "runtime=$runtime" "version=$version" "install=$install" "rebuild=$rebuild" >>"$GITHUB_OUTPUT"
fi

unzip_to() {
	if command -v unzip >/dev/null 2>&1; then
		unzip -q -o "$1" -d "$2"
	else
		# Git Bash ships no unzip.
		powershell -NoProfile -Command "Expand-Archive -Force -LiteralPath '$(cygpath -w "$1")' -DestinationPath '$(cygpath -w "$2")'"
	fi
}

# first prints the first of its arguments that exists, so that a glob without
# a match comes out empty.
first() {
	for f in "$@"; do
		if [ -e "$f" ]; then
			echo "$f"
			return
		fi
	done
}

install_godot() {
	# godot-builds tags a pre-release bare ("4.8-beta2"), a release -stable.
	tag=$install
	[ "$install" != "$version" ] || tag=${version}-stable
	base=https://github.com/godotengine/godot-builds/releases/download/$tag
	case $os in
	Windows) asset=Godot_v${tag}_win64.exe.zip ;;
	*) case $version in
		3.*) asset=Godot_v${tag}_linux_headless.64.zip ;;
		*) asset=Godot_v${tag}_linux.x86_64.zip ;;
		esac ;;
	esac
	dl=$root/dl
	mkdir -p "$dl" "$root/godot"
	curl -fsSL --retry 3 -o "$dl/$asset" "$base/$asset"
	curl -fsSL --retry 3 -o "$dl/SHA512-SUMS.txt" "$base/SHA512-SUMS.txt"
	(cd "$dl" && grep -F "  $asset" SHA512-SUMS.txt | sha512sum -c -)
	unzip_to "$dl/$asset" "$root/godot"
	if [ "$os" = Windows ]; then
		# The console build when the zip has one; the plain one otherwise.
		exe=$(first "$root"/godot/*_console.exe)
		[ -n "$exe" ] || exe=$(first "$root"/godot/*.exe)
	else
		exe=$(first "$root"/godot/Godot_v*)
		chmod +x "$exe"
	fi
	[ -n "$exe" ] || { echo "compat-setup: no Godot binary in $root/godot" >&2; exit 1; }
	echo "godot: $exe"
	if [ -n "${GITHUB_ENV:-}" ]; then
		echo "GODOT_BIN=$exe" >>"$GITHUB_ENV"
	fi
}

install_addon() {
	name=$(field "$1" name)
	repo=$(field "$1" repo)
	tag=$(field "$1" tag)
	dir=$(field "$1" dir)
	asset=$(field "$1" asset)
	src=$root/src/$name
	rm -rf "$src" "$root/addons/$name"
	mkdir -p "$root/src" "$root/addons"
	if [ -n "$asset" ]; then
		# The repository holds no built addon, its release does.
		mkdir -p "$src"
		curl -fsSL --retry 3 -o "$root/src/$asset" "$repo/releases/download/$tag/$asset"
		unzip_to "$root/src/$asset" "$src"
	else
		git clone -q --depth 1 --branch "$tag" "$repo" "$src"
	fi
	cp -R "$src/$dir" "$root/addons/$name"
	link_libraries "$root/addons/$name"
	echo "addon: $name $tag"
}

# link_libraries gives each library a .gdextension file of the addon names
# but the addon ships only under a lib prefix a copy under the named file:
# NanoCoverage 0.1.1 builds lib*.so and lib*.dylib and names them without it,
# so Godot on Linux and macOS loads no extension.
link_libraries() {
	for ext in "$1"/*.gdextension; do
		[ -e "$ext" ] || continue
		sed -n 's|^[^#=]*= *"res://.*/\([^/"]*\)"[[:space:]]*$|\1|p' "$ext" | while IFS= read -r lib; do
			for have in "$1"/*/"lib$lib"; do
				want=$(dirname "$have")/$lib
				if [ -e "$have" ] && [ ! -e "$want" ]; then
					cp "$have" "$want"
					echo "addon: $lib copied from lib$lib"
				fi
			done
		done
	done
}

case $runtime in
godot)
	install_godot
	printf '%s\n' "$desc" | grep -o '{"name":[^}]*}' | while IFS= read -r addon; do
		install_addon "$addon"
	done
	;;
python) uv python install "$install" ;;
go) ;;
*)
	echo "compat-setup: no setup for runtime $runtime" >&2
	exit 1
	;;
esac
