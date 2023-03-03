#!/usr/bin/env bash

set -e

go run ../../../cmd/kanvas export -d .github/workflows && act pull_request
