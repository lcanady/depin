---
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# System Patterns

## Architectural Design Patterns

### Command Pattern Implementation
- **Command Files**: Each PM operation is a separate markdown file
- **Consistent Structure**: Frontmatter + instructions + error handling
- **Parameterization**: Commands accept arguments through special variables
- **Execution Flow**: Claude Code reads file and executes instructions

### Agent Pattern (Context Firewall)
- **Specialization**: Each agent handles specific types of work
- **Isolation**: Agents prevent context pollution in main conversation
- **Summary Returns**: Agents return 10-20% of processed information
- **Stateless Operation**: Each agent invocation is independent

### Repository Pattern (GitHub as Database)
- **Issues as Records**: GitHub Issues store project state
- **Comments as History**: Progress updates maintained in issue comments
- **Labels as Categories**: Systematic labeling for organization
- **Relations via References**: Issue numbers create linkage

## Data Flow Patterns

### Spec-Driven Development Flow
```
PRD Creation → Epic Planning → Task Decomposition → GitHub Sync → Parallel Execution
```

#### Phase-Gate Pattern
- **Gate 1**: PRD approval before epic creation
- **Gate 2**: Epic review before task decomposition  
- **Gate 3**: Task validation before GitHub sync
- **Gate 4**: Sync confirmation before parallel execution

#### Traceability Chain
Every code change traces back through:
1. **Commit** → linked to specific task
2. **Task** → part of defined epic
3. **Epic** → derived from approved PRD
4. **PRD** → addresses business requirement

### Context Optimization Pattern

#### Agent Hierarchy
```
Main Conversation (Strategic)
├── file-analyzer (File Processing)
├── code-analyzer (Bug Hunting)  
├── test-runner (Test Execution)
└── parallel-worker (Issue Coordination)
    ├── Sub-agent 1 (Database)
    ├── Sub-agent 2 (API)
    ├── Sub-agent 3 (UI)
    └── Sub-agent N (Tests)
```

#### Information Filtering
- **Heavy Lifting**: Agents do messy work (reading files, running tests)
- **Context Isolation**: Implementation details stay in agent context
- **Concise Returns**: Only essential information returns to main thread
- **Parallel Processing**: Multiple agents work without context collision

## State Management Patterns

### Local-First Architecture
- **Local Operations**: All work happens locally first
- **Explicit Sync**: GitHub synchronization is deliberate and controlled
- **Offline Capable**: Most operations work without network connectivity
- **Sync Validation**: Bidirectional sync ensures consistency

### File System as State Store
- **Epic Directories**: `.claude/epics/{name}/` contain all epic state
- **Task Files**: Individual markdown files track task progress
- **Context Files**: `.claude/context/` maintains project awareness
- **Git Integration**: Version control provides state history

### Progressive Enhancement Pattern
- **Core Functionality**: Basic PM workflow without external dependencies
- **GitHub Enhancement**: Issues provide team visibility and collaboration
- **Agent Enhancement**: Specialized agents improve performance
- **Tool Enhancement**: Additional tools (ast-grep, ripgrep) add capabilities

## Error Handling Patterns

### Fail-Fast Principle
- **Preflight Checks**: Validate prerequisites before operations
- **Early Validation**: Check inputs and environment before processing  
- **Clear Messages**: Specific error messages with suggested solutions
- **Graceful Degradation**: Fallback options when possible

### Resilience Patterns
- **Retry Logic**: Network operations retry with backoff
- **Partial Success**: Some operations can complete partially
- **State Recovery**: Operations can resume from interruption points
- **User Guidance**: Clear instructions for manual recovery

## Integration Patterns

### GitHub Integration Strategy
- **API-First**: Use GitHub CLI and API for all operations
- **Batch Operations**: Group related API calls to minimize requests
- **Rate Limiting**: Respect GitHub API rate limits
- **Authentication**: Leverage existing GitHub authentication

### Git Worktree Pattern
- **Isolation**: Each epic gets its own worktree directory
- **Parallel Development**: Multiple features can develop simultaneously
- **Clean Merges**: Finished work merges cleanly to main branch
- **Resource Management**: Worktrees cleaned up after completion

### Claude Code Integration
- **Command Interface**: Slash commands provide consistent UX
- **Agent Spawning**: Task tool launches appropriate specialized agents
- **Context Management**: Strategic conversation stays clean and focused
- **Parallel Coordination**: Multiple agents work on same issue simultaneously

## Quality Patterns

### Documentation as Code
- **Executable Documentation**: Command files serve as both docs and implementation
- **Version Control**: All documentation versioned with code
- **Single Source of Truth**: Documentation and implementation can't diverge
- **Automated Testing**: Commands can be validated programmatically

### Testing Strategy
- **Test-Driven Documentation**: Examples in docs serve as tests
- **Integration Testing**: End-to-end workflow validation
- **Manual Testing**: Human verification of critical paths
- **Continuous Validation**: `/pm:validate` checks system integrity

### Code Quality Patterns
- **Single Responsibility**: Each script/agent/command has one clear purpose
- **Consistent Naming**: Kebab-case for files, clear descriptive names
- **Error Handling**: Every operation checks for and handles failure modes
- **User Experience**: Clear feedback, progress indicators, helpful messages

## Scalability Patterns

### Horizontal Scaling (Team Size)
- **GitHub Native**: Multiple developers can work simultaneously
- **Context Sharing**: Shared context files keep team aligned
- **Issue Assignment**: GitHub issue assignment for coordination
- **Audit Trail**: Complete history in GitHub for transparency

### Vertical Scaling (Project Size)
- **Agent Specialization**: More specialized agents for complex projects
- **Command Extension**: New commands can be added without changing core system
- **Rule Refinement**: Rules can be customized per project
- **Context Segmentation**: Context can be organized by domain/feature

### Performance Scaling
- **Parallel Execution**: Multiple agents work simultaneously
- **Local Processing**: Minimize network dependencies
- **Efficient Tools**: Use fast tools (ripgrep vs grep, ast-grep vs regex)
- **Smart Caching**: Cache when beneficial, always-fresh when needed