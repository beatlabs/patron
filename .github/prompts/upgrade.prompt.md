---
mode: agent
---

- Create a branch to work on.
- Upgrade all Go modules, tidy and vendor.
- Upgrade all Docker images in the docker-compose files.
- Run unit and integration tests to ensure everything works as expected.
- Commit changes and push the branch to origin.
- Create a pull request to merge the changes into the master branch.
- After CI checks are successful, merge (squash and merge) the pull request with admin privileges.