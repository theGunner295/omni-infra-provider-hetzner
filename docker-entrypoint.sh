#!/bin/sh
set -eu

if [ "$#" -gt 0 ]; then
    exec /app/omni-infra-provider-hetzner "$@"
fi

require_env() {
    var_name="$1"

    if [ -z "$(eval "printf '%s' \"\${$var_name:-}\"")" ]; then
        echo "error: required environment variable $var_name is not set" >&2
        exit 1
    fi
}

require_env OMNI_ENDPOINT
require_env CONFIG_FILE
require_env PROVIDER_NAME
require_env ID

set -- /app/omni-infra-provider-hetzner

set -- "$@" "--omni-api-endpoint=${OMNI_ENDPOINT}"
set -- "$@" "--config-file=${CONFIG_FILE}"
set -- "$@" "--provider-name=${PROVIDER_NAME}"
set -- "$@" "--id=${ID}"

exec "$@"