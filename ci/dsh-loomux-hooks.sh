#!/bin/sh
# The tests of the dsh-loomux-hooks npm package; it has no dependencies to install.
set -eu
cd "$(dirname "$0")/../dsh-loomux-hooks"
npm test
