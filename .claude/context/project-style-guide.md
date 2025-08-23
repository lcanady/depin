---
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# Project Style Guide

## File and Directory Conventions

### Naming Standards
- **Kebab-case for files**: `epic-show.md`, `issue-start.md`, `context-create.md`
- **Lowercase directories**: `commands/`, `scripts/`, `agents/`, `context/`
- **Descriptive names**: Names should clearly indicate purpose and scope
- **Consistent prefixes**: Group related files with common prefixes (`epic-*`, `issue-*`, `prd-*`)

### Directory Structure Patterns
- **Functional organization**: Group files by purpose, not implementation details
- **Shallow hierarchies**: Prefer shallow over deep directory structures
- **Hidden system files**: System configuration in `.claude/` to keep project root clean
- **Logical separation**: Clear boundaries between user-facing and internal files

### Task File Naming Evolution
```
Decomposition Phase: 001.md, 002.md, 003.md
Post-GitHub Sync: 1234.md, 1235.md, 1236.md
Navigation Pattern: Issue #1234 → File 1234.md
```

## Shell Script Conventions

### Script Structure
```bash
#!/bin/bash
# 
# Brief description of what the script does
# Usage: script-name.sh [required-args] [optional-args]

# Preflight checks first
if [ ! -d ".git" ]; then
    echo "❌ Not a git repository"
    exit 1
fi

# Main logic with clear error handling
# Exit codes: 0 for success, non-zero for failure
```

### Error Handling Patterns
- **Fail fast**: Check prerequisites before starting work
- **Clear messages**: Specific error messages with suggested solutions
- **Consistent exit codes**: 0 for success, 1 for user error, 2 for system error
- **User-friendly output**: Use emoji indicators (✅ ❌ ⚠️) for status

### Output Formatting
- **Status indicators**: ✅ for success, ❌ for errors, ⚠️ for warnings
- **Progress feedback**: Show what's happening during long operations
- **Structured output**: Consistent formatting for similar operations
- **Actionable messages**: Tell users what to do next, not just what happened

## Markdown Documentation Standards

### File Structure
```markdown
---
frontmatter: Required for all files
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# Primary Heading (matches filename)

## Clear section organization
Content organized in logical sections...
```

### Frontmatter Requirements
- **created**: ISO 8601 UTC timestamp when file was created
- **last_updated**: ISO 8601 UTC timestamp of last modification
- **version**: Semantic version (1.0, 1.1, 2.0)
- **author**: "Claude Code PM System" for system-generated files

### Content Organization
- **Single H1**: One primary heading per file matching the filename
- **Logical hierarchy**: H2 for major sections, H3 for subsections
- **Consistent formatting**: Code blocks, lists, and emphasis used consistently
- **Actionable content**: Focus on what users need to do, not just information

### Code Block Standards
```bash
# Command examples with comments explaining purpose
/pm:prd-new feature-name  # Create new PRD

# Multi-line examples with clear structure
git status
git add .
git commit -m "feat: implement user authentication"
```

## Command Definition Patterns

### Command File Structure
```markdown
---
allowed-tools: Read, Write, LS, Bash
---

# Command Name

Brief description of command purpose and usage.

## Required Rules
[Reference to relevant rule files]

## Preflight Checklist
[Validation steps before execution]

## Instructions
[Step-by-step implementation]

## Error Handling
[Common failure modes and recovery]
```

### Command Naming
- **Verb-noun pattern**: `epic-show`, `issue-start`, `prd-create`
- **Category prefixes**: `pm:`, `context:`, `testing:` for namespace organization
- **Action clarity**: Command name should clearly indicate what it does
- **Consistency**: Similar operations use similar naming patterns

## Agent Definition Standards

### Agent File Structure
```markdown
# Agent Name

**Purpose**: Single sentence describing agent's specialized function
**Pattern**: Input → Processing → Output workflow
**Usage**: When to use this agent vs others
**Returns**: What information comes back to main thread

## Core Function
[Detailed description of agent capabilities]

## Usage Examples
[Concrete examples of agent invocation and results]
```

### Agent Responsibilities
- **Single purpose**: Each agent has one clear, focused responsibility  
- **Context isolation**: Agents prevent pollution of main conversation
- **Summary returns**: Return 10-20% of processed information
- **Error handling**: Graceful failure with clear error reporting

## Configuration File Standards

### JSON Configuration
```json
{
    "setting_name": "value",
    "nested_settings": {
        "subsetting": "value"
    },
    "arrays": [
        "item1",
        "item2"
    ]
}
```

### YAML Frontmatter
```yaml
---
required_field: value
optional_field: value
arrays:
  - item1
  - item2
nested:
  field: value
---
```

## Communication Patterns

### User-Facing Messages
- **Clear status**: Always indicate what's happening and why
- **Actionable guidance**: Tell users what to do next
- **Error recovery**: Provide specific steps to recover from failures
- **Progress indication**: Show progress during long-running operations

### Internal Communication
- **Structured logging**: Consistent log formats for debugging
- **Error propagation**: Preserve error context through system layers
- **State tracking**: Clear indication of system state at all times
- **Audit trails**: All significant actions logged for review

## Quality Standards

### Code Quality
- **No partial implementation**: Complete all features fully
- **No dead code**: Remove unused functionality completely  
- **Consistent naming**: Follow established patterns throughout codebase
- **Error handling**: Every operation checks for and handles failures
- **Resource cleanup**: Close connections, clear timeouts, remove listeners

### Documentation Quality
- **Executable documentation**: Documentation that serves as implementation
- **Current information**: Documentation updated with code changes
- **Complete examples**: Working examples for all documented features
- **User perspective**: Documentation written from user's point of view

### Testing Standards
- **Comprehensive coverage**: Test all significant functionality
- **Realistic scenarios**: Tests reflect actual usage patterns
- **Clear assertions**: Test failures indicate exactly what's wrong
- **Fast execution**: Tests run quickly to encourage frequent use

## Version Control Patterns

### Commit Message Format
```
type(scope): description

feat(pm): add epic decomposition with parallel task identification
fix(agents): resolve context pollution in parallel-worker agent
docs(readme): update installation instructions for gh-sub-issue
```

### Branch Management
- **Feature branches**: One branch per epic or significant feature
- **Worktree isolation**: Use git worktrees for parallel development
- **Clean merges**: Squash commits when merging to main
- **Descriptive names**: Branch names indicate purpose and scope

### File Organization
- **Logical grouping**: Related files in same directory
- **Clear dependencies**: Dependencies flow in one direction where possible
- **Separation of concerns**: Different types of functionality in different areas
- **Minimal coupling**: Changes in one area don't require changes in others

These conventions ensure consistency across all CCPM components and make the system maintainable as it grows in complexity and adoption.