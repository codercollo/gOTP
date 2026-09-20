#!/usr/bin/env bash
set -euo pipefail

echo "Scaffolding GoTP project..."

# Directories
mkdir -p \
  .github/workflows \
  cmd/api \
  internal/otp \
  internal/guard \
  internal/data \
  internal/sms \
  internal/validator \
  internal/jsonlog \
  api \
  web/static \
  configs \
  build \
  scripts \
  docs

# GitHub Actions
touch \
  .github/workflows/test.yml \
  .github/workflows/deploy-staging.yml \
  .github/workflows/deploy-production.yml

# API
touch \
  cmd/api/main.go \
  cmd/api/server.go \
  cmd/api/routes.go \
  cmd/api/handlers.go \
  cmd/api/middleware.go \
  cmd/api/errors.go \
  cmd/api/helpers.go

# Internal packages
touch \
  internal/otp/otp.go \
  internal/otp/otp_test.go \
  internal/guard/velocity.go \
  internal/guard/ratio.go \
  internal/guard/guard_test.go \
  internal/data/models.go \
  internal/data/store.go \
  internal/sms/africastalking.go \
  internal/validator/validator.go \
  internal/jsonlog/jsonlog.go

# API / frontend / configuration
touch \
  api/openapi.yaml \
  web/static/index.html \
  configs/.env.example

# Build / scripts / docs
touch \
  build/Dockerfile \
  scripts/demo.sh \
  docs/design.md \
  docs/marketplace.md

# Root files
touch \
  Makefile \
  README.md

# Initialise Go module if it doesn't already exist.
if [[ ! -f go.mod ]]; then
    echo "Initialising Go module..."
    go mod init github.com/itsco/gotp
fi

# Make shell scripts executable
chmod +x scripts/demo.sh

echo
echo "✓ GoTP scaffold created."
echo
echo "Project tree:"
find . \
  -not -path './.git/*' \
  -not -path './.github/*' \
  -type f \
  | sort

echo
echo "Next:"
echo "  1. Implement internal/otp"
echo "  2. Implement internal/guard"
echo "  3. Add tests"
echo "  4. Wire cmd/api"
echo
echo "Run:"
echo "  bash scaffold.sh"

