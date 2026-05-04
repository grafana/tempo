# Personas and User Model

This document defines a shared persona and intent model for product documentation.

The goal is to:

- Align content to user needs
- Enable consistent information architecture
- Support AI agents and automation

---

## Core Principle

Personas are defined by **capability**, not job title.

We focus on:

- What users can do
- What they are trying to accomplish
- How much guidance they need

---

## Personas

### Learner

A technical user who needs guidance and conceptual understanding while trying to accomplish a task.

- Not an observability expert
- May not know query languages
- Learns in the context of doing

**Needs:**

- Conceptual overviews
- Use cases and scenarios
- Step-by-step walkthroughs

**Modes:**

- Orientation: "What can I do with this?"
- Execution: "Help me do this"

---

### Practitioner

A task-oriented user who is comfortable with tools and workflows.

- Understands core concepts
- May use UI + some configuration
- Focused on completing tasks efficiently

**Needs:**

- Task-based guides
- Real-world examples
- Connections between features

---

### Expert

An advanced user with deep technical knowledge.

- Comfortable with query languages and internals
- Prefers precision and control
- Skips introductory material

**Needs:**

- Reference documentation
- Advanced examples
- Edge cases and limitations

---

### Operator

A user responsible for running and maintaining systems.

- Thinks in terms of systems, scale, and reliability
- Often works across components

**Needs:**

- Architecture and system design
- Setup and configuration
- Troubleshooting and failure modes

---

## Use Cases (Primary Axis)

Documentation should be organized around what users want to accomplish.

Core use cases:

- **Understand** — Learn concepts and capabilities
- **Investigate** — Troubleshoot and diagnose issues
- **Implement** — Set up and configure systems
- **Operate** — Monitor and maintain systems
- **Optimize** — Improve performance, cost, or reliability

---

## Entry State (User Starting Point)

Users arrive with different levels of clarity.

- "I don't know what this does"
- "I know what I want to do"
- "I need exact syntax or configuration"
- "I need to run or maintain this system"

Entry state helps determine:

- Persona
- Content type
- Level of detail

---

## Interaction Mode

AI changes how users interact with the system.

### Manual

- Uses UI, docs, and queries
- Capability determines what they can do

### Assisted (AI-enabled)

- Uses natural language and AI guidance
- Can perform tasks above their skill level

**Implications for docs:**

- Include AI-assisted workflows where relevant
- Provide validation and explanation
- Offer manual alternatives

---

## Learning Context

Distinguishes between two types of learning:

### Product Usage (default)

- User is trying to complete a task
- Learning happens as needed

### Structured Learning

- User is following courses or learning paths
- Content is progressive and instructional

---

## How to Use This Model

When creating or updating content:

1. Identify the **use case**
2. Identify the **persona**
3. Identify the **entry state**
4. Consider **interaction mode (AI vs manual)**

Then choose:

- Content type
- Level of detail
- Structure

---

## Example

User goal: "Debug a slow service"

| Persona      | Content                             |
| ------------ | ----------------------------------- |
| Learner      | Guided troubleshooting walkthrough  |
| Practitioner | Task-based guide with examples      |
| Expert       | Query examples and reference        |
| Operator     | System-level investigation workflow |
