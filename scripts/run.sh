#!/bin/sh

set -eu

DOPPLER_ADDR=$(cf curl /v2/info | jq -r .doppler_logging_endpoint)
CF_ACCESS_TOKEN=$(cf oauth-token)
LOKI_URL=http://localhost:3100/api/prom/push

export DOPPLER_ADDR
export CF_ACCESS_TOKEN
export LOKI_URL

# Make sure loki / grafana are running
cd "$(dirname "$0")"
docker-compose up -d

cd ".."
GO111MODULE=on go run -mod=vendor main.go

