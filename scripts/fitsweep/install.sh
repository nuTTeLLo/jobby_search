#!/usr/bin/env bash
# Installs the fit sweep on this Mac: builds backend/cmd/fitsweep to
# ~/.local/bin/fitsweep, installs run.sh as ~/.local/bin/jobsearch-fitsweep, and
# loads a LaunchAgent that runs it daily at 09:30, half an hour after the
# cluster's 09:00 board scrape. A Mac asleep at 09:30 runs it on waking.
#
# Re-run after pulling changes to rebuild. Needs ~/.config/jobsearch/fitsweep.env
# (see the README next to this script); it is created from a template if missing.
set -euo pipefail

repo="$(cd "$(dirname "$0")/../.." && pwd)"
label="com.adriantan.jobsearch.fitsweep"
plist="$HOME/Library/LaunchAgents/$label.plist"
config="$HOME/.config/jobsearch/fitsweep.env"
logs="$HOME/Library/Logs/JobSearch"

if [ ! -f "$config" ]; then
    mkdir -p "$(dirname "$config")"
    cat > "$config" <<'EOF'
# Settings for the fit sweep (scripts/fitsweep in jobby_search). Fill in, no quotes.
# The account is the one in "My Job Search/.tracker.env".
TRACKER_USER_ID=
TRACKER_EMAIL=
TRACKER_URL=https://job-tracker.tailee323f.ts.net
DISCOVERED_URL=https://job-tracker.tailee323f.ts.net/discovered
# The 1Password Environment behind backend/.env (holds JWT_SECRET).
OP_ENVIRONMENT=
EOF
    chmod 600 "$config"
    echo "Created $config - fill it in, then run this again."
    exit 1
fi

echo "Building fitsweep..."
(cd "$repo/backend" && go build -o "$HOME/.local/bin/fitsweep" ./cmd/fitsweep)
install -m 755 "$repo/scripts/fitsweep/run.sh" "$HOME/.local/bin/jobsearch-fitsweep"
mkdir -p "$logs"

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>$label</string>
	<key>ProgramArguments</key>
	<array>
		<string>$HOME/.local/bin/jobsearch-fitsweep</string>
	</array>
	<key>RunAtLoad</key>
	<false/>
	<key>StandardErrorPath</key>
	<string>$logs/fit_sweep.err.log</string>
	<key>StandardOutPath</key>
	<string>$logs/fit_sweep.out.log</string>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Hour</key>
		<integer>9</integer>
		<key>Minute</key>
		<integer>30</integer>
	</dict>
	<key>WorkingDirectory</key>
	<string>$HOME</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
echo "Installed. Runs daily at 09:30. Try it now with: jobsearch-fitsweep --dry-run"
echo "Logs: $logs/fit_sweep.{out,err}.log"
