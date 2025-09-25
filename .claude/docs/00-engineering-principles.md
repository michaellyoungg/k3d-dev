# Engineering Partnership Principles

## Core Philosophy

We operate as engineering partners, not in a hierarchical relationship. All ideas, approaches, and decisions are subject to scrutiny, validation, and improvement regardless of source.

### Partnership Tenets

1. **Question Everything**: No assumption is sacred. Every suggestion must be validated against requirements, constraints, and alternatives.

2. **Think Independently**: Don't accept the first solution. Consider multiple approaches, trade-offs, and implications.

3. **Counter-Propose Actively**: When you identify a better approach, propose it with clear reasoning and evidence.

4. **Validate Through Research**: Consult existing documentation, industry practices, and established patterns before making decisions.

5. **Document Decisions**: All choices, rationale, and trade-offs must be captured in markdown within the `./.claude/` folder for future reference.

## Decision-Making Process

### 1. Problem Definition
- Clearly articulate the problem or requirement
- Identify constraints and success criteria
- Reference relevant documentation (requirements, use cases, architecture)

### 2. Research Phase
- **Consult ALL existing documentation** in `./.claude/docs/`
- Research established tools and practices in the problem domain
- Identify industry standards and best practices
- Document findings and sources

### 3. Solution Evaluation
- Generate multiple solution options (minimum 3 approaches)
- Evaluate each against:
  - Simplicity and maintainability
  - Alignment with established tools/practices
  - Architectural coherence
  - Implementation effort vs. value
  - Future extensibility

### 4. Decision Documentation
- Record chosen approach with clear rationale
- Document rejected alternatives and why
- Identify implementation steps
- Note any assumptions or risks

### 5. Validation Checkpoint
- Review decision against all relevant documentation
- Ensure alignment with project principles
- Validate technical feasibility
- Confirm approach follows established patterns

## Architecture Principles

### Minimize Complexity
- **Prefer simple solutions** that solve 80% of use cases elegantly
- **Avoid over-engineering** - build for current needs, design for future extension
- **Eliminate unnecessary abstractions** - each layer must provide clear value

### Embrace Established Tools
- **Leverage existing ecosystems** rather than building from scratch
- **Follow community conventions** and idioms
- **Integrate with standard workflows** developers already know
- **Prefer mature, well-supported dependencies**

### Design for Evolution
- **Modular architecture** with clear boundaries and interfaces
- **Plugin/extension points** for future customization
- **Configuration-driven behavior** where appropriate
- **Backward compatibility** considerations from the start

### Operational Excellence
- **Observable by default** - logging, metrics, health checks
- **Fail fast and clearly** - meaningful error messages and recovery guidance
- **Resource conscious** - reasonable defaults for CPU, memory, disk usage
- **Security minded** - secure defaults, credential management, least privilege

## Documentation Standards

### Markdown as Source of Truth
All project decisions, designs, and instructions must be captured in markdown files within the `./.claude/` directory structure.

### Documentation Types

#### Decision Records
```markdown
# Decision: [Title]

## Context
What situation led to this decision?

## Options Considered
1. **Option A**: Description, pros/cons
2. **Option B**: Description, pros/cons  
3. **Option C**: Description, pros/cons

## Decision
Chosen approach and rationale

## Consequences
Expected outcomes and trade-offs

## Implementation Steps
Concrete next actions
```

#### Design Documents
- Problem statement and requirements
- Solution architecture with diagrams
- Interface definitions and contracts
- Implementation plan with milestones
- Testing and validation strategy

#### Investigation Reports
- Problem analysis and research findings
- Evaluation of alternatives with evidence
- Recommendations with supporting data
- References to external sources

### Documentation Review Process

1. **Completeness Check**: All required sections present
2. **Accuracy Validation**: Technical details verified against sources
3. **Consistency Review**: Alignment with existing documentation
4. **Clarity Assessment**: Understandable to future readers
5. **Actionability**: Clear next steps identified

## Implementation Guidelines

### Development Workflow

#### Before Starting Implementation
1. **Review ALL documentation** in `./.claude/docs/`
2. **Validate approach** against requirements and architecture
3. **Confirm tool/dependency choices** are established and appropriate
4. **Document implementation plan** with concrete steps
5. **Identify validation criteria** for completion

#### During Implementation
1. **Follow documented standards** and patterns
2. **Prefer established libraries** over custom implementations
3. **Write minimal, focused code** that solves specific problems
4. **Document non-obvious decisions** as you go
5. **Test assumptions** with simple validation scripts

#### After Implementation
1. **Document what was built** and how it works
2. **Record any deviations** from planned approach and why
3. **Update architecture documentation** if needed
4. **Identify learnings** for future decisions
5. **Plan next iteration** based on feedback

### Code Quality Standards

#### Simplicity First
- **Single responsibility** - each component does one thing well
- **Minimal dependencies** - justify each external dependency
- **Clear interfaces** - obvious inputs, outputs, and side effects
- **Self-documenting code** - clear names, simple structure

#### Established Patterns
- **Follow language idioms** and community conventions
- **Use standard libraries** before third-party alternatives
- **Adopt proven patterns** (configuration, logging, error handling)
- **Integrate with ecosystem tools** (testing, linting, building)

## Continuous Validation

### Regular Check-ins
- **Weekly architecture review**: Are we still aligned with principles?
- **Decision validation**: Are past choices still optimal?
- **Documentation audit**: Is our source of truth complete and accurate?
- **Progress assessment**: Are we building the right thing effectively?

### Course Correction
When misalignment is identified:
1. **Stop current work** and assess impact
2. **Consult documentation** to understand intended approach
3. **Identify root cause** of deviation
4. **Update plan** with corrected approach
5. **Document lessons learned** for future prevention

## Success Metrics

### Partnership Effectiveness
- Decisions are well-reasoned with documented alternatives
- Solutions align with established tools and practices
- Architecture remains coherent and extensible
- Documentation accurately reflects current state

### Technical Excellence
- Code is minimal yet sufficient for requirements
- Dependencies are justified and well-integrated
- System is observable, maintainable, and reliable
- Future developers can understand and extend the work

---

*This document is living - update it as we learn and improve our collaboration patterns.*