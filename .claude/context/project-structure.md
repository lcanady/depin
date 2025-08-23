---
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# Project Structure

## Root Directory Organization

```
depin/                              # Project root (CCPM system)
├── .claude/                        # Core PM system configuration
├── .git/                          # Git repository metadata
├── .gitignore                     # Git ignore patterns
├── AGENTS.md                      # Agent system documentation
├── CLAUDE.md                      # Claude Code instructions (updated)
├── COMMANDS.md                    # Command reference documentation  
├── LICENSE                        # MIT License
├── README.md                      # Primary project documentation
└── screenshot.webp                # Project screenshot/demo
```

## Core System Architecture (.claude/)

### Primary Directories

```
.claude/
├── agents/                        # Specialized agent definitions (4 agents)
├── commands/                      # Custom command implementations
├── context/                       # Project context files (this directory)
├── epics/                        # Local epic workspace (git ignored)
├── prds/                         # Product Requirements Documents  
├── rules/                        # System operation rules (10 rule files)
├── scripts/                      # Utility and PM scripts
├── CLAUDE.md                     # Core system rules and instructions
└── settings.local.json           # Local system settings
```

### Agent System (agents/)
Four specialized agents for context optimization:
- `code-analyzer.md` - Bug hunting and code analysis across multiple files
- `file-analyzer.md` - Verbose file reading and summarization
- `parallel-worker.md` - Parallel work stream coordination in worktrees
- `test-runner.md` - Test execution with result analysis

### Command System (commands/)

```
commands/
├── context/                      # Context management commands (3 commands)
│   ├── create.md                # Initialize project context
│   ├── prime.md                 # Load context into conversation
│   └── update.md                # Refresh existing context
├── pm/                          # Project management commands (40+ commands)
│   ├── [epic-*].md             # Epic management commands
│   ├── [issue-*].md            # Issue workflow commands
│   ├── [prd-*].md              # PRD creation and management
│   └── [workflow commands]      # Status, sync, validation commands
├── testing/                     # Test execution commands (2 commands)
├── code-rabbit.md              # CodeRabbit integration
├── prompt.md                   # Complex prompt handling
└── re-init.md                  # CLAUDE.md updating
```

### Rules System (rules/)
Ten specialized rule files governing system operations:
- `agent-coordination.md` - Multi-agent coordination patterns
- `branch-operations.md` - Git branch management rules
- `datetime.md` - Date/time handling standards
- `frontmatter-operations.md` - Frontmatter processing rules
- `github-operations.md` - GitHub API interaction patterns
- `standard-patterns.md` - Common implementation patterns
- `strip-frontmatter.md` - Content processing utilities
- `test-execution.md` - Testing framework rules
- `use-ast-grep.md` - Code analysis tool usage
- `worktree-operations.md` - Git worktree management

### Scripts System (scripts/)

```
scripts/
├── pm/                          # Project management automation (14 scripts)
│   ├── epic-*.sh               # Epic management automation
│   ├── issue-*.sh              # Issue workflow automation  
│   ├── prd-*.sh                # PRD processing automation
│   ├── help.sh                 # Command help system
│   ├── init.sh                 # System initialization
│   ├── status.sh               # Project status reporting
│   └── validate.sh             # System integrity checking
└── test-and-log.sh             # Test execution with logging
```

## File Naming Conventions

### Task Files
- **During decomposition**: `001.md`, `002.md`, `003.md`
- **After GitHub sync**: `{issue-id}.md` (e.g., `1234.md`)
- **Navigation pattern**: Issue #1234 maps to file `1234.md`

### Epic Organization
- **Epic directory**: `.claude/epics/{epic-name}/`
- **Epic file**: `epic.md` (implementation plan)
- **Task files**: Individual task markdown files
- **Updates**: `updates/` subdirectory for work-in-progress

### Command Files
- **Kebab-case naming**: `epic-show.md`, `issue-start.md`
- **Category prefixes**: Commands grouped by functionality
- **Consistent structure**: Frontmatter + instructions + error handling

## Module Organization

### Functional Separation
- **PM Workflow**: Complete project management lifecycle
- **Agent System**: Context optimization and parallel execution
- **Command System**: User interaction and workflow automation
- **Integration Layer**: GitHub, Git, testing frameworks

### Data Flow Patterns
- **PRD → Epic → Tasks → Issues → Code**: Clear traceability chain
- **Local First**: Operations work locally, sync to GitHub explicitly
- **Agent Isolation**: Context preservation through specialized agents
- **Worktree Execution**: Parallel development in isolated environments

## Key Architectural Decisions

### Directory Placement
- **System files in .claude/**: Keeps project root clean
- **Generated content git ignored**: Epics directory excluded from version control  
- **Documentation at root**: README, AGENTS, COMMANDS visible immediately
- **Context separation**: Project context isolated in dedicated directory

### Scalability Patterns
- **Command extensibility**: Easy addition of new commands
- **Agent specialization**: Purpose-built agents for specific tasks
- **Rule-based operations**: Consistent behavior through shared rules
- **GitHub native**: Leverages existing team infrastructure