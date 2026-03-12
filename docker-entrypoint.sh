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

extra_args=""

if ! has_flag --omni-api-endpoint "$@"; then
    require_env OMNI_ENDPOINT
    extra_args="$extra_args --omni-api-endpoint=${OMNI_ENDPOINT}"
fi

if ! has_flag --config-file "$@"; then
    require_env CONFIG_FILE
    extra_args="$extra_args --config-file=${CONFIG_FILE}"
fi

if ! has_flag --provider-name "$@"; then
    require_env PROVIDER_NAME
    extra_args="$extra_args --provider-name=${PROVIDER_NAME}"
fi

if ! has_flag --id "$@"; then
    require_env ID
    extra_args="$extra_args --id=${ID}"
fi

# shellcheck disable=SC2086
exec /app/omni-infra-provider-hetzner $extra_args "$@"