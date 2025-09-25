# Project Rules and Implementation Guidelines

## Mandatory Rules

### Documentation Location
- **ALL documentation** must be stored in the `./.claude/` folder
- **ALL decisions** must be captured in markdown files  
- **ALL architectural discussions** must be documented before implementation
- **ALL code decisions** must reference supporting documentation

### Engineering Partnership
- **Question every assumption** - nothing is sacred or beyond scrutiny
- **Validate all suggestions** against requirements and existing documentation
- **Counter-propose better approaches** when identified
- **Think independently** - don't accept first solution without analysis

### Technology Choices
- **Prefer established tools** and industry-standard practices
- **Minimize complexity** - simple solutions over bespoke implementations
- **Strong architectural design** over complex feature-rich solutions
- **Research thoroughly** before selecting dependencies or patterns

### Process Requirements
- **Consult ALL documentation** in `./.claude/docs/` before making decisions
- **Follow all documented steps** in decision-making workflow
- **Update documentation** immediately after decisions or changes
- **Validate against architectural principles** before implementation

## Implementation Workflow

### Before Starting Any Work
1. **Read complete documentation** in `./.claude/docs/`
   - [ ] 00-engineering-principles.md
   - [ ] 01-project-overview.md  
   - [ ] 02-requirements.md
   - [ ] 03-use-cases.md
   - [ ] 04-ergonomics.md
   - [ ] 05-architecture.md
   - [ ] 06-technology-integration.md
   - [ ] 07-decision-workflow.md

2. **Understand the problem** completely
   - Define success criteria clearly
   - Identify all constraints
   - Reference applicable use cases
   - Validate against requirements

3. **Research established approaches**
   - Industry standards and best practices
   - Existing tools and libraries
   - Community recommendations
   - Reference implementations

### During Implementation
1. **Follow documented architecture** and patterns
2. **Use established tools** from technology integration guide
3. **Document deviations** and rationale immediately
4. **Validate frequently** against requirements and use cases
5. **Update documentation** as decisions are made

### After Implementation
1. **Document what was built** and how it works
2. **Update architecture docs** if patterns changed
3. **Record lessons learned** for future reference
4. **Plan next iteration** based on validation results

## Quality Gates

### Decision Quality
- [ ] Multiple options considered (minimum 3)
- [ ] Established tools/practices researched
- [ ] Trade-offs clearly documented
- [ ] Alignment with architecture validated
- [ ] Implementation steps defined

### Implementation Quality  
- [ ] Follows documented architecture patterns
- [ ] Uses established tools appropriately
- [ ] Code is minimal yet sufficient
- [ ] Error handling is robust
- [ ] Documentation is updated

### Documentation Quality
- [ ] All decisions captured in markdown
- [ ] Clear rationale provided
- [ ] References to supporting materials
- [ ] Actionable next steps identified
- [ ] Stored in `./.claude/` folder structure

## Escalation Criteria

### Mandatory Discussion Points
- **Core architecture changes** - any modification to fundamental design
- **Tool/dependency additions** - new external dependencies
- **Deviation from standards** - non-standard approaches or patterns
- **Complex trade-offs** - when multiple viable options exist
- **Performance implications** - choices affecting system performance

### Discussion Process
1. **Document the dilemma** using decision record template
2. **Present options** with research and analysis
3. **Identify missing information** or validation needs
4. **Propose experiments** to validate approaches
5. **Make decision** based on evidence and principles

## Success Indicators

### Partnership Success
- Decisions are well-reasoned with documented alternatives
- Solutions consistently use established tools and practices  
- Architecture remains coherent and extensible
- Documentation accurately reflects current state and decisions

### Technical Success  
- Code is minimal, focused, and maintainable
- Dependencies are justified and well-integrated
- System follows planned architecture patterns
- Future developers can understand and extend the work

---

**Remember**: These rules exist to ensure we build the right thing, the right way, with clear reasoning and maintainable results. When in doubt, document the question and research thoroughly before proceeding.