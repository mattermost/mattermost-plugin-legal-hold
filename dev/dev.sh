#!/usr/bin/env bash
# Development stack for this plugin: a Mattermost server, PostgreSQL and a bootstrap that creates
# users and installs the plugin. The stack is named after the git worktree.
#
#   ./dev.sh up        start the stack, bootstrap the server, build and install the plugin
#   ./dev.sh deploy    build the plugin again and install it
#   ./dev.sh down      stop the stack and keep its data
#   ./dev.sh destroy   stop the stack and delete its data
#   ./dev.sh logs      follow the server logs
#   ./dev.sh status    show the containers, the URL and the logins
#   ./dev.sh env       print the environment for the Playwright tests
#   ./dev.sh mmctl …   run mmctl --local inside the server container
#
# Optional settings (environment variables):
#   MM_IMAGE     server image (default: mattermost/mattermost-enterprise-edition:11.7)
#   MM_PORT      host port (default: derived from the worktree name)
#   MM_LICENSE   path to a license file, uploaded on bootstrap (default: dev/license.txt)

set -euo pipefail

DEV_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(git -C "$DEV_DIR" rev-parse --show-toplevel)

ADMIN_USER=admin
ADMIN_PASSWORD='Testing-123!'
TEAM_NAME=dev
USERS=(user-1 user-2 user-3)

# The stack and the port follow the worktree, so worktrees do not clash.
WORKTREE=$(basename "$ROOT")
PROJECT=$(printf '%s' "$WORKTREE" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9_-]+/-/g; s/^[^a-z0-9]+//')
PORT=${MM_PORT:-$((18000 + $(printf '%s' "$PROJECT" | cksum | cut -d' ' -f1) % 1000))}
URL="http://localhost:${PORT}"

write_env() {
    {
        echo "COMPOSE_PROJECT_NAME=${PROJECT}"
        echo "MM_PORT=${PORT}"
        if [ -n "${MM_IMAGE:-}" ]; then
            echo "MM_IMAGE=${MM_IMAGE}"
        fi
    } > "$DEV_DIR/.env"
}

compose() {
    docker compose --project-directory "$DEV_DIR" -f "$DEV_DIR/docker-compose.yml" "$@"
}

mmctl() {
    compose exec -T mattermost mmctl --local "$@"
}

log() {
    printf '\033[1m==> %s\033[0m\n' "$*"
}

wait_for_server() {
    log "Waiting for the server at ${URL}"
    for _ in $(seq 1 90); do
        if curl -sf "${URL}/api/v4/system/ping" > /dev/null; then
            return 0
        fi
        sleep 2
    done
    echo "The server did not become ready. Check: ./dev.sh logs" >&2
    return 1
}

bootstrap() {
    wait_for_server

    log "Creating users and the ${TEAM_NAME} team"
    mmctl user create --username "$ADMIN_USER" --email "${ADMIN_USER}@example.com" \
        --password "$ADMIN_PASSWORD" --system-admin --email-verified > /dev/null 2>&1 || true
    mmctl team create --name "$TEAM_NAME" --display-name 'Dev Team' --email "${ADMIN_USER}@example.com" > /dev/null 2>&1 || true
    for user in "${USERS[@]}"; do
        mmctl user create --username "$user" --email "${user}@example.com" \
            --password "$ADMIN_PASSWORD" --email-verified > /dev/null 2>&1 || true
    done
    mmctl team users add "$TEAM_NAME" "$ADMIN_USER" "${USERS[@]}" > /dev/null 2>&1 || true

    local license=${MM_LICENSE:-$DEV_DIR/license.txt}
    if [ -f "$license" ]; then
        log "Uploading the license"
        compose cp "$license" mattermost:/tmp/license.mattermost-license
        mmctl license upload /tmp/license.mattermost-license > /dev/null || true
    fi
}

deploy() {
    wait_for_server

    log "Building the plugin for linux/amd64"
    (
        cd "$ROOT"
        make dist MM_SERVICESETTINGS_ENABLEDEVELOPER=true DEFAULT_GOOS=linux DEFAULT_GOARCH=amd64
    )

    local plugin_id version bundle
    # The manifest tool reads plugin.json from the working directory.
    plugin_id=$(cd "$ROOT" && ./build/bin/manifest id)
    version=$(cd "$ROOT" && ./build/bin/manifest version)
    bundle="$ROOT/dist/${plugin_id}-${version}.tar.gz"

    log "Installing ${plugin_id} ${version}"
    compose cp "$bundle" mattermost:/tmp/plugin.tar.gz
    mmctl plugin add /tmp/plugin.tar.gz > /dev/null 2>&1 || {
        mmctl plugin delete "$plugin_id" > /dev/null 2>&1 || true
        mmctl plugin add /tmp/plugin.tar.gz
    }
    mmctl plugin enable "$plugin_id"
}

status() {
    compose ps
    cat <<INFO

Stack:    ${PROJECT}
URL:      ${URL}
Admin:    ${ADMIN_USER} / ${ADMIN_PASSWORD}
Users:    ${USERS[*]} (same password)
Team:     ${TEAM_NAME}
INFO
}

print_env() {
    cat <<ENV
export PW_BASE_URL=${URL}
export PW_ADMIN_USERNAME=${ADMIN_USER}
export PW_ADMIN_PASSWORD='${ADMIN_PASSWORD}'
export PW_HEADLESS=true
ENV
}

write_env

case "${1:-}" in
    up)
        compose up -d --wait
        bootstrap
        deploy
        status
        ;;
    bootstrap) bootstrap ;;
    deploy) deploy ;;
    down) compose down ;;
    destroy) compose down --volumes ;;
    logs) compose logs -f --tail 100 mattermost ;;
    status) status ;;
    env) print_env ;;
    mmctl) shift; mmctl "$@" ;;
    *)
        sed -n '2,/^$/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
        exit 1
        ;;
esac
