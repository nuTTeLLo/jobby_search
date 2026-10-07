#!/usr/bin/env bash
# Sourced by mise (see [env] in mise.toml) to load backend/.env from the default
# workspace.
#
# That .env may be a 1Password "local env file": a named pipe that 1Password writes
# the secrets into on each read, so they never sit on disk. `mise run dev` loads its
# config in several processes at once, and concurrent readers of one pipe each get
# interleaved fragments ("failed to parse dotenv file"). So read it under a lock, one
# process at a time. A regular .env file works the same way.

env_file="$("$(dirname "${BASH_SOURCE[0]}")/dev-workspace.sh" root)/backend/.env"

if [ -e "$env_file" ]; then
  lock="${TMPDIR:-/tmp}/jobby-dev-env.lock"
  # Give up waiting after ~10s rather than hang on a lock left by a killed process.
  for _ in $(seq 100); do
    mkdir "$lock" 2>/dev/null && break
    sleep 0.1
  done
  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a
  rmdir "$lock" 2>/dev/null
fi
