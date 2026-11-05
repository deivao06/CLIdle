#!/bin/bash
# usage: run-model.sh <model> [hours] [seeds]
# <model> is A-exp (commit $BASE) or a patch name in docs/balance-audit/models (A-quad, B35, E13, ...).
# Creates a git worktree at $WORK/wt-<model>, applies the patch, runs the 4 classes in parallel
# and writes CSVs to $WORK/sim-out/<model>. WORK defaults to /tmp/clidle-audit.
set -e
BASE=${BASE:-2d21a25}
M=$1; H=${2:-48}; S=${3:-1,2,3}
REPO=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
WORK=${WORK:-/tmp/clidle-audit}
WT=$WORK/wt-$M; OUT=$WORK/sim-out/$M
mkdir -p "$OUT"
if [ ! -d "$WT" ]; then
  git -C "$REPO" worktree add --detach "$WT" "$BASE"
  [ "$M" = A-exp ] || git -C "$WT" apply "$REPO/docs/balance-audit/models/$M.patch"
  cp "$REPO/internal/game/balance_audit_test.go" "$WT/internal/game/"
fi
(cd "$WT" && go test -c -o "$OUT/audit.test" ./internal/game/)
for c in 0 1 2 3; do
  (cd "$WT/internal/game" && BALANCE_AUDIT=1 AUDIT_LABEL=$M AUDIT_HOURS=$H AUDIT_SEEDS=$S AUDIT_CLASSES=$c AUDIT_OUT=$OUT \
    timeout 900 "$OUT/audit.test" -test.run TestBalanceAudit -test.count=1 -test.timeout 0 > "$OUT/log_c$c.txt" 2>&1) &
done
wait
echo "done $M -> $OUT (remove the worktree with: git worktree remove --force $WT)"
