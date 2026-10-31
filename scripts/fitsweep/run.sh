#!/usr/bin/env bash
# Runs the fit sweep (backend/cmd/fitsweep) unattended, from launchd.
#
# Loads the account and Telegram settings from ~/.config/jobsearch, then gets
# JWT_SECRET from the 1Password Environment behind backend/.env through the
# JOBSEARCH_OP_TOKEN service account, the same route the rejection sweeps use.
# backend/.env itself is a 1Password pipe that is empty while the app is locked,
# so it is never read here. Nothing here touches the OneDrive mount, which keeps
# the job clear of the macOS network-volume prompt.
#
# Installed as ~/.local/bin/jobsearch-fitsweep by install.sh. Extra arguments go
# to the sweep, e.g. `jobsearch-fitsweep --dry-run`.
set -euo pipefail

export PATH="/etc/profiles/per-user/$USER/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin"

set -a
# TRACKER_USER_ID, TRACKER_EMAIL, DISCOVERED_URL, OP_ENVIRONMENT
. "$HOME/.config/jobsearch/fitsweep.env"
# TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID (blank means no summary)
. "$HOME/.config/jobsearch/telegram.env"
set +a

# fnox loads ~/.config/fnox/config.toml (its global config) from any directory.
sa_token="$(fnox get JOBSEARCH_OP_TOKEN)"
# The Environment's variables exist only inside the child; it hands back
# JWT_SECRET alone.
JWT_SECRET="$(OP_SERVICE_ACCOUNT_TOKEN="$sa_token" "$HOME/.local/bin/op-beta" run \
    --environment "$OP_ENVIRONMENT" --no-masking -- /bin/sh -c 'printf %s "$JWT_SECRET"')"
unset sa_token
export JWT_SECRET

echo "=== fit sweep $(date '+%Y-%m-%d %H:%M %Z')"
exec "$HOME/.local/bin/fitsweep" "$@"
