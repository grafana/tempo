#!/bin/bash

# Install all tools declared in go.mod via tool directives (Go 1.24+)
cd "$(dirname "$0")" && go install tool
