# Development instructions

The below instructions are non-negotiable:

- we ALWAYS do TDD!!!
- any change we add to the codebase MUST be testable and tested
- we maintain a reasonable test coverage of more than 80%
- when we work on bugs we start with a test to reproduce the bug - we run the test, and it should fail - then we fix the issue - and the test should then pass
- use the available make targets to interact with the repo - e.g. to run tests, or build the services
- before you claim you're done with a task - run make test and ensure all tests are passing!!!

## Context

- [ADRs are defining how we build and what we build](docs/ADRs-index.md)
- 