---
created: 2025-08-23T17:03:39Z
last_updated: 2025-08-23T17:03:39Z
version: 1.0
author: Claude Code PM System
---

# Project Brief

## What It Does

**Claude Code Project Management (CCPM)** is a sophisticated workflow system that bridges the gap between product requirements and shipped code through structured, AI-assisted development processes. It transforms traditional single-threaded AI development into a coordinated, parallel execution system that maintains context and traceability.

### Primary Function
CCPM operates as a **collaboration protocol** that enables humans and AI agents to work together at scale, using GitHub Issues as the shared database and Claude Code as the execution environment.

## Why It Exists

### The Problem
Every development team struggles with the same fundamental issues:
- **Context evaporates** between sessions, forcing constant re-discovery
- **Parallel work creates conflicts** when multiple developers touch the same code
- **Requirements drift** as verbal decisions override written specs
- **Progress becomes invisible** until the very end
- **AI-assisted development becomes a silo** without team visibility

### The Solution Philosophy
CCPM follows a strict **"No Vibe Coding"** principle: every line of code must trace back to a specification. This eliminates the biggest source of technical debt and ensures that development work addresses real business needs.

### The Strategic Advantage
By using GitHub Issues as the database, CCPM unlocks **true team collaboration** where multiple Claude instances can work on the same project simultaneously, human developers see AI progress in real-time, and team members can jump in anywhere because the context is always visible.

## Core Objectives

### Immediate Goals
1. **Eliminate Context Loss**: Persistent context across all work sessions
2. **Enable Parallel Execution**: Multiple AI agents working simultaneously 
3. **Ensure Traceability**: Complete audit trail from idea to production
4. **Improve Team Collaboration**: Seamless human-AI handoffs
5. **Increase Development Velocity**: 3-8x faster feature delivery

### Strategic Goals
1. **Scale AI-Assisted Development**: From solo work to team coordination
2. **Establish Development Standards**: Spec-driven development as the norm
3. **Create Reusable Patterns**: Template workflows for common scenarios
4. **Build Team Capabilities**: Human developers working effectively with AI
5. **Ensure Quality and Compliance**: Full audit trails for regulated environments

## Success Criteria

### Technical Success
- **System Reliability**: Commands execute consistently without errors
- **GitHub Integration**: Seamless synchronization with GitHub Issues
- **Parallel Execution**: Multiple agents work without conflicts
- **Context Preservation**: No loss of project state between sessions
- **Traceability**: Every code change links back to business requirement

### Business Success
- **Developer Productivity**: Measurable increase in feature delivery speed
- **Code Quality**: Reduction in bugs and technical debt
- **Team Efficiency**: Better coordination between human and AI work
- **Process Compliance**: Audit trails meet organizational requirements
- **Knowledge Retention**: Project knowledge persists beyond individual sessions

### User Experience Success
- **Intuitive Commands**: Easy to learn and remember command structure
- **Clear Feedback**: Users always know what's happening and why
- **Error Recovery**: Graceful handling of failures with clear guidance
- **Documentation**: Self-documenting system with executable examples
- **Onboarding**: New users productive within minutes

## Project Scope

### In Scope
- **Complete PM Workflow**: PRD creation through code delivery
- **GitHub Integration**: Issues, comments, labels, parent-child relationships
- **Parallel Agent System**: Specialized agents for context optimization
- **Command Interface**: 40+ commands for all PM operations
- **Context Management**: Persistent project knowledge system
- **Git Integration**: Worktree management for parallel development
- **Quality Assurance**: Testing integration and validation commands

### Out of Scope (Current Version)
- **Multi-repository Support**: Single repository focus
- **Advanced Project Management**: No Gantt charts, resource allocation
- **Real-time Collaboration**: Async coordination only
- **Mobile Interface**: Command-line interface only
- **Custom GitHub Apps**: Uses existing GitHub CLI and APIs
- **Database Storage**: GitHub Issues serve as the database

### Future Scope (Potential)
- **Multi-repository Orchestration**: Cross-repo epic management
- **Advanced Analytics**: Delivery metrics and team performance
- **Integration APIs**: Connect with other project management tools
- **Visual Dashboards**: Web interface for progress visualization
- **Advanced Automation**: Smart task prioritization and assignment

## Key Constraints

### Technical Constraints
- **GitHub Dependency**: Requires GitHub repository and CLI access
- **Unix Environment**: Designed for Unix-like systems (bash scripts)
- **Internet Connectivity**: GitHub operations require network access
- **Git Repository**: Must be run within a git repository
- **Claude Code**: Designed specifically for Claude Code interface

### Business Constraints
- **Open Source**: MIT license requires attribution
- **GitHub Rates**: Subject to GitHub API rate limiting
- **Team Size**: Optimized for small to medium teams (2-20 people)
- **Learning Curve**: Requires understanding of git concepts and GitHub

### Quality Constraints
- **No Partial Implementation**: Complete features only
- **Full Traceability**: Every change must link to specification
- **Testing Requirements**: All functionality must include tests
- **Documentation Standards**: Self-documenting system requirement

## Stakeholder Alignment

### Development Teams
CCPM serves teams that want to move beyond ad-hoc AI assistance to structured, trackable development processes while maintaining the speed advantages of AI-assisted development.

### Product Managers
CCPM provides product managers with unprecedented visibility into AI development work and ensures that technical implementation stays aligned with business requirements.

### Engineering Managers
CCPM gives engineering managers the tools to oversee AI-assisted development with the same rigor as traditional development, including audit trails and quality assurance processes.