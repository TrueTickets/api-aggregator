# Contributing to API Aggregator

Thank you for your interest in contributing to the API Aggregator project! This document provides guidelines and information for contributors.

## 🤝 How to Contribute

### Types of Contributions

We welcome several types of contributions:

- **🐛 Bug Reports**: Help us identify and fix issues
- **✨ Feature Requests**: Suggest new functionality or improvements
- **📚 Documentation**: Improve or add to our documentation
- **🔧 Code Contributions**: Submit bug fixes or new features
- **🧪 Testing**: Help improve test coverage or add test cases
- **🔍 Code Reviews**: Review pull requests from other contributors

### Getting Started

1. **Fork the Repository**: Click the "Fork" button on GitHub
2. **Clone Your Fork**: `git clone https://github.com/your-username/api-aggregator.git`
3. **Set Up Development Environment**: Follow the setup instructions below
4. **Create a Branch**: `git checkout -b feature/your-feature-name`
5. **Make Changes**: Implement your changes
6. **Test Your Changes**: Ensure all tests pass
7. **Submit a Pull Request**: Open a PR with a clear description

## 🛠️ Development Setup

### Prerequisites

- **Go 1.23+**: Required for building the application
- **Docker**: For containerized testing and development
- **Make**: For running build commands
- **Git**: For version control

### Setting Up the Environment

```bash
# Clone the repository
git clone https://github.com/TrueTickets/api-aggregator.git
cd api-aggregator

# Install dependencies
go mod download

# Install development tools
make install-tools

# Set up pre-commit hooks
pre-commit install
```

### Development Commands

```bash
# Development workflow
make dev                     # Build, test, and lint
make build                   # Build binary
make test                    # Run unit tests
make lint                    # Run linting
make integration-test        # Run integration tests
make clean                   # Clean build artifacts

# Running the service
./api-aggregator             # Run with default config
API_AGGREGATOR_CONFIG_PATH=custom.yaml ./api-aggregator
```

## 📋 Contribution Guidelines

### Code Style

- **Go Formatting**: Use `gofmt` and `goimports` (automated by pre-commit)
- **Linting**: Code must pass `golangci-lint` checks
- **Comments**: Add comments for complex logic and exported functions
- **Error Handling**: Use proper error handling patterns
- **Testing**: Write tests for new functionality

### Commit Messages

We follow emoji-based commit message conventions. See `.claude/custom/commit_formatting.md` for detailed guidelines.

Examples:

- `🐛 Fix response merging for empty backend responses`
- `✨ Add support for XML response transformation`
- `📚 Update configuration documentation`
- `🧪 Add integration tests for timeout scenarios`

### Pull Request Process

1. **Create Descriptive PR**: Use the PR template and fill out all sections
2. **Link Issues**: Reference related issues using keywords like "Closes #123"
3. **Add Tests**: Include tests for new functionality
4. **Update Documentation**: Update relevant documentation
5. **Pass CI Checks**: Ensure all GitHub Actions pass
6. **Request Review**: Tag maintainers for review
7. **Address Feedback**: Make changes based on review comments

### Testing Requirements

- **Unit Tests**: All new code must have unit tests
- **Integration Tests**: Add integration tests for new endpoints or features
- **Test Coverage**: Maintain or improve test coverage
- **Manual Testing**: Test your changes manually before submitting

### Documentation Requirements

- **Code Comments**: Document complex functions and logic
- **README Updates**: Update README.md for new features
- **Configuration Examples**: Add examples for new configuration options
- **CLAUDE.md**: Update context guide for significant changes

## 🔧 Technical Guidelines

### Architecture Principles

- **Modular Design**: Keep components loosely coupled
- **Error Handling**: Use consistent error patterns
- **Configuration**: Make features configurable when appropriate
- **Performance**: Consider performance implications of changes
- **Security**: Follow security best practices

### Adding New Features

1. **Design Discussion**: Open an issue to discuss the design
2. **Configuration**: Add configuration options if needed
3. **Implementation**: Implement the feature following existing patterns
4. **Testing**: Add comprehensive tests
5. **Documentation**: Update all relevant documentation

### Backend Integration

When adding support for new backend types or features:

