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
  got_lock=
  for _ in $(seq 100); do
    mkdir "$lock" 2>/dev/null && { got_lock=1; break; }
    sleep 0.1
  done
  # A read takes well under a second, so a lock still held after ~10s was left by
  # a killed process: take it over rather than fail on every later load.
  if [ -z "$got_lock" ]; then
    rmdir "$lock" 2>/dev/null
    if ! mkdir "$lock" 2>/dev/null; then
      echo "dev-env.sh: could not lock $lock; not reading $env_file" >&2
      return 1
    fi
  fi
  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a
  rmdir "$lock" 2>/dev/null
fi
