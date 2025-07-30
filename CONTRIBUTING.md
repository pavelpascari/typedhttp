# Contributing to TypedHTTP

Thank you for your interest in contributing to TypedHTTP! This document outlines the process for contributing to this project and helps ensure a smooth collaboration.

## 🎯 Project Philosophy

TypedHTTP follows Go community standards and best practices:

- **Simplicity**: Keep APIs simple and intuitive
- **Readability**: Code should be self-documenting and clear
- **Performance**: Efficient implementations with minimal allocations
- **Type Safety**: Leverage Go's type system for compile-time safety
- **Testing**: Comprehensive test coverage with TDD approach
- **Documentation**: Clear documentation and examples

## 🚀 Getting Started

### Prerequisites

- **Go 1.21+**: We use modern Go features including generics
- **Git**: For version control
- **Make**: For build automation (optional but recommended)

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/typedhttp.git
   cd typedhttp
   ```

3. **Set up the upstream remote**:
   ```bash
   git remote add upstream https://github.com/pavelpascari/typedhttp.git
   ```

4. **Install development dependencies**:
   ```bash
   make dev-setup
   ```

5. **Verify everything works**:
   ```bash
   make test
   make lint
   ```

## 🛠️ Development Workflow

### TDD Approach (Non-Negotiable)

We **ALWAYS** follow Test-Driven Development:

1. **Write a failing test** that describes the desired behavior
2. **Run the test** to confirm it fails
3. **Write minimal code** to make the test pass
4. **Refactor** while keeping tests green
5. **Repeat** for each feature/change

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Write tests first** (TDD requirement):
   ```bash
   # Create or update test files
   # Run tests to see them fail
   make test
   ```

3. **Implement your changes**:
   - Keep commits small and focused
   - Follow Go conventions and idioms
   - Ensure all tests pass

4. **Verify quality**:
   ```bash
   make test              # All tests must pass
   make test-coverage-check  # Coverage must be >80%
   make lint              # No linting errors
   make fmt               # Code properly formatted
   ```

### Available Make Targets

```bash
make test              # Run all tests
make test-verbose      # Run tests with race detection
make test-coverage     # Generate coverage report
make test-coverage-check # Verify >80% coverage
make lint              # Run linter
make fmt               # Format code
make build             # Build the project
make examples          # Build examples
make benchmark         # Run benchmarks
make ci                # Run all CI checks
make clean             # Clean build artifacts
make help              # Show all available targets
```

## 📝 Code Guidelines

### Go Style

Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go.html):

- Use `gofmt` for formatting (automated in our CI)
- Follow Go naming conventions
- Write clear, self-documenting code
- Use meaningful variable and function names
- Avoid unnecessary complexity

### Architecture Principles

- **Hexagonal Architecture**: Separate business logic from HTTP concerns
- **Single Responsibility**: Each component has one clear purpose
- **Dependency Injection**: Use interfaces for testability
- **Immutability**: Prefer immutable data structures where possible

### Code Structure

```go
// ✅ Good: Clear, focused function with single responsibility
func (d *JSONDecoder[T]) Decode(r *http.Request) (T, error) {
    var result T
    
    if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
        return result, fmt.Errorf("failed to decode JSON: %w", err)
    }
    
    if d.validator != nil {
        if err := d.validator.Struct(result); err != nil {
            return result, NewValidationError("validation failed", extractValidationErrors(err))
        }
    }
    
    return result, nil
}

// ❌ Avoid: Overly complex functions with multiple responsibilities
func (d *ComplexDecoder) DoEverything(r *http.Request) (interface{}, error) {
    // 50+ lines of mixed concerns...
}
```

### Testing Standards

Write comprehensive tests following these patterns:

```go
func TestJSONDecoder_Success(t *testing.T) {
    // Arrange
    decoder := NewJSONDecoder[TestRequest](validator.New())
    req := createTestRequest(`{"name": "John", "age": 30}`)
    
    // Act
    result, err := decoder.Decode(req)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "John", result.Name)
    assert.Equal(t, 30, result.Age)
}

