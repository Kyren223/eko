#!/usr/bin/env bash

test -z "$(go fmt ./...)"
staticcheck ./...
