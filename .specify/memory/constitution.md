<!--
Sync Impact Report:
  Version change: none → 1.0.0 (initial)
  Modified principles: none (first edition)
  Added sections: Core Principles, Quality Standards, Development Workflow, Governance
  Removed sections: none
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ aligned (Constitution Check placeholder is per-feature)
    - .specify/templates/spec-template.md ✅ aligned (no constitution-specific rules)
    - .specify/templates/tasks-template.md ✅ aligned (testing tasks match quality gates)
  Follow-up TODOs: none
-->

# Radius Server Constitution

## Core Principles

### I. Test-First & High Coverage (NON-NEGOTIABLE)

Unit test coverage MUST exceed 70% for all new and modified code. Tests are written before implementation, following TDD discipline. Red-Green-Refactor cycle is the standard development rhythm. No code merges without corresponding tests that demonstrate the feature works.

### II. Clean Code & Brevity

Code self-documents through clear naming and structure. Avoid comments and documentation unless the logic is non-trivial for experienced developers in the domain. If a reader needs to understand what the code does, improve the code rather than adding explanation. Prose belongs in specs, not source files.

### III. Modularity & Size Limits

Files MUST NOT exceed 500 lines. Classes and top-level modules MUST NOT exceed 500 lines. Code is organized into small, focused, single-responsibility units. When a file nears the limit, extract logically cohesive parts into new files. Imports should be minimal—each module owns a narrow concern.

### IV. Minimal UI for Power Users

UI surfaces MUST be minimal, dense, and efficient. Prioritize keyboard-driven workflows, shortcuts, and configuration over menus and wizards. Assume users understand domain concepts—no hand-holding, no tutorial overlays, no redundant labels. Every pixel must carry information or afford action.

### V. Pre-Commit Quality Gates

Lint and test MUST pass before every commit. Pre-commit hooks enforce this gate automatically. The build pipeline reflects the same checks—what passes locally passes in CI. No exceptions for hotfixes; fast fixes can skip documentation but not tests or lint.

## Quality Standards

**Coverage threshold**: 70% minimum unit test coverage, measured on changed lines.
**Linter**: Zero warnings permitted. Warnings are errors.
**Line limits**: 500 lines per file, 500 lines per class/module definition.
**Complexity**: Functions should be short and focused. If cyclomatic complexity feels high, extract.

## Development Workflow

1. Write tests that fail for the expected behavior.
2. Implement minimum code to pass those tests.
3. Refactor while keeping tests green.
4. Lint must pass. Commit fails if pre-commit hooks fail.
5. All checks visible and reproducible locally.

Code reviews verify: coverage thresholds, file size limits, no unnecessary comments, minimal and functional UI decisions.

## Governance

This constitution supersedes all other development practices for this project. Amendments require: proposed change, rationale, version bump per semver (MAJOR for principle removals/redefinitions, MINOR for additions/expansions, PATCH for clarifications), and update of this document.

All commits are implicitly reviewed against these principles. Pre-commit gates enforce test and lint compliance automatically. Complexity must be justified—simplicity is the default.

**Version**: 1.0.0 | **Ratified**: 2026-06-21 | **Last Amended**: 2026-06-21
