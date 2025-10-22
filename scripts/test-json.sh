#!/usr/bin/env bash
# test-json.sh - run go tests with -json output and optional filters, applying quiet flags.
# Usage:
#   ./scripts/test-json.sh                   # all packages (may be noisy)
#   TEST=TestCapabilitySunsetEnforcement ./scripts/test-json.sh ./web
#   PKG=./web TEST=TestCapabilitySunsetEnforcement ./scripts/test-json.sh
# Environment:
#   TEST   - optional regex passed to -run
#   PKG    - package pattern (default ./web)
#   SKIP_WEB_ASSETS=1 recommended for speed when front-end not needed.
#   GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1 applied by default unless QUIET_FLAGS=0.
#   QUIET_FLAGS=0 disables automatic quiet flags.
set -euo pipefail
PKG=${PKG:-./web}
RUN_ARG=""
if [ -n "${TEST:-}" ]; then
  RUN_ARG="-run ${TEST}"
fi
QUIET_ENV=""
if [ "${QUIET_FLAGS:-1}" = "1" ]; then
  QUIET_ENV="SKIP_WEB_ASSETS=1 GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1"
fi
echo "[test-json] package=${PKG} test=${TEST:-<all>} quiet_flags=${QUIET_FLAGS:-1}" >&2
# shellcheck disable=SC2086
eval ${QUIET_ENV} go test -json -count=1 -v ${RUN_ARG} ${PKG} | jq -r 'select(.Action!=null) | (.Time + " " + .Action + " " + (.Test//"<pkg>") + " " + (.Output//""))'
