# Contributing to the High-Performance Fibonacci Calculator

First off, thank you for considering contributing to this project! Your help is greatly appreciated.

## How Can I Contribute?

### Reporting Bugs

If you find a bug, please open an issue and provide the following:

-   A clear and descriptive title.
-   A detailed description of the problem, including the exact command you ran.
-   Steps to reproduce the bug.
-   The expected behavior and what happened instead.
-   Your operating system and Go version.

### Suggesting Enhancements

If you have an idea for an enhancement, please open an issue to discuss it. This allows us to align on the feature before you put effort into implementing it.

### Pull Requests

1.  **Fork the repository** and create your branch from `main`.
2.  **Make your changes.** Ensure that your code follows the existing style and conventions.
3.  **Add or update tests** for your changes. We aim for high test coverage.
4.  **Ensure all tests pass** by running `go test ./...`.
5.  **Format your code** with `gofmt`.
6.  **Submit a pull request** with a clear description of your changes.

## Development Setup

1.  Clone your fork of the repository.
2.  Ensure you have Go 1.25 or later installed.
3.  Run `go build -o fibcalc ./cmd/fibcalc` to build the executable.
4.  Run `go test ./...` to run the test suite.

## Code Style

-   Follow the standard Go formatting guidelines (`gofmt`).
-   Write clear and concise comments where necessary.
-   Keep functions and methods short and focused on a single responsibility.

Thank you for your contribution!
