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

echo Testing Partitioning w/out timings...

set +e
part1=$(./captain partition \
  .github/workflows/fixtures/partition/{x,y,z}.rb \
  --suite-id "captain-cli-functional-tests" --index 0 --total 2)

if [[ "$part1" == ".github/workflows/fixtures/partition/x.rb .github/workflows/fixtures/partition/z.rb" ]]; then
  echo PASSED;
else
  echo FAILED;
  exit 1;
fi

part2=$(./captain partition \
  .github/workflows/fixtures/partition/{x,y,z}.rb \
  --suite-id "captain-cli-functional-tests" --index 1 --total 2)

if [[ "$part2" == ".github/workflows/fixtures/partition/y.rb" ]]; then
  echo PASSED;
else
  echo FAILED;
  exit 1;
fi
set -e

echo Testing Partitioning w/ timings...

set +e
./captain upload results ./.github/workflows/fixtures/partition/rspec-partition.json --suite-id captain-cli-functional-tests

part1=$(./captain partition \
  .github/workflows/fixtures/partition/*_spec.rb \
  --suite-id "captain-cli-functional-tests" --index 0 --total 2)

if [[ "$part1" == ".github/workflows/fixtures/partition/a_spec.rb .github/workflows/fixtures/partition/d_spec.rb" ]]; then
  echo PASSED;
else
  echo "FAILED: $part1";
  exit 1;
fi

part2=$(./captain partition \
  .github/workflows/fixtures/partition/*_spec.rb \
  --suite-id "captain-cli-functional-tests" --index 1 --total 2)

if [[ "$part2" == ".github/workflows/fixtures/partition/b_spec.rb .github/workflows/fixtures/partition/c_spec.rb" ]]; then
  echo PASSED;
else
  echo "FAILED: $part2";
  exit 1;
fi
set -e


echo Testing upload short circuits when nothing to upload...

set +e
./captain upload results nonexistingfile.json --suite-id captain-cli-functional-tests
if [[ $? -eq 0 ]]; then
  echo PASSED;
else
  echo FAILED;
  exit 1;
fi
set -e