- Add configuration validation
- Implement proper error handling
- Add timeout support
- Include comprehensive tests
- Document configuration options

### Response Transformation

When extending transformation capabilities:

- Follow existing transformation patterns
- Add configuration schema validation
- Include edge case handling
- Add performance optimizations
- Document transformation behavior

## 🐛 Bug Reports

Use the bug report template and include:

- **Clear Description**: Describe the issue clearly
- **Reproduction Steps**: Provide step-by-step instructions
- **Environment Details**: Include OS, Go version, deployment method
- **Configuration**: Share relevant configuration (sanitized)
- **Logs**: Include relevant log output
- **Expected vs Actual**: Describe expected and actual behavior

## ✨ Feature Requests

Use the feature request template and include:

- **Problem Statement**: Explain what problem you're solving
- **Proposed Solution**: Describe your proposed approach
- **Use Cases**: Provide concrete use cases
- **Configuration Examples**: Show how it might be configured
- **Impact Assessment**: Consider performance and compatibility

## 🧪 Testing

### Running Tests

```bash
# Unit tests
make test
go test ./...

# Integration tests
make integration-test

# Specific package tests
go test ./internal/merger -v

# Test coverage
go test -cover ./...
```

### Writing Tests

- **Test Structure**: Use table-driven tests where appropriate
- **Test Names**: Use descriptive test names
- **Edge Cases**: Test edge cases and error conditions
- **Mocking**: Use interfaces for mocking dependencies
- **Integration**: Add end-to-end integration tests

## 📚 Documentation

### Types of Documentation

- **Code Documentation**: Inline comments and godoc
- **Configuration Documentation**: README.md and examples
- **API Documentation**: OpenAPI/Swagger specs
- **Development Documentation**: CLAUDE.md and contributing guides

### Documentation Standards

- **Clarity**: Write clear, concise documentation
- **Examples**: Include practical examples
- **Completeness**: Cover all configuration options
- **Maintenance**: Keep documentation up-to-date

## 🔒 Security

### Security Considerations

- **Input Validation**: Validate all inputs
- **Authentication**: Handle authentication securely
- **Secrets Management**: Never hardcode secrets
- **Error Messages**: Avoid leaking sensitive information
- **Dependencies**: Keep dependencies updated

### Reporting Security Issues

Report security vulnerabilities privately through GitHub's security advisory system, not through public issues.

## 🎯 Code Review

### As a Contributor

- **Self-Review**: Review your own code before submitting
- **Respond Promptly**: Address review feedback quickly
- **Be Respectful**: Maintain a professional tone
- **Learn**: Use reviews as learning opportunities

### As a Reviewer

- **Be Constructive**: Provide helpful, actionable feedback
- **Explain Reasoning**: Explain why changes are needed
- **Appreciate Effort**: Acknowledge good work
- **Focus on Code**: Review code, not the person

## 🏷️ Labels and Project Management

### Issue Labels

- **Type**: `bug`, `enhancement`, `documentation`, `question`
- **Priority**: `priority/high`, `priority/medium`, `priority/low`
- **Status**: `needs-triage`, `in-progress`, `blocked`
- **Area**: `area/config`, `area/transformation`, `area/client`

### Project Boards

We use GitHub Projects to track:

- **Backlog**: Planned features and improvements
- **In Progress**: Currently being worked on
- **Review**: Ready for review
- **Done**: Completed items

## 📞 Getting Help

### Communication Channels

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and community discussion
- **Pull Request Comments**: For code-specific discussions
- **Documentation**: Check README.md and CLAUDE.md first

### Response Times

- **Issues**: We aim to respond within 2-3 business days
- **Pull Requests**: Initial review within 5 business days
- **Security Issues**: Immediate attention

## 🎉 Recognition

### Contributors

All contributors are recognized in:

- GitHub contributor graphs
- Release notes for significant contributions
- Project documentation

### Maintainers

Project maintainers are responsible for:

- Reviewing and merging pull requests
- Triaging issues and feature requests
- Maintaining code quality standards
- Coordinating releases

## 📄 License

By contributing to this project, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to API Aggregator! Your efforts help make this project better for everyone. 🚀
