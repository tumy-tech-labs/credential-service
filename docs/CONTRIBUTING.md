# Contributing to Credential-Service

Thank you for your interest in contributing to Credential-Service! We appreciate your time and effort. This guide will help you understand how to contribute effectively.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [How to Get Started](#how-to-get-started)
3. [Reporting Issues](#reporting-issues)
4. [Submitting Changes](#submitting-changes)
5. [Branching Model](#branching-model)
6. [Style Guidelines](#style-guidelines)
7. [Testing](#testing)
8. [Documentation](#documentation)
9. [License](#license)

## Code of Conduct

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) to understand the expected behavior when contributing to this project.

## How to Get Started

1. **Fork the repository**: Start by forking the main repository to your own GitHub account.
2. **Clone the forked repository**: Clone the fork locally on your machine.

   ```bash
   git clone https://github.com/bradtumy/credentia-service.git
   cd [repo-name]
   ```

3. **Set up your environment**: Follow the instructions in the `README.md` to set up your development environment.

### Project Structure

The repository is organized into different folders for each service:

```
issuer/           # Services related to issuing credentials
holder/           # Services related to managing and holding credentials
verifier/         # Services related to verifying credentials
common/           # Shared utilities and libraries used across services
docs/             # Project documentation
```

## Reporting Issues

If you encounter bugs or have feature requests, please check the [issue tracker](https://github.com/[org]/[repo]/issues) first to see if it has already been reported. If not, feel free to open a new issue and provide the following details:

- A clear and descriptive title.
- A detailed description of the issue.
- Steps to reproduce (if applicable).
- Screenshots or code snippets (if applicable).

## Submitting Changes

### Workflow

1. **Create a new branch** for your changes. Use a descriptive name that reflects the nature of the change (e.g., `fix/verifier-bug` or `feature/issuer-service`).

   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes**: Work on your feature or bug fix.

3. **Commit your changes**: Write clear and concise commit messages.

   ```bash
   git commit -m "Add feature X to the issuer service"
   ```

4. **Push your branch**: Push your branch to your forked repository.

   ```bash
   git push origin feature/my-new-feature
   ```

5. **Open a pull request (PR)**: Go to the main repository and open a pull request. Link any related issues and provide a brief description of your changes.

6. **Review process**: One of the maintainers will review your changes and provide feedback. Once approved, your PR will be merged into the main branch.

## Branching Model

We use the following branch model:

- **`main`**: The stable version of the project.
- **`dev`**: The active development branch. All new features should be merged here first.
- **Feature branches**: Use feature branches for new features or bug fixes. Branch off from `dev`.

## Style Guidelines

Follow these guidelines to ensure consistency across the project:

### Golang

- Use `gofmt` to format your Go code.
- Follow the official [Effective Go](https://golang.org/doc/effective_go) guidelines.
- Variable and function names should be descriptive and use camelCase.

### General

- Write meaningful and clear commit messages.
- Ensure your code is well-documented with comments.
  
## Testing

All code should be covered by unit tests, and existing tests should pass before submitting a pull request. Follow these steps:

1. Write unit tests in the `*_test.go` files.
2. Run tests for the relevant service. For example, to run tests in the `issuer` service:

   ```bash
   cd issuer
   go test ./...
   ```

3. Ensure that the entire test suite passes before submitting:

   ```bash
   go test ./...
   ```

4. Run linters and code formatters:

   ```bash
   gofmt -w .
   golangci-lint run
   ```

## Documentation

- Ensure that any new feature or API change is documented.
- Update the `docs/` directory or relevant `README.md` file with any changes.
- Consider adding comments to exported functions and types to improve GoDoc.

## License

By contributing, you agree that your contributions will be licensed under the [LICENSE](LICENSE) file in the repository.
