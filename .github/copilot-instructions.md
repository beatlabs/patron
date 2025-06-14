# Copilot configuration for this repository
# For documentation, see: https://docs.github.com/en/copilot/configuration/copilot-configuration-in-your-repository

ignore:
  - 'vendor/**'
  - 'docs/**'
  - 'internal/test/**'
  - 'encoding/protobuf/test/**'
  - '**/test/**'
  - '**/*_test.go'
  - '**/integration_test.go'
  - '.github/**'

default_language: go

enabled:
  - '**/*.go'
  - '!**/*_test.go'

disabled:
  - '**/test/**'
  - '**/*_test.go'
  - '**/integration_test.go'
  - 'vendor/**'
  - 'docs/**'
  - '.github/**'

# Add your custom configuration below.
#
# We use the testify framework for testing. Copilot suggestions should not conflict with testify idioms.
test_framework: testify

# Code style and conventions:
- Use context.Context as the first argument in functions that may block or are request-scoped.
- Prefer error wrapping using fmt.Errorf or errors.Join when returning errors.
- Follow Go idioms for naming, error handling, and documentation.
- Avoid global variables except for constants or configuration.
- Document public functions and types with clear, concise comments.

