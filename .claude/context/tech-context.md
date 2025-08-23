---
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# Technical Context

## Technology Stack

### Primary Technologies
- **Shell Scripting (Bash)**: Core automation and system integration
- **Markdown**: Documentation, specifications, and command definitions
- **Git**: Version control and worktree management for parallel execution
- **GitHub CLI (gh)**: Issue management and repository operations

### Language Ecosystem
- **No traditional programming languages**: System built entirely on shell scripts and markdown
- **Shell compatibility**: Designed for Unix-like systems (macOS, Linux)
- **Cross-platform considerations**: Windows support through Git Bash/WSL

## Core Dependencies

### Essential Tools
1. **GitHub CLI (gh)** - Version 2.72.0 detected
   - Used for: Issue creation, management, and synchronization
   - Extensions: `gh-sub-issue` for parent-child relationships
   - Authentication: Required for GitHub operations

2. **Git** - Standard version control
   - Used for: Repository management, worktree operations, branching
   - Features utilized: Worktrees, branches, remote operations
   - Required for: All PM workflow operations

3. **Standard Unix Tools**
   - `find`, `grep`, `sed`, `awk`: Text processing and file operations
   - `date`: Timestamp generation for context files
   - `wc`, `head`, `tail`: Text analysis and output formatting
   - `mkdir`, `touch`, `chmod`: File system operations

### Optional Enhancements
- **ripgrep (rg)**: Enhanced text search capabilities
- **ast-grep**: Advanced code analysis patterns
- **jq**: JSON processing for GitHub API responses

## Development Environment

### System Requirements
- **Operating System**: Unix-like (macOS, Linux, WSL on Windows)
- **Shell**: Bash 3.0+ (modern features used)
- **Git**: 2.0+ (worktree support required)
- **GitHub CLI**: 2.0+ (modern issue management features)

### File System Layout
- **Working Directory**: Project root with `.claude/` subdirectory
- **Worktree Location**: `../epic-{name}/` (parallel to main project)
- **Log Files**: `tests/logs/` (created as needed)
- **Context Storage**: `.claude/context/` (persistent across sessions)

### Configuration Management
- **Local Settings**: `.claude/settings.local.json`
- **Git Configuration**: Standard git config for authentication
- **GitHub Authentication**: Via `gh auth` commands
- **Environment Variables**: Minimal usage, mostly PATH dependencies

## Architecture Patterns

### Shell Script Organization
- **Single Responsibility**: Each script handles one PM operation
- **Error Handling**: Consistent exit codes and error messages
- **Input Validation**: Preflight checks before operations
- **Output Formatting**: Structured output with status indicators

### Markdown as Configuration
- **Command Definitions**: Commands are markdown files with frontmatter
- **Documentation as Code**: Specifications directly executable
- **Frontmatter Metadata**: YAML headers for command configuration
- **Content Processing**: Markdown content becomes command instructions

### Agent System Architecture
- **Stateless Agents**: Each agent invocation is independent
- **Context Isolation**: Agents prevent main thread pollution
- **Specialized Purposes**: Each agent handles specific task types
- **Return Summaries**: Agents return concise summaries, not raw data

## Integration Patterns

### GitHub Integration
- **Issues as Database**: GitHub Issues serve as the primary data store
- **Comment-based Updates**: Progress tracking through issue comments
- **Label Organization**: Systematic labeling for epic/task relationships
- **Parent-Child Structure**: `gh-sub-issue` extension for hierarchical issues

### Git Workflow Integration
- **Worktree Isolation**: Each epic gets its own worktree
- **Branch Strategy**: Feature branches per epic/issue
- **Merge Strategy**: Clean merges back to main branch
- **Commit Traceability**: Link commits to specific issues

### Claude Code Integration
- **Command System**: Slash commands trigger PM operations
- **Agent Spawning**: Task tool launches specialized agents
- **Context Preservation**: Main conversation stays strategic
- **Parallel Execution**: Multiple agents work simultaneously

## Performance Characteristics

### Execution Speed
- **Local Operations**: Fast file system and git operations
- **GitHub API**: Rate-limited external calls (handled gracefully)
- **Parallel Agents**: Multiple concurrent work streams
- **Caching Strategy**: Minimal caching, always fresh data

### Scalability Factors
- **Issue Count**: Scales with GitHub's issue limits
- **Parallel Tasks**: Limited by system resources and git worktree capacity
- **Context Size**: Markdown files scale well, agent summaries keep context lean
- **Team Size**: Multiple team members can work simultaneously

### Resource Usage
- **Memory**: Minimal, mostly shell processes and file operations
- **Disk Space**: Worktrees increase disk usage during parallel work
- **Network**: GitHub API calls for synchronization operations
- **CPU**: Light usage except during parallel agent execution

## Security Considerations

### Authentication
- **GitHub Token**: Stored securely by GitHub CLI
- **Repository Access**: Requires push/pull permissions
- **No Secrets in Code**: All authentication external to system

### Data Privacy
- **Local Processing**: Most operations happen locally
- **GitHub Visibility**: Issues are public/private based on repository settings
- **No External Services**: Besides GitHub, no other external dependencies

### Access Control
- **Repository Permissions**: Controlled by GitHub repository settings
- **Branch Protection**: Can be configured independently
- **Issue Management**: Based on GitHub team permissions