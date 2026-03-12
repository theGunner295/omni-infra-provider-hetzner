#!/bin/sh
set -eu

require_env() {
    var_name="$1"

    value="$(printenv "$var_name" || true)"
    if [ -z "$value" ]; then
        echo "error: required environment variable $var_name is not set" >&2
        exit 1
    fi
}

# Check whether a given long option is already present in the arguments.
# Usage: has_flag --option-name "$@"
has_flag() {
    flag="$1"
    shift

    for arg in "$@"; do
        case "$arg" in
            "$flag"|"${flag}"=*)
                return 0
                ;;
        esac
    done

    return 1
}

# If the first argument doesn't start with '-', and resolves to an actual
# executable in PATH, exec it directly so that commands like
# `docker run <image> sh` work without going through the provider entrypoint
# logic. Otherwise, fall through to the provider CLI.
case "${1:-}" in
    -* | "") ;;
    *)
        if command -v "$1" >/dev/null 2>&1; then
            exec "$@"
        fi
        ;;
esac

extra_args=""

if ! has_flag --omni-api-endpoint "$@"; then
    require_env OMNI_ENDPOINT
    set -- "$@" --omni-api-endpoint "${OMNI_ENDPOINT}"
fi

if ! has_flag --config-file "$@"; then
    require_env CONFIG_FILE
    set -- "$@" --config-file "${CONFIG_FILE}"
fi

if ! has_flag --provider-name "$@"; then
    if [ -n "${PROVIDER_NAME:-}" ]; then
        set -- "$@" --provider-name "${PROVIDER_NAME}"
    fi
fi

if ! has_flag --id "$@" && [ -n "${ID:-}" ]; then
    set -- "$@" --id "${ID}"
fi

exec /app/omni-infra-provider-hetzner "$@"