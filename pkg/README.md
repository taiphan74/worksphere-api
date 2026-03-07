# pkg/

**Role:** Reusable, framework-agnostic utility packages.

Unlike `internal/`, code inside `pkg/` is intended to be importable by other projects
or services in the future. Keep packages here *generic* — they should have **no
dependencies** on Gin, GORM, or any other application-specific library.

## Planned sub-packages

| Package          | Purpose                                              |
|------------------|------------------------------------------------------|
| `pkg/response`   | Standardised JSON response envelope helpers          |
| `pkg/validator`  | Custom validation rules / error formatting           |
| `pkg/logger`     | Structured logger setup (e.g. zerolog / zap wrapper) |
| `pkg/pagination` | Generic pagination utilities                         |
| `pkg/apperror`   | Typed application error definitions                  |
