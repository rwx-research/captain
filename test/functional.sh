#!/usr/bin/env bash

set -euo pipefail

echo Testing Quarantining...

./captain run \
  --suite-id "captain-cli-quarantine-test" \
  --test-results .github/workflows/fixtures/rspec-quarantine.json \
  --github-job-name "Build & Test" \
  --fail-on-upload-error \
  -- bash -c "exit 1"
echo PASSED;

echo Testing Failures not Quarantined...

set +e
./captain run \
  --suite-id "captain-cli-functional-tests" \
  --test-results .github/workflows/fixtures/rspec-failed-not-quarantined.json \
  --github-job-name "Build & Test" \
  --fail-on-upload-error \
  -- bash -c "exit 123"
if [[ $? -eq 123 ]]; then
  echo PASSED;
else
  echo FAILED;
  exit 1;
fi

echo Testing Failures without Results File...

set +e
./captain run \
  --suite-id "captain-cli-functional-tests" \
  --test-results .github/workflows/fixtures/does-not-exist.json \
  --github-job-name "Build & Test" \
  -- bash -c "exit 123"
if [[ $? -eq 123 ]]; then
  echo PASSED;
else
  echo FAILED;
  exit 1;
fi

set -e
