#!/usr/bin/env bash
# Sourced by mise (see [env] in mise.toml) to load backend/.env from the default
# workspace.
#
# That .env may be a 1Password "local env file": a named pipe that 1Password writes
# the secrets into on each read, so they never sit on disk. `mise run dev` loads its
# config in several processes at once, and concurrent readers of one pipe each get
# interleaved fragments ("failed to parse dotenv file"). So read it under a lock, one
# process at a time. A regular .env file works the same way.
#
# The lock is a kernel file lock held by the reading `cat` (lockf on macOS, flock on
# Linux): it is released when that process exits, even if it is killed, so there is
# never a stale lock to clean up. The contents go through a shell variable, not disk.

env_file="$("$(dirname "${BASH_SOURCE[0]}")/dev-workspace.sh" root)/backend/.env"

if [ -e "$env_file" ]; then
  lock="${TMPDIR:-/tmp}/jobby-dev-env.lock"
  if command -v lockf >/dev/null; then
    contents=$(lockf -k -t 30 "$lock" cat "$env_file")
  else
    contents=$(flock -w 30 "$lock" cat "$env_file")
  fi || {
    echo "dev-env.sh: could not read $env_file under lock $lock" >&2
    return 1
  }
  set -a
  # shellcheck disable=SC1090
  . <(printf '%s\n' "$contents")
  set +a
  unset contents
fi
