# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Claude Code Project Management (CCPM) system - a sophisticated project management workflow that transforms PRDs into shipped code through GitHub Issues integration and parallel AI agent execution. The system follows spec-driven development with full traceability from idea to production.

## Architecture

### Core Components

- **`.claude/`** - System configuration and workspace
  - `CLAUDE.md` - Core rules and agent instructions
  - `commands/` - Custom command definitions (`/pm:*`, `/context:*`, `/testing:*`)
  - `agents/` - Specialized agent definitions for context preservation
  - `scripts/` - Utility scripts including test execution
  - `epics/` - Local workspace for feature development (git ignored)
  - `prds/` - Product Requirements Documents
  - `context/` - Project context files for agent priming
  - `rules/` - System operation rules and patterns

### System Philosophy

1. **Spec-Driven Development** - Every line of code traces back to a specification
2. **Context Preservation** - Use specialized agents to prevent context pollution
3. **Parallel Execution** - Multiple agents work simultaneously on independent tasks
4. **GitHub Native** - Issues are the source of truth, not separate project tools
5. **Full Traceability** - Complete audit trail: PRD → Epic → Task → Issue → Code → Commit

## Common Development Commands

### Project Management Workflow
```bash
# Initialize the PM system
/pm:init

# Create new feature specification
/pm:prd-new feature-name

# Transform PRD into technical epic
/pm:prd-parse feature-name

# Break into tasks and sync to GitHub
/pm:epic-oneshot feature-name

# Start work on specific issue
/pm:issue-start 1234

# Check project status
/pm:status

# Get next priority task
/pm:next
```

### Context Management
```bash
# Create initial project context
/context:create

# Load context into conversation
/context:prime

# Update context after changes
/context:update
```

### Testing
```bash
# Configure testing setup
/testing:prime

# Run tests with analysis
/testing:run

# Run specific test (manual)
bash .claude/scripts/test-and-log.sh path/to/test.py
```

### Utilities
```bash
# Handle complex prompts
/prompt

# Update CLAUDE.md with PM rules
/re-init

# Process CodeRabbit reviews intelligently
/code-rabbit
```

## Development Patterns

### Agent Usage (Critical)
- **ALWAYS** use `file-analyzer` agent when reading verbose files
- **ALWAYS** use `code-analyzer` agent for code analysis and bug hunting
- **ALWAYS** use `test-runner` agent for test execution
- **ALWAYS** use `parallel-worker` agent for multi-stream issue work

### Parallel Execution Model
Issues are not atomic units. A single issue like "Implement user authentication" becomes:
- Agent 1: Database schemas and migrations
- Agent 2: Service layer and business logic  
- Agent 3: API endpoints and middleware
- Agent 4: UI components and forms
- Agent 5: Tests and documentation

All agents work simultaneously in git worktrees for maximum velocity.

### Context Optimization
- Main conversation stays strategic, agents handle implementation details
- Use agents as "context firewalls" to preserve main thread clarity
- Return summaries, not raw data from agent work

## System Rules

### Absolute Requirements
- **NO PARTIAL IMPLEMENTATION** - Complete all features fully
- **NO SIMPLIFICATION** - No "simplified for now" implementations
- **NO CODE DUPLICATION** - Reuse existing functions and constants
- **NO DEAD CODE** - Remove unused code completely
- **IMPLEMENT TESTS** - Every function needs tests
- **NO CHEATER TESTS** - Tests must be accurate and reveal flaws
- **NO INCONSISTENT NAMING** - Follow existing codebase patterns
- **NO OVER-ENGINEERING** - Simple functions over enterprise patterns
- **NO MIXED CONCERNS** - Proper separation of responsibilities
- **NO RESOURCE LEAKS** - Clean up connections, timeouts, listeners

### Error Handling Philosophy
- **Fail fast** for critical configuration issues
- **Log and continue** for optional features
- **Graceful degradation** when external services unavailable
- **User-friendly messages** through resilience layer

### Testing Guidelines
- Use test-runner agent for all test execution
- Do not use mock services
- Complete current test before moving to next
- Verify test structure before refactoring codebase
- Make tests verbose for debugging

## Directory Structure

```
depin/                          # Project root
├── .claude/                    # PM system configuration
│   ├── CLAUDE.md              # Core system rules
│   ├── commands/              # Custom command definitions
│   │   ├── pm/               # Project management commands
│   │   ├── context/          # Context management commands
│   │   └── testing/          # Test execution commands
│   ├── agents/                # Specialized agent definitions
│   ├── scripts/               # Utility scripts
│   │   ├── pm/               # PM workflow scripts
│   │   └── test-and-log.sh   # Test execution with logging
│   ├── epics/                 # Local workspace (git ignored)
│   ├── prds/                  # Product Requirements Documents
│   ├── context/               # Project context files
│   └── rules/                 # System operation patterns
├── CLAUDE.md                   # This file
├── README.md                   # Project documentation
├── AGENTS.md                   # Agent system documentation
└── COMMANDS.md                 # Command reference
```

## Integration Notes

### GitHub Integration
- Uses `gh-sub-issue` extension for parent-child relationships
- Epic issues automatically track sub-task completion
- Labels provide organization (`epic:feature`, `task:feature`)
- Comments maintain audit trail

### Git Worktree Usage
- Issues work in isolated worktrees: `../epic-{name}/`
- Enables true parallel development without conflicts
- Clean merges when work completes

### File Naming Conventions
- Tasks start as `001.md`, `002.md` during decomposition  
- After GitHub sync: renamed to `{issue-id}.md` (e.g. `1234.md`)
- Direct navigation: issue #1234 = file `1234.md`

## Tone and Behavior Expectations
- Criticism is welcome - tell me when I'm wrong
- Suggest better approaches when available
- Point out relevant standards or conventions
- Be skeptical and concise
- Ask questions rather than guessing intent
- No flattery or unnecessary compliments