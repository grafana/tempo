# Doc agents

We use two types of documentation agents: a generic AI agent, which serves as the default tool for product teams without a writer and handles the basic, standardized workflow; and the AI twin, a personalized agent that encodes your unique research, structure, writing, and review process. The generic agent fills the gap when no writer is available, while the AI twin amplifies your individual craft and raises the quality bar for the teams you support.

Both types of doc agent are designed to help you generate, update, and maintain documentation across any Grafana project. They guide you through the entire documentation workflow—from understanding the product to drafting, reviewing, and preparing PRs—or you can run them at any individual stage you choose. They build structure, create and update content, validate links, and surface issues, while you stay in control of what to approve, refine, or publish.

This guide explains what each agent does within the documentation workflow and what responsibilities remain with you as the writer.

## What you do (as the writer)

The agents are installed in your project.

Use the agent directly in VS Code with any AI model. Tell Copilot, for example, “Run brenda_agent.md using style-guide.md.” You can run the full workflow from start to finish, or jump into a specific stage—Teacher, Information Architect, Author, Reviewer, or Committer—depending on the task you’re working on.

Your workflow as a writer is:

- Run the agent files
- Answer yes/no questions from the agent
- Review the drafts it produces
- Approve or edit the content
- Decide when to commit a PR

The agent does everything else automatically.

## What the agents do

Once the agent is installed into your project, it takes you through the entire documentation workflow.

You can run it in its entirely, or run a specific stage of it.

### Teach you about the product
Helps you quickly understand a new product area by explaining its purpose, concepts, terminology, user journeys, workflows, and system behavior, while flagging uncertainties before moving on to the next stage.

###  Determine what needs documenting
It scans changes or your whole repository to understand what should be added or updated.

### Create your documentation structure
It builds folders, section index pages, and introduction pages based on the context.

###  Write new docs or edit existing documentation
The agent can draft:
- Get started pages
- Setup guides
- Configuration pages
- Concepts
- Task/guide documentation

You can choose to generate all pages or just specific sections.

###  Review your documentation
It checks:
- internal links
- folder structure
- formatting
- style rules

###  Prepare a pull request
It writes the PR title and summary, increments version numbers, and prepares changes for you to review.

