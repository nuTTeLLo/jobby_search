#!/usr/bin/env bash
# Dev-environment values for mise.toml, so `mise run dev` works from any jj workspace.
#
#   root    the default workspace's root. It holds the gitignored state (backend/.env,
#           .venv) and the jobspy-mcp-server submodule that a secondary workspace lacks.
#   offset  this workspace's dev port offset: 0 in the default workspace, otherwise the
#           smallest offset no other live workspace holds, remembered in .dev-port-offset.
#
# mise runs this on every config load, so the common path never invokes jj.
set -euo pipefail

here=$(cd "$(dirname "$0")/.." && pwd)

# Only a secondary workspace has .jj/repo as a file: the path to the default one's.
if [ ! -f "$here/.jj/repo" ]; then
  case "${1:-}" in
    root) echo "$here" ;;
    offset) echo 0 ;;
    *) echo "usage: $0 root|offset" >&2; exit 2 ;;
  esac
  exit
fi

case "${1:-}" in
  root)
    cd "$here/.jj" && cd "$(cat repo)/../.." && pwd
    ;;
  offset)
    if [ ! -f "$here/.dev-port-offset" ]; then
      # Workspaces first loading mise at the same moment must not pick the same
      # offset, so allocate under a lock in the shared .jj/repo.
      lock=$(cd "$here/.jj" && cd "$(cat repo)" && pwd)/dev-port-offset.lock
      until mkdir "$lock" 2>/dev/null; do sleep 0.1; done
      trap 'rmdir "$lock"' EXIT
      if [ ! -f "$here/.dev-port-offset" ]; then
        used=" "
        for name in $(jj -R "$here" --ignore-working-copy workspace list -T 'name ++ "\n"'); do
          other=$(jj -R "$here" --ignore-working-copy workspace root --name "$name")
          if [ -f "$other/.dev-port-offset" ]; then
            used+="$(cat "$other/.dev-port-offset") "
          fi
        done
        offset=1
        while [[ "$used" == *" $offset "* ]]; do offset=$((offset + 1)); done
        echo "$offset" > "$here/.dev-port-offset"
      fi
    fi
    cat "$here/.dev-port-offset"
    ;;
  *) echo "usage: $0 root|offset" >&2; exit 2 ;;
esac
