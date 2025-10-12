---
mode: agent
model: Claude Sonnet 4.5 (Preview) (copilot)
---

- Create a branch to work on.
- Upgrade all Go modules, tidy and vendor.
- Upgrade all Docker images in the docker-compose files. Always pin to a specific version that depicts the highest major one.
- Upgrade Github Action workflows to the latest versions. Always pin to a specific version that depicts the highest major one.
- Run unit and integration tests with no caching to ensure everything works as expected.
- Commit changes and push the branch to origin.
- Create a pull request and wait for the CI to succeed.
- Report the overall changes that were made in the upgrade.