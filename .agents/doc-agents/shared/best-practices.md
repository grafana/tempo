# Documentation Best Practices

> **For human writers only.** Do not include this file in agent or skill instructions. It is reference material for people, not a task list for agents.

This guide captures best practices learned from documentation work, particularly around verifying accuracy, addressing user confusion, and maintaining consistency.

## Pre-Writing Checklist

Before writing or updating documentation:

- [ ] Identify the user problem or confusion point (e.g., GitHub issues)
- [ ] Locate the relevant codebase implementation
- [ ] Check the configuration reference documentation
- [ ] Review related documentation for consistency
- [ ] Understand feature scope (what's processor-specific vs. shared)

## Verification Process

Always verify documentation against multiple sources:

### 1. Codebase Verification
- Read the actual implementation code
- Verify feature names, configuration options, and behavior
- Check for default values and optional parameters
- Understand the relationship between features

### 2. Configuration Reference Check
- Compare documented configuration options with `docs/sources/tempo/configuration/_index.md`
- Ensure all options are documented consistently
- Verify YAML structure and examples match the reference

### 3. Version Compatibility
- Check CHANGELOG.md for when features were introduced
- Verify feature availability in target version (e.g., 2.10 vs. 3.0)
- Note any version-specific behavior or requirements

### 4. Style Guide Compliance
- Review against `.agents/doc-agents/shared/style-guide.md`
- Check for "see" vs "refer to" usage
- Verify heading structure and introductions
- Ensure examples use proper phrasing

## Common Pitfalls to Avoid

### 1. Confusing Examples
**Problem**: Examples that show default behavior instead of the feature's purpose
- **Bad**: Showing `deployment.environment` → `deployment_environment` (default sanitization)
- **Good**: Showing `deployment.environment` → `env` (actual renaming)

**Solution**: Always test examples to ensure they demonstrate the intended use case

### 2. Missing Clarifications
**Problem**: Assuming users understand implicit requirements
- **Bad**: Not explaining that `source_labels` must use original attribute names
- **Good**: Explicitly stating "must contain original span or resource attribute names (with dots)"

**Solution**: Add admonitions or explicit notes for non-obvious requirements

### 3. Unclear Feature Relationships
**Problem**: Not explaining how related features interact
- **Bad**: Listing `dimensions` and `dimension_mappings` without explaining they're alternatives
- **Good**: Explaining when to use each and that they're alternatives, not complementary

**Solution**: Add "Understanding X vs Y" sections for related features

### 4. Inaccurate Statements
**Problem**: Stating incorrect information (e.g., "two metrics" when there are three)
- **Solution**: Always verify against codebase, especially for counts, lists, and defaults

### 5. Missing Feature Documentation
**Problem**: Not documenting features that exist in code but aren't in docs
- **Solution**: Cross-reference codebase with documentation to find gaps

## Documentation Patterns

### Good Patterns

**Clear Introductions After Headings**
```markdown
### Disabling intrinsic dimensions

You can control which intrinsic dimensions are included in your metrics. Disable any of the default intrinsic dimensions using the `intrinsic_dimensions` configuration.
```

**Explicit Clarifications**
```markdown
{{< admonition type="note" >}}
The `source_labels` field must contain the **original span or resource attribute names** (with dots), not sanitized Prometheus label names.
{{< /admonition >}}
```

**Prose Over Lists for Explanations**
```markdown
Use `dimensions` when you want to add span attributes as labels using their default (sanitized) names. Use `dimension_mappings` when you want to rename attributes to custom label names or combine multiple attributes.
```

**Proper Example Phrasing**
```markdown
The following example shows how to rename the `deployment.environment` attribute to a shorter label called `env`, for example:
```

### Patterns to Avoid

**Vague References**
- ❌ "see below"
- ✅ "as described in the sections below" or "refer to the sections below"

**Passive Voice**
- ❌ "This processor mirrored the implementation"
- ✅ "This processor mirrors the implementation"

**Lists as Paragraph Substitutes**
- ❌ Using bullet lists for explanatory content
- ✅ Using prose paragraphs for explanations

## Addressing User Confusion

When addressing GitHub issues or user feedback:

1. **Identify Root Cause**: Understand what's actually confusing, not just the symptom
2. **Verify Against Code**: Ensure the documentation matches actual behavior
3. **Add Clarifications**: Don't just fix examples—add explicit guidance
4. **Explain Relationships**: Help users understand how features relate
5. **Provide Clear Examples**: Show actual use cases, not edge cases

## Review Process

Before submitting documentation, use [`verification-checklist.md`](verification-checklist.md).

## Continuous Improvement

- Review GitHub issues for documentation gaps
- Cross-reference codebase changes with documentation
- Update knowledge base when discovering new patterns
- Share learnings with the documentation team
