#!/bin/sh
# Runs frontend checks in Docker (no Node needed on the host). node_modules
# lives in a named volume (Linux binaries), separate from the host's.
# usage: scripts/test-web.sh ["<command>"]   default: svelte-check + vitest
# e.g.   scripts/test-web.sh "npx vitest run src/lib/projectQuery.test.ts"
set -e
cd "$(dirname "$0")/../web"
MSYS_NO_PATHCONV=1 exec docker run --rm -v "$(pwd -W 2>/dev/null || pwd):/web" -w /web \
  -v projectlist-webnm:/web/node_modules -v projectlist-npm:/root/.npm \
  node:26-alpine sh -c "[ package-lock.json -ot node_modules/.installed ] || { npm ci --no-audit --no-fund >/dev/null && touch node_modules/.installed; }; npx svelte-kit sync && ${1:-npm run check -- --threshold error && npm test}"
