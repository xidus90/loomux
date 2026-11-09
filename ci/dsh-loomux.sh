#!/bin/sh
# The tests of the dsh-loomux npm package; it has no dependencies to install.
set -eu
cd "$(dirname "$0")/../dsh-loomux"
npm test