func TestJSONDecoder_ValidationError(t *testing.T) {
    // Test error conditions with descriptive names
    decoder := NewJSONDecoder[TestRequest](validator.New())
    req := createTestRequest(`{"name": "", "age": -1}`)
    
    _, err := decoder.Decode(req)
    
    require.Error(t, err)
    var validationErr *ValidationError
    assert.True(t, errors.As(err, &validationErr))
}
```

### Documentation

- **Public APIs**: Must have comprehensive documentation
- **Examples**: Include runnable examples for complex features
- **ADRs**: Document architectural decisions in `docs/adrs/`
- **Comments**: Explain *why*, not *what* (code should be self-explanatory)

```go
// ✅ Good: Explains why and provides context
// CombinedDecoder implements multi-source request decoding with configurable
// precedence rules. It allows extracting data from multiple HTTP sources
// (path, query, headers, etc.) with intelligent fallback behavior.
type CombinedDecoder[T any] struct {
    // ...
}

// ❌ Avoid: Stating the obvious
// CombinedDecoder is a decoder that combines multiple decoders
```

## 🔄 Pull Request Process

### Before Submitting

1. **Sync with upstream**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**:
   ```bash
   make ci  # This runs fmt, vet, lint, and test-coverage-check
   ```

3. **Update documentation** if needed:
   - Update README.md for new features
   - Add/update examples
   - Create ADR for significant architectural changes

### PR Requirements

- **Clear title**: Describe what the PR does in one line
- **Comprehensive description**: Explain the problem, solution, and impact
- **Tests included**: All new code must have tests
- **Documentation updated**: If the change affects public APIs
- **Breaking changes**: Clearly marked and justified

### PR Template

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] All new and existing tests pass
- [ ] Test coverage is >80%

## Checklist
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have updated the documentation accordingly
- [ ] My changes generate no new warnings
```

## 🐛 Bug Reports

### Before Reporting

1. **Check existing issues** to avoid duplicates
2. **Try the latest version** to see if it's already fixed
3. **Create a minimal reproduction** example

### Bug Report Template

```markdown
## Bug Description
Clear and concise description of the bug.

## To Reproduce
Steps to reproduce the behavior:
1. Create a request with '...'
2. Call handler with '...'
3. See error

## Expected Behavior
What you expected to happen.

## Minimal Reproduction
```go
// Minimal code example that reproduces the issue
```

## Environment
- Go version: [e.g., 1.21.0]
- TypedHTTP version: [e.g., v1.0.0]
- OS: [e.g., macOS 14.0]
```

## 💡 Feature Requests

### Before Requesting

1. **Check existing issues** and discussions
2. **Consider the scope**: Does it fit the project's goals?
3. **Think about implementation**: How might it work?

### Feature Request Template

```markdown
## Problem Statement
What problem does this feature solve?

## Proposed Solution
How should this feature work?

## Alternatives Considered
What other approaches did you consider?

## Additional Context
Any other context, examples, or screenshots.
```

## 🏗️ Architecture Decisions

For significant changes, we use Architecture Decision Records (ADRs):

1. **Create an ADR** in `docs/adrs/` following the existing format
2. **Discuss the approach** before implementation
3. **Update the ADR** with implementation details
4. **Mark as implemented** when complete

## 🚀 Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### Release Checklist

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release notes
4. Tag the release
5. Update documentation

## 🤝 Community Guidelines

### Code of Conduct

- **Be respectful** and inclusive
- **Provide constructive feedback**
- **Help others learn** and grow
- **Focus on what's best** for the community

### Communication

- **GitHub Issues**: Bug reports and feature requests
- **Pull Requests**: Code contributions and discussions
- **Discussions**: General questions and ideas

## 📚 Resources

### Go Community Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Blog](https://blog.golang.org/)
- [Go Package Layout](https://github.com/golang-standards/project-layout)

### Project Resources

- [Architecture Decision Records](docs/adrs/)
- [API Documentation](https://pkg.go.dev/github.com/pavelpascari/typedhttp)
- [Examples](examples/)

## 🙋 Questions?

If you have questions about contributing, feel free to:

1. **Open a discussion** on GitHub
2. **Check existing issues** for similar questions
3. **Review the documentation** and examples

Thank you for contributing to TypedHTTP! Together, we're building better HTTP APIs for the Go community. 🚀