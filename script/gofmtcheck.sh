#!/usr/bin/env bash

# Check gofmt
echo "==> Checking that code complies with gofmt requirements..."
gofmt_files=$(find . -type f -name '*.go' \
  -not \( -path ./.git -prune \) \
  -not \( -path ./vendor -prune \) \
  -not \( -path '*/vendor' -prune \) \
  -print0 | xargs -0 gofmt -l)
if [[ -n ${gofmt_files} ]]; then
    echo 'gofmt needs running on the following files:'
    echo "${gofmt_files}"
    echo "You can use the command: \`make fmt\` to reformat code."
    exit 1
fi

exit 0
