#!/usr/bin/env bash

set -euo pipefail

echo Testing ABQ environment ABQ_SET_EXIT_CODE when not set...

# shellcheck disable=SC2016
result=$(./captain run \
  --suite-id "captain-cli-abq-test" \
  --test-results .github/workflows/fixtures/rspec-quarantine.json \
  --github-job-name "Functional Test ABQ" \
  --fail-on-upload-error \
  -- bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE' \
  2>&1)
expected="exit_code=false"
if [[ "$result" == "$expected"* ]]; then
  echo PASSED;
else
  echo FAILED;
  echo "expected result to start with '$expected' but it was: $result"
  exit 1;
fi

echo Testing ABQ environment ABQ_SET_EXIT_CODE when already set overrides...

# shellcheck disable=SC2016
result=$(env ABQ_SET_EXIT_CODE=1234 ./captain run \
  --suite-id "captain-cli-abq-test" \
  --test-results .github/workflows/fixtures/rspec-quarantine.json \
  --github-job-name "Functional Test ABQ" \
  --fail-on-upload-error \
  -- bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE' \
  2>&1)
expected="exit_code=false"
if [[ "$result" == "$expected"* ]]; then
  echo PASSED;
else
  echo FAILED;
  echo "expected result to start with '$expected' but it was: $result"
  exit 1;
fi

echo Testing ABQ environment ABQ_STATE_FILE when not set...

# shellcheck disable=SC2016
result=$(./captain run \
  --suite-id "captain-cli-abq-test" \
  --test-results .github/workflows/fixtures/rspec-quarantine.json \
  --github-job-name "Functional Test ABQ" \
  --fail-on-upload-error \
  -- bash -c 'echo state_file=$ABQ_STATE_FILE' \
  2>&1)
expected="state_file=/tmp/captain-abq-"
if [[ "$result" == "$expected"* ]]; then
  echo PASSED;
else
  echo FAILED;
  echo "expected result to start with '$expected' but it was: $result"
  exit 1;
fi

echo Testing ABQ environment ABQ_STATE_FILE when already set does not override...

# shellcheck disable=SC2016
result=$(env ABQ_STATE_FILE=/tmp/functional-abq-1234.json ./captain run \
  --suite-id "captain-cli-abq-test" \
  --test-results .github/workflows/fixtures/rspec-quarantine.json \
  --github-job-name "Functional Test ABQ" \
  --fail-on-upload-error \
  -- bash -c 'echo state_file=$ABQ_STATE_FILE' \
  2>&1)
expected="state_file=/tmp/functional-abq-1234.json"
if [[ "$result" == "$expected"* ]]; then
  echo PASSED;
else
  echo FAILED;
  echo "expected result to start with '$expected' but it was: $result"
  exit 1;
fi
