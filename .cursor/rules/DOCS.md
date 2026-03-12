
# AI toolkit

The AI toolkit helps you use AI to write better documentation, faster.

It’s a starter kit packed with ready-to-use prompts and proven methods. You can either automate the toolkit directly in your code or simply copy and paste the instructions and prompts into your favorite web-based AI.

## Who is this toolkit for?

This toolkit is designed for engineers and technical writers who create documentation using our supported editors, agents, and web-based AI tools. You should be familiar with docs-as-code workflows and understand how to add context files to AI agents.

The toolkit works best if you have familiarity with:

- **Documentation workflows**: Writing and maintaining technical documentation as part of software development processes
- **AI agents**: Using GitHub Copilot, Cursor, or web-based AI tools like Gemini for content creation
- **Context management**: Providing relevant files and instructions to AI agents to improve output quality

## Set up the Docs AI toolkit

{{< section withDescriptions="true" >}}


# AI tools

The Docs AI working group currently supports OpenAI and Google web agents, Visual Studio Code GitHub Copilot, and Cursor. We've also provided information on other popular tools.

For our recommendation on why you shouldn't use API-based AI tools as they're extremely expensive, refer to the [agent billing documentation](billing.md).

Choosing the right AI tool for your documentation work, whether a web-based or an "as-code" tool like Cursor, isn't about which one is universally "better." Instead, the best choice depends entirely on the specific task at hand.

Each type of tool has distinct strengths, and knowing when to use one over the other will make your workflow more efficient and your output more effective.

## Web-based AI tools

Web-based AI tools such as ChatGPT, Claude, Gemini, Google AI Studio, and others are great solutions for working on content with minimal or no setup. You can log in and use the UI to easily create distinct projects, where you can also add files, text snippets, and direct the agent to URLs when you build your prompts.

Choose a web-based AI for tasks that are primarily about writing, brainstorming, and transforming text, especially when the context lives outside your code editor.

They are an excellent fit for:

- High-level ideation and structuring when you're planning content, not yet writing it.

- Rewriting and repurposing content when you have existing text that needs to be changed for a new audience or purpose.

- Working with external sources when your source material isn't in your code repository.

For non-technical contributors when the person writing isn't a developer and doesn't work in a code editor.

Use our [`AGENTS.md`](https://github.com/grafana/docs-ai/blob/main/AGENTS.md) file with your preferred web AI agent.

The `AGENTS.md` file contains our agent instructions including role, Grafana products, and writing style guide.

There is also a [`DOCS.md`](https://github.com/grafana/docs-ai/blob/main/DOCS.md) file with all the documentation for the Docs AI toolkit.

These files make it easy to add context to your queries.

## As-code based AI tools

AI-as-code tools like GitHub Copilot and Cursor integrate directly into your code editor, giving them the full context of your repository. This allows them to help you generate code, explain complex files, and make changes across your entire project with high accuracy.

Choose an as-code AI tool when your task requires a deep understanding of your codebase and the ability to read or modify files directly.

They are an excellent fit for:

- Generating code examples when you need a code snippet that uses your project's specific libraries and conventions.

- Explaining code when you highlight a function and ask, "Explain what this code does so I can document it."

- Refactoring and updating when a change requires updating text or code across multiple files in your repository.

- Drafting content in situ when you are writing documentation directly alongside the code it describes.

Follow the [setup documentation](./set-up.md) to use the AI-as-code instructions with your project.

## Copilot

You can give GitHub Copilot in VS Code a set of custom rules using instruction files. These files teach Copilot to follow your team's specific coding style and guidelines.

To set up GitHub Copilot in VS Code, follow the [official documentation](https://code.visualstudio.com/docs/copilot/setup).

This is a good choice if you're already using Visual Studio Code and like a docs-as-code workflow. You can easily reference files for context, including images with Gemini models, and create custom instruction and prompt files. You can select from a list of different LLMs like Sonnet 4, Gemini 2.5, GPT-4.1, and o4.

For more information about Visual Studio Code GitHub Copilot, refer to the [Copilot Customization documentation](https://code.visualstudio.com/docs/copilot/).

## Cursor

Cursor supports custom instructions through Project Rules that provide persistent, reusable context for code generation and editing.

[Download](https://cursor.com/) and install Cursor.

It's also a good choice for those who like a docs-as-code workflow. It's a fork of VS Code that has a simpler UX and support for multiple cloud LLMs and custom models. Due to licensing, some Visual Studio Code extensions aren't available in Cursor.

For more information about Cursor, refer to the [Cursor Rules documentation](https://docs.cursor.com/).

## Claude Code

Claude Code is a terminal agent that supports custom instructions through memory files that guide AI behavior for coding tasks.

This is an advanced solution for technical users who like to work in the terminal. Unlike Visual Studio Code GitHub Copilot and Cursor, you only have access to Anthropic's models and no image support. Currently at Grafana we don't have subscription access to Claude.

For more information about Claude Code, refer to the [Claude Memory documentation](https://docs.anthropic.com/en/docs/claude-code/).

## OpenAI Codex CLI

OpenAI Cortex is another terminal agent that supports custom instructions through agent files that guide AI behavior for coding tasks.

This is an advanced solution for technical users who like to work in the terminal. Unlike Claude Code, with a ChatGPT Pro subscription, you can use OpenAI Cortex without going through API billing, which makes it significantly more cost-effective. With the release of GPT-5, it's a strong terminal agent to consider.

## Zed

Zed is a fast code editor with minimal UX and built-in agent that supports their AI subscription, cloud models via API access, custom models, and Gemini CLI and Claude Code as first party agents via [Agent Client Protocol](https://agentclientprotocol.com/overview/introduction) (ACP).

Zed supports many of the other editor agent instruction formats and works with the Docs AI toolkit.

## Other VS Code forks

There are many other Visual Studio Code forks like Cursor. They tend to offer subscription-based access for more affordable AI usage. Like other VS Code forks, they don't have access to all extensions.

Popular forks include:

- Windsurf
- Void

## Other API-based tools

There are many other tools, most require you to use them via API access, which is extremely expensive. For more information on why you should avoid using tools via API billing, refer to the [API billing documentation](billing.md).

Popular tools include:

- Cline: VS Code extension
- Roocode: VS Code extension
- Opencode: terminal agent
- Gemini CLI: terminal agent
- Zed: editor, that supports API agents

## Other tools

We don't currently have documentation to support asynchronous agents such as Google's Jules or OpenAI's Codex (different from Codex CLI).

We don't currently have documentation to support Warp terminal.


# Agent billing

At Grafana we have access to some agents through subscription models, for example, Google's Gemini and AI Studio, ChatGPT and OpenAI Codex CLI, GitHub Copilot, and Cursor.

We don't currently have subscription support for Claude and therefore Claude Code. To use Claude Code and many other agents with cloud models you need to provide API access.

API access is significantly more expensive.

Think carefully about whether the value you get from these agents justifies the cost. It's a matter of $20 versus thousands of dollars.

We highly recommend that you use agents with a subscription model. Many times you get access to the same models and request-based versus token-based billing that favors a more considered workflow with our instruction files, prompts, and planning complex tasks.


## AI-as-code setup

For GitHub Copilot, Cursor, and other code editors.

Defaulting to a Copilot Docs AI toolkit specific instruction file to not overlap with other agent files.

### First time set up

Copy the [`toolkit.instructions.md`](https://github.com/grafana/docs-ai/blob/main/.github/instructions/docs/toolkit.instructions.md) file to your repository, rename as appropriate for agents other than GitHub.

### Updating toolkit instructions

Replace the existing toolkit instruction file with the new one.

### Custom instruction files

Put custom instructions in their own instruction files.

## Web-based AI setup

For Gemini, Google AI Studio, ChatGPT, and other web interfaces.

### Files needed

- [`toolkit.instructions.md`](https://github.com/grafana/docs-ai/blob/main/.github/instructions/docs/toolkit.instructions.md) - Core agent instructions
- [`DOCS.md`](https://github.com/grafana/docs-ai/blob/main/DOCS.md) - Complete toolkit documentation

### Setup steps

1. Open your web-based AI tool
2. Upload or paste the `toolkit.instructions.md` file content
3. Add the `DOCS.md` file if you need toolkit documentation
4. Prompt the agent with your documentation task

## Next steps

- Read the [best practice workflows](workflows/_index.md)
- Review the [agent instructions](instructions/_index.md)
- Check the [AI Policy](https://wiki.grafana-ops.net/w/index.php/AI_Policy_FAQ)
- Request premium access: [GitHub Copilot](https://helpdesk.grafana.com/support/catalog/items/332) or [Cursor](https://helpdesk.grafana.com/support/catalog/items/379)


# Glossary

This glossary defines key terminology and concepts used in the Docs AI repository.

## Agent

Software systems that use AI to pursue goals and complete tasks on behalf of users. AI agents demonstrate reasoning, planning, and memory, with a level of autonomy to make decisions, learn, and adapt. Unlike AI assistants, which respond to user prompts, agents can perform tasks autonomously.

For more information, see the [Google Cloud definition](https://cloud.google.com/discover/what-are-ai-agents?hl=en).

See also: [Agentic](#agentic), [Mode - Agent](#mode---agent).

## Agentic

Describes the capability of a system to act autonomously without requiring a prompt.

See also: [Agent](#agent).

## AI Studio

A platform from Google that provides access to multiple AI models with customizable settings and parameters, such as "Grounding with Google Search". AI Studio offers more capabilities than the standard Gemini interface.

For more information, see [Google AI Studio](http://aistudio.google.com) and the [AI Studio Quickstart](https://ai.google.dev/gemini-api/docs/ai-studio-quickstart).

See also: [Gemini](#gemini).

## Anthropic

The company that develops Claude and the Claude family of AI models.

See also: [Claude](#claude), [Claude Sonnet](#claude-sonnet).

## ChatGPT

A generative AI chatbot developed by OpenAI, powered by the GPT (Generative Pre-trained Transformer) family of models. ChatGPT can be used to interact with various GPT models, including GPT-4o and GPT-5.

For more information, see [ChatGPT](https://chatgpt.com/).

See also: [OpenAI](#openai), [LLM](#llm).

## Claude

A family of AI models developed by Anthropic. Claude also refers to the chat interface provided by Anthropic for interacting with these models.

For more information, see [Claude](https://claude.ai/new).

See also: [Claude Sonnet](#claude-sonnet), [Anthropic](#anthropic).

## Claude Sonnet

A specific family of models within the Claude series, developed by Anthropic. Available versions include Claude Sonnet 4 and Claude Sonnet 4.5.

See also: [Claude](#claude), [Anthropic](#anthropic).

## Context

Resources that guide an AI chatbot's responses to prompts or an agent's work. Context can include documentation, images, transcripts, and instructions.

For more information, see [Manage context for AI](https://code.visualstudio.com/docs/copilot/chat/copilot-chat-context).

See also: [Context folder](#context-folder), [Context window](#context-window).

## Context folder

A folder in an integrated development environment (IDE) that provides context for an AI chatbot or agent. Context folders help keep resources organized and easily accessible.

See also: [Context](#context).

## Context window

The amount of information a generative AI model can process and recall during a session, measured in tokens. Longer context windows enable models to process and use more data.

For more information, see [What is long context and why does it matter for your AI?](https://cloud.google.com/transform/the-prompt-what-are-long-context-windows-and-why-do-they-matter) (Google Cloud Blog).

See also: [Token](#token), [Context](#context).

## Cursor

An AI-enhanced code editor based on Visual Studio Code (VS Code). Cursor provides similar functionality to VS Code with GitHub Copilot integration, along with additional AI capabilities.

See also: [GitHub Copilot](#github-copilot).

## Gemini

Google's AI chatbot and virtual assistant, formerly known as Bard. Gemini Code Assist is a related product that integrates with IDEs to perform code generation and completion tasks.

See also: [AI Studio](#ai-studio).

## GitHub Copilot

An AI coding assistant developed by GitHub. GitHub Copilot integrates with IDEs, command-line interfaces through the GitHub CLI, mobile chat interfaces, and the GitHub website. The GitHub Copilot coding agent can be initiated by assigning an issue to Copilot or through [other methods](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#introduction).

## LLM

Large Language Model. A type of AI program that can recognize and generate text, among other tasks. LLMs are trained on large datasets.

For more information, see [What is a large language model?](https://www.cloudflare.com/learning/ai/what-is-large-language-model/) (Cloudflare).

See also: [Model](#model), [ChatGPT](#chatgpt).

## MCP

Model Context Protocol. An open-source standard for connecting AI applications to external systems. MCP enables AI applications like Claude or ChatGPT to connect to data sources (such as local files and databases), tools (such as search engines and calculators), and workflows (such as specialized prompts).

For more information, see [Model Context Protocol](https://modelcontextprotocol.io/docs/getting-started/intro).

## Mode - Agent

The mode in which an AI tool behaves autonomously to complete an assigned task.

See also: [Modes](#modes), [Agent](#agent).

## Modes

Different operational modes available in AI tools. For example, Visual Studio Code offers Agent, Plan, Ask, and Edit modes.

See also: [Mode - Agent](#mode---agent).

## Model

A system that learns patterns from training data (examples) and applies that learning to new situations. AI models form the foundation of AI applications and services.

See also: [LLM](#llm).

## NotebookLM

A Google tool that summarizes and analyzes uploaded resources. NotebookLM provides citations from source materials when responding to prompts.

## OpenAI

The company that develops ChatGPT and the GPT family of large language models.

See also: [ChatGPT](#chatgpt), [LLM](#llm).

## Token

The smallest unit a model can process, such as a portion of a word, an image segment, or a video frame. Tokens are the fundamental building blocks used to measure and process information in AI models.

See also: [Context window](#context-window).

## Warp

A terminal emulator with integrated AI features, available for multiple platforms.


# Agent instructions

Think of agent instructions as a style guide for your AI coding assistant.

They're different from a prompt: the instruction is your set of reusable rules, while the prompt is the specific task you ask for right now.

You combine the two, providing the standing instructions and your specific prompt to get helpful and consistent answers every time.

The Docs AI Toolkit keeps these instructions organized in a central .docs/instructions/ folder, making them easy to find and use.

## What makes a good instruction

A good instruction has these qualities:

- **General**: Widely applicable rather than specific use cases
- **Concise**: Uses clear, direct language without unnecessary words
- **Positive**: Write "do this" instead of "don't do that"
- **Actionable**: Provides specific guidance that can be immediately applied

## Current instructions

We have the following agent-agnostic instructions for:

- Grafana technical writer role
- Grafana naming
- Style guide

## Future work

Incorporate other generally applicable instructions from user feedback.


# Agent prompts

Think of a prompt as a specialized tool for a specific job, while an instruction is your general-purpose user manual.

A prompt is a set of directions you activate on-demand for a particular task, like drafting release notes. This is different from a general instruction, which provides the AI with your core, always-on rules.

By keeping these specialized prompts separate, you use the model's processing power (tokens) more efficiently and get better, more focused results.

The Docs AI toolkit establishes a convention of storing these prompts in a .docs/prompts/ folder, keeping them ready to be used when needed.

<!-- ## Future work

Include prompts for specific use cases, for example:

- Index and overview articles
- Introduction articles
- Setup articles
- Configuration sections
- Troubleshooting docs
- Release notes -->


# AI prompts inventory, tips, tricks

This page captures small additions, prompts, and tips/tricks that are helpful but not substantial enough to warrant a full workflow document. Contributions are welcome!

- **Prompts**: Quick prompts you can use for specific tasks. Add new prompts to the table below with a brief description.
- **Tip/trick**: Useful tips or tricks related to AI prompting or usage. Add new tips/tricks to the second table with a short explanation.

Please keep entries concise. If your contribution is more substantial, consider creating a dedicated workflow document instead.

| Prompt                                                                             | Description                                                                                                     |
| ---------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `Identify any accessibility issues in this documentation and suggest improvements` | Use this to check documentation for accessibility concerns like unclear headings or missing alt text            |
| `Don't make assumptions. Ask for clarification.`                                   | Add to prompts to encourage AI to seek clarification rather than making assumptions when encountering ambiguity |

| Tip / trick                           | Description                                                      |
| ------------------------------------- | ---------------------------------------------------------------- |
| Use keyboard shortcuts for efficiency | For example, use `Ctrl+F` to quickly find text in documentation. |


# Prompts for creating scripts

Some tasks are resource-intensive for an AI agent, but very fast if completed by a script.
In these cases, it makes more sense to ask an AI agent to _create a script for you_ than to have the agent carry out the task itself.
Tasks suitable for a script include:

- Find and replace content that spans multiple lines
- Format and organize content into a specific order
- Delete specific content

{{< admonition type="tip" >}}
Think of tasks that you could do mindlessly but that would take a long time.
These are great for a script and terrible for AI agents.
{{< /admonition >}}

This page covers how to generate and use scripts with the help of an AI agent, as well as example scenarios.
Each of the examples on this page cover the following:

- A scenario in which a script is useful.
- An example of a prompt to generate the script. These prompts are formatted so that you can easily copy, update, and paste them into your IDE.

## Generate and use scripts

To generate and use a script, follow these steps:

1. Provide the context for the agent.

   Provide robust context rather than a single test file. If you're going to run the script on a folder, provide the folder as context so the agent can analyze all its contents when it's generating the script. This helps the agent generate a script that covers all the possible scenarios it'll encounter. If the script doesn't work properly at first, the agent can quickly revert the work and start over with more guidance.

1. Prompt the agent with a detailed description of what the script should do.

   {{< admonition type="tip" >}}
   It can be helpful to provide an example of the result you want as part of your prompt.
   {{< /admonition >}}

1. (Optional) Share the script in the #docs-platform channel so it can be reviewed to ensure it's doing what you intend.
1. Follow the agent's prompts for it to run the script, or run it yourself in Terminal.
1. Review the results.
1. If needed, ask the agent to fix the script with clear directions about what it's done incorrectly.
1. Delete the script so that it's not added to the repository when you commit your changes.
1. Save (if needed), add, and commit your changes.

   {{< admonition type="note" >}}
   If you don't make any changes beyond what the script does, you might not need to save the changes, just add and commit them.
   {{< /admonition >}}

1. (Optional) Include the script in your PR description or as a comment so reviewers can verify its intent if they wish.

If you forget to delete the script before committing your changes, make sure to go back and delete it so that you don't add it to your repository.

## Scenarios

Following are some scenarios where scripts have been useful, along with the prompts to generate the scripts, and the resulting script.

### Fix front matter

In this scenario, there's front matter that's not formatted correctly:

```Markdown
tags:
  - Business Input
title: 'Business Input'
description: 'Learn how to store and emulate data in Grafana using the Business Input data source plugin.'
labels:
  products:
    - enterprise
    - oss
weight: 10
```

#### Prompt

```text
Create a script for me that will do the following tasks to fix the front matter of the files in the sources folder:

- Make all the tags lowercase
- Replace the word `tags` with `keywords`
- Remove quotation marks from the `title` and `description` fields
- Add `cloud` to the list of product labels
- Put the front matter in the following order: title, menuTitle, description, keywords, labels, weight
- Remove anything that's not title, menuTitle, description, keywords, labels, weight
- Leave a line between the front matter and the H1 heading of the file
```

### Update link type

In this scenario, a number of ref URIs need to be replaced with URLs.

#### Prompt

```text
Create a script to do the following for the file in context:

- Find all instances of links using the `(ref:)` format and replace them with full URLs
- The full URLs should begin with `https://grafana.com`
- Complete the URLs with the first destination option listed for that ref in the front matter of the file
- For example: `(ref:about-users-and-permissions)` should be replaced with `(https://grafana.com/docs/grafana/<GRAFANA_VERSION>/administration/roles-and-permissions/)`
```

### Replace a multi-line element with a shortcode

All of the following scenarios take a multi-line element that's not formatted for Hugo, repurposes a property, and formats it using a shortcode.

- [Admonition](#admonition)
- [Figure](#figure)

#### Admonition

In this scenario, there's informational text that needs to be formatted as an admonition.
For example:

```text
:::info Version

The tooltip feature is supported starting from version 3.5.0.

:::
```

##### Prompt

```text
Create a script to do the following for the files in the sources folder:

- Find all instances of the `::info` tag and replace it with the `{{</* admonition */>}}` ... `{{</* /admonition */>}}` shortcode, taking into account that the tag spans multiple lines
- Delete any text on the same line as the `::info` tag
- In the shortcode, use `note` as the value for the `type` parameter
```

#### Figure

In this scenario, there are image files that have been moved from their original location into our media folder, so file paths need to be updated.
Also, the format of the references need to be updated.
For example:

```text
<Image
  title="Display Pie Charts based on the data from the Static Data Source."
  src="/img/plugins/business-input/dashboard.png"
/>
```

##### Prompt

```text
Create a script to do the following for the files in the sources folder:

- Find instances of the `<Image />` tag and replace it with the `{{</* figure */>}}` shortcode, taking into account that the tag spans multiple lines
- Format the shortcode as follows: `{{</* figure src="" alt="" */>}}`
- Use the `src` parameter from the tag in the shortcode, but replace everything before the image name with `/media/docs/grafana/panels-visualizations/business-forms/`
- Use the value from the `title` parameter in `alt`
- Don't make this change if the src value is a .gif file
```

The last line of the prompt is to account for the fact that we use the video-embed shortcode for screen recordings.

{{< admonition type="tip" >}}
You could update this prompt to use regular markdown notation for images or to update references for any kind of media, such as a `youtube` or `video-embed` shortcode.
{{< /admonition >}}


# AI best practice workflows

Docs are important. Technical content is often the first introduction customers get to our tools, products, and features, and also the first place they go to troubleshoot when things go wrong or they hit a barrier in Grafana products.

Usually the term best practice refers to a studied and verified approach to a task or discipline that has been created by experts in that field. In the case of AI and documentation, there really aren't any experts yet. The best practices listed below are from the personal and professional experience of those of us who have been using the tools and have determined what works best through experimentation over a period of months to a year. These approaches are not definitive, but they're effective. We offer our experiences as a jumping-off point, and as you evolve your own use of AI tools and workflows, we'll also evolve this guidance and the best practices will get better.

{{< section withDescriptions="true" >}}


# Create scenario documentation from customer calls

Customer calls are a goldmine of documentation insights, but manually extracting their value is a slow and difficult process. AI is the perfect tool to solve this, rapidly analyzing hours of conversation to surface the key pain points and the real world use cases that should drive our content.

This best practice covers how to use AI to turn customer calls into authentic scenarios and best practice content.

## Customer calls

Customer research, support calls, voice of the customer calls, and community calls are all examples of valuable user feedback. You can obtain transcripts either by feeding in audio or video, or copy/pasting the transcript directly into your AI tool.

As a first task, always prompt the agent to anonymize the content by scrubbing personal identifying information. This is important for data privacy, compliance, and security. You can then start to ask it to summarize the key points, focus on customer goals, their main challenges, and any workarounds.

### Prompts

```markdown
Act as a data privacy specialist. Review the following customer call transcript and anonymize it. Replace all personal names with generic roles (e.g., 'the customer,' all company names with 'xxx'). Output the result as a TXT file.
```

```markdown
Summarize this customer call transcript. Identify the user's primary goal, the three main pain points they encountered, and the final resolution.
```

{{< admonition type="note" >}}
For added value, you can ask for a sentiment analysis.
{{< /admonition >}}

## Multiple sources

You can also feed the agent multiple transcripts and use AI to identify common themes and patterns. It can tell you which are mentioned most frequently, helping you prioritize what to document.

Start by anonymizing each transcript as above.

### Prompts

```markdown
Attached are five transcripts from customer calls about xxx. What are the most common questions and points of confusion across all of them?
```

Once you have the key painpoints, you can ask AI how the PM or engineers responded to them and ask it to write the first draft of a best practices or scenario based document.

```markdown
Based on the summary of the call, write a scenario document for a typical use case the customer would use xxx for. Use the persona of Sara, the Admin, use the company name Innovate Inc, use the team Developers in "Platform Engineering" team at Innovate Inc. and walk through the problem and the steps to solve it.
```

{{< admonition type="note" >}}
For better results, consider providing a template or example of how scenarios are documented at Grafana. This helps guide the AI to produce content that matches your preferred style and structure.
{{< /admonition >}}

```markdown
Create a best practices guide based on the solutions identified in this call transcript. Organize it using clear headings.
```


# Use Copilot to generate draft docs from new feature source code

Use this workflow when a developer has updated source code that introduces a new feature but documentation hasn't been written yet. This workflow generates draft concept and task topics using GitHub Copilot to analyze code changes and apply Grafana's documentation standards.

## Prerequisites

Before you begin, ensure you have the following:

- VS Code with GitHub Copilot extension installed and configured
- Access to the feature branch containing the new code
- Claude Sonnet 4.0 model selected in Copilot settings
- Write access to the target documentation directory
- The AGENTS.md file with our style guidance has been added to the repo you're working in

## Generate draft documentation

To use Copilot to generate draft docs from new feature source code, complete the following steps:

1. In VS Code, check out the feature branch.

1. In Copilot Chat mode, run the following command: `git diff origin/main`

   {{< admonition type="note" >}}
   Do not use this prompt in the command line. Add the prompt to Copilot.
   {{< /admonition >}}

1. Instruct Copilot to analyze the changes.

   Copilot provides a summary of what’s changed in the source code.

1. Prompt Copilot to review and become familiar with the Writers’ Toolkit.

   For example: `You are a technical writer at Grafana Labs. Review and understand the Writers’ Toolkit.`

   Copilot reviews the contents of the Writers’ Toolkit and provides a summary. This prompt frontloads Copilot with our approach to docs and prepares it to write a concept topic and task topic.

1. Switch from Chat mode to Agent mode by clicking the **@** symbol in Copilot Chat.

1. Add the target documentation directory as context:
   1. Right-click the documentation folder where you want to create the new files
   1. Select **Add to Chat Context** or drag the folder into the Copilot Chat window

   This step ensures Copilot knows the file structure and where to place the generated documentation.

1. Write the following Copilot prompt:

   `Use your understanding of the Writers’ Toolkit guidelines and based on the changes to the code that introduces [feature name], propose a concept topic that you should write for end users who will use this new feature.`

   Copilot adds a draft concept topic file to the specified directory.

1. Write the following Copilot prompt:

   `Use your understanding of the Writers’ Toolkit to write a task topic that explains to users how to use [feature name]. The task topic should include an introduction and step-by-step instructions.`

   Copilot adds a draft task topic file to the specified directory.

1. Review and adjust content as required and push your changes for review.

1. Review the front matter to ensure it aligns with the guidelines set forth in the Writers' Toolkit.


# Edit source content for style and grammar

This workflow helps you use AI agents to improve the grammar, style, and consistency of existing documentation while maintaining your content's technical accuracy and voice.

## Before you begin

Before you begin, ensure you have the following:

- Access to an AI agent (GitHub Copilot, Cursor, or web agent)
- The source content files you want to edit
- Understanding of your content's technical context

## Workflow overview

Follow this process for effective AI-assisted editing:

1. **Prepare your context** - Load style guides and source files
2. **Start with analysis** - Have the agent identify issues first
3. **Review suggestions** - Validate recommendations before applying
4. **Apply edits incrementally** - Work section by section for complex documents
5. **Verify technical accuracy** - Ensure AI hasn't changed meanings
6. **Test examples** - Confirm code samples and procedures still work

## Copy edit source content

Provide clear and simple instructions by specifying the action to perform, such as "copy edit to our style guide." Indicate the scope of the edit, for example, "this file" for the current context, "all the attached files" for multiple files, or reference a specific file using `@file_name`. Clearly state the expected outcomes, such as "improve grammar and fix spelling."

With the following prompt, the agent will edit the file and create a diff in the editor:

```markdown
Copy edit this file according to the Grafana style guide, improve grammar, and fix spelling.
```

### For larger documents

Break large documents into sections to maintain quality:

```markdown
Copy edit the "Installation" section of this file for grammar, clarity, and adherence to our style guide. Focus on improving sentence structure and consistency.
```

### For technical accuracy

When editing technical content, emphasize preserving accuracy:

```markdown
Copy edit this configuration guide for style and grammar while preserving all technical details, code examples, and parameter names exactly as written.
```

## Discuss improvements

You can also discuss the improvements and fixes before editing. The agent will give you a list of areas to address. You can then prompt it to fill in gaps or leave out certain actions. Finally, when you're happy, you can ask it to make its suggested changes.

```markdown
Discuss how we can improve the grammar and fix spelling errors while adhering to our style guide.
```

## Advanced editing techniques

### Structural improvements

For content that needs reorganization:

```markdown
Review this article's structure and suggest improvements to the heading hierarchy, paragraph organization, and information flow while maintaining all technical content.
```

### Voice and tone consistency

To align content with your brand voice:

```markdown
Adjust the tone of this document to be more conversational and user-friendly while maintaining technical accuracy. Focus on using active voice and second person perspective.
```

## Common issues and solutions

### AI changes technical terms

**Problem**: The agent modifies product names, API endpoints, or technical terminology.

**Solution**: Use more specific prompts:

```markdown
Copy edit for grammar and style only. Do not change any product names, API endpoints, code examples, or technical terminology.
```

### Overly formal language

**Problem**: AI makes content too formal or corporate.

**Solution**: Specify your desired tone:

```markdown
Edit this content for clarity and grammar while keeping a friendly, approachable tone. Use contractions and conversational language appropriate for developers.
```

### Loss of context

**Problem**: AI suggestions don't account for the broader document context.

**Solution**: Provide more context in your prompt:

```markdown
This is a troubleshooting guide for system administrators. Copy edit for clarity while maintaining the step-by-step instructional format and preserving all warning callouts.
```

## Next steps

After completing your style and grammar edits:

- Test any code examples or procedures mentioned in the edited content
- Share drafts with subject matter experts for technical review
- Consider running the content through additional AI workflows for structure or accessibility improvements
- Update related documentation that might reference the edited content


# Use AI to create and maintain Grot Guides

Grot Guides are interactive onboarding experiences that help users make decisions and quickly reach the documentation that supports their specific use case.

Because Grot Guides encode complex user journeys and branching logic, AI is especially effective at helping you design, validate, and maintain them.

This workflow explains how to use AI to model user journeys, generate Grot Guide YAML, and iteratively improve both logic and language.

## What is a Grot Guide?

A Grot Guide is an interactive decision-making process that asks users a small number of targeted questions and then routes them to the most relevant documentation.

Instead of browsing or searching, users answer questions and are guided to setup guides, integrations, or reference material that matches their goals.

For an example, refer to the [Infrastructure Observability Grot Guide](https://grafana.com/docs/grafana-cloud/monitor-infrastructure/#guidance-and-help) in the Grafana Cloud documentation.

### Underlying format

Grot Guides are defined using YAML. The YAML describes:

- The welcome screen.
- Question screens and answer options.
- Branching logic between screens.
- Result screens that link to documentation.

Because this format is structured but verbose, AI is well-suited to generating and validating Grot Guide definitions.

## Before you begin

Before you begin, ensure you have the following:

- A clearly defined product area or onboarding goal.
- Existing documentation that the Grot Guide should link to.
- Access to subject matter experts (SMEs).
- An AI tool that supports long context and image uploads.
- At least one existing Grot Guide to use as a reference.

## Workflow overview

Use this workflow to create or update a Grot Guide with AI:

1. Model the user journey.
2. Corroborate the journey with SMEs.
3. Generate the Grot Guide YAML.
4. Test and refine the guide.

Each step builds confidence in both the logic and the user-facing experience.

## Step 1: Model the user journey

Start by using AI to understand the full user journey and all possible decision paths.

At this stage, focus on conceptual clarity rather than implementation details. Your goal is to identify:

- User goals.
- Questions you need to ask.
- Valid answer options.
- Documentation outcomes for each path.

### Provide this context to the AI

Share concrete inputs so the AI can model realistic paths:

- Product area or onboarding goal.
- Audience and primary user goals.
- Destination documentation and URLs you want to route to.
- Constraints, prerequisites, and non-goals.
- A reference Grot Guide YAML to mimic structure (for example, the Infrastructure Observability Grot Guide).
- Acceptance criteria for what a “good” journey includes.

### Example prompt

```markdown
Act as a senior technical writer and product thinker.

Help me model an onboarding decision tree for users who want to monitor databases in Grafana Cloud.

Identify:

- The primary user goals
- The questions we should ask users
- The answer options for each question
- The documentation outcomes each path should lead to

Do not write YAML yet. Focus on understanding the user journey.
```

Iterate until the journey feels complete and no major paths are missing.

## Step 2: Corroborate the journey with SMEs

Review the draft journey with SMEs to validate accuracy and completeness.

A mind map is especially effective for this step because it makes branching logic easy to review and discuss.

Best practices:

- Treat AI output as a starting point, not a final answer.
- Ask SMEs to identify missing paths or incorrect assumptions.
- Capture agreed changes directly in the mind map.

Once finalized, export or capture the mind map for use in the next step.

## Step 3: Generate the Grot Guide YAML

Feed the validated journey back into AI and ask it to generate the Grot Guide YAML.

Screenshots of the mind map work particularly well because they preserve hierarchy and relationships between decisions.

### Provide this context to the AI

Include the artifacts the model needs to produce correct, well-formed YAML:

- Mind map screenshots of the full journey.
- A reference Grot Guide YAML (for example, the Infrastructure Observability Grot Guide) to use as a structural example.
- Front matter template to reuse.
- Result link map (result screen ids to target documentation URLs).
- Naming conventions for `screen_id` values.
- Tone and style guidance (Grafana Style Guide).

### Example prompt

```markdown
I’m creating a Grafana Cloud Grot Guide.

Attached is a mind map that represents the full user journey.
A reference Grot Guide YAML is attached as a structural example.
Use it to generate a complete Grot Guide YAML file.

Requirements:

- Follow the structure used by existing Grafana Cloud Grot Guides.
- Include welcome, question, and result screens.
- Ensure all paths are reachable.
- Include the same front matter used by the Infrastructure Observability Grot Guide.
- Use clear, concise, user-facing language.
- Use the attached reference Grot Guide YAML as a structural reference.
```

Review the generated YAML for:

- Missing or broken `screen_id` references.
- Inconsistent naming.
- Ambiguous question wording.
- Results that don’t clearly map to documentation outcomes.

## Example: Grot Guide YAML excerpt

The following excerpt shows a simplified Grot Guide flow with a welcome screen, a question, and a result. It illustrates the structure AI should generate, not a complete guide.

```yaml


# Iterative content planning with AI

This workflow helps you create high-quality documentation through collaborative planning and iterative refinement with AI agents, rather than expecting perfect output from a single prompt.

Zero or one-shot prompts, such as simply asking "Write documentation for this feature," often lead to suboptimal results. These prompts set vague expectations, lack necessary context or constraints, and typically produce generic content that's difficult to refine or tailor to specific needs.

Iterative refinement allows you to improve content step by step, rather than aiming for perfection from the start. By planning collaboratively and incorporating your expertise, you ensure that each stage focuses on validated approaches. This method enables easy adjustments throughout the process, supporting more effective and adaptable execution.

## Before you begin

Before you begin, ensure you have the following:

- Access to an AI agent (GitHub Copilot, Cursor, or web agent)
- Source materials (code, screenshots, existing docs, requirements)
- Understanding of your target audience and their needs
- Time to work through multiple iterations (typically 3-5 rounds)

## Workflow overview

Follow this process for effective iterative planning:

1. **Define clear actions** - Specify what you want the AI to analyze or create
2. **Set scope and context** - Provide relevant materials and boundaries
3. **State expected outcomes** - Define what success looks like
4. **Discuss before executing** - Review suggestions and refine direction
5. **Plan collaboratively** - Create detailed plans together
6. **Iterate and improve** - Refine through multiple rounds

## Focus on actions and outcomes

Structure your initial prompts using clear actions, specific context, and defined outcomes. This approach ensures productive collaboration with AI agents and reduces the need for multiple clarification rounds.

Give clear and simple instructions:

- **Action to perform**: "analyze and suggest use case topics"
- **On what**: "this codebase and these application screenshots" (for current context) or @folder_name
- **Expected outcomes**: "3 use case topics to document"

### Basic action prompt

```markdown
Analyze this codebase and these application screenshots and suggest 3 use case topics to document.
```

### For targeted analysis

Focus the AI's analysis on specific user needs:

```markdown
Analyze this API documentation and suggest 3 tutorial topics for developers new to observability who need to implement monitoring in under 2 hours.
```

### For content gap identification

Direct the AI to identify specific documentation needs:

```markdown
Review this existing documentation and user feedback, then suggest 3 content gaps that would reduce the most common support requests.
```

## Discuss before doing

The agent will suggest priorities and identify gaps. You can then ask for more details on specific sections, request alternative approaches, or clarify requirements or constraints.

### Discussion prompt

```markdown
Where do these use cases fit into the user's onboarding journey, how easy or complex are they?
```

### For alternative approaches

Get different perspectives on the same problem:

```markdown
Suggest 2 different content strategies for this feature: one focused on quick implementation and another for comprehensive understanding.
```

### For requirement clarification

Refine scope and constraints:

```markdown
Which of these topics would require input from the engineering team versus what we can document with existing resources?
```

## Plan the project together

The following prompt gives you a structured approach, an opportunity to review before full execution, and control over scope and direction.

### Planning prompt

```markdown
Create the first draft of a detailed plan for the X use case topic we can discuss and iterate on.
```

### For comprehensive planning

Get detailed project breakdowns:

```markdown
Create a content plan for [topic] including outline, required resources, timeline, and review checkpoints.
```

### For iterative development

Plan content in phases:

```markdown
Break this documentation project into 3 phases: basic implementation, advanced features, and troubleshooting. Provide a plan for each phase.
```

## Use version control

The result of iterative development doesn't have to be a single commit.
Commit often and with meaningful messages to help you undo undesirable changes.
Meaningful commits also help reviewers understand the iteration you went through to produce the final changes.

## Advanced techniques

### Content validation

Test your plans before execution:

```markdown
Review this content plan and identify potential issues with user flow, missing prerequisites, or technical accuracy concerns.
```

### Integration planning

Connect new content with existing documentation:

```markdown
Analyze how this new content fits with our existing documentation structure and suggest the best integration points.
```

### Feedback incorporation

Use existing user feedback to guide planning:

```markdown
Based on these support tickets and user feedback, prioritize which aspects of this feature need the most detailed documentation.
```

## Common issues and solutions

### Vague requirements

**Problem**: Planning produces generic or unfocused suggestions.

**Solution**: Add specific constraints and context:

```markdown
Plan documentation for users migrating from [specific tool] to our platform, focusing on the differences in workflow and configuration.
```

### Scope creep

**Problem**: Plans become too ambitious or complex.

**Solution**: Set clear boundaries:

```markdown
Focus this plan on the core workflow only. Advanced configurations and edge cases will be separate documentation.
```

### Missing user perspective

**Problem**: Plans don't address real user needs.

**Solution**: Include user context explicitly:

```markdown
Plan this content for developers who have never used observability tools before and need to understand both concepts and implementation.
```

## Next steps

After completing iterative planning:

- Begin content creation using your detailed plan as a guide
- Schedule regular check-ins to validate progress against user needs
- Test content structure with team members before full execution
- Document lessons learned to improve future planning processes


# Develop learning journey

This process uses Copilot/Cursor to research, outline, and structure a new learning journey, ensuring compliance with our internal guidelines.

## Before you begin

Before starting the workflow, ensure you have the following tools and context loaded:

- Tool: Use Copilot/Cursor in **Ask** mode.
- Model: Select the current Claude Sonnet Large Language Model (LLM).
- Environment: Ensure you are working within the `website` repository.
- Be familiar with the [learning journey writing guidelines](https://wiki.grafana-ops.net/w/index.php/Product/Documentation_and_technical_writing/Learning_journeys).

## Generate the learning journey proposal

1. Add `.github/learning-journey/learning-journey-agent.md` as context to Copilot or Cursor.

1. Enter the following prompt: `Use the attached instruction file to help me write a learning journey.`

Follow the prompts provided by Copilot/Cursor.

- You will be asked which feature to focus on.
- You will see a list of learning journeys with a description and a list of milestones
- You will be asked if you'd like to see a more detailed outline for any of the proposed learning journeys.

The detailed outline is ideal for sending to stakeholders for review as it gives them a really good sense of what's covered in the learning journey.

The detailed outline should look similar to the following example:

```
Milestone 2: Navigate to RCA Workbench (weight 200)
Content Structure:
H1 heading: "Navigate to RCA Workbench"
Introduction (1-2 paragraphs): Explains that RCA Workbench is accessed directly through the Grafana Cloud navigation menu and why systematic incident investigation is important for reducing MTTR
Stem sentence: "To access RCA Workbench, complete the following steps:"
Numbered steps:
1. Sign in to your Grafana Cloud environment, for example mystack.grafana.net.
1. On the Grafana Cloud home page, open the navigation menu on the left side of the screen.
1. Click RCA Workbench (or Asserts > RCA Workbench depending on menu structure).
1. Observe that the RCA Workbench interface loads, displaying the main investigation dashboard.

Verification note: "The RCA Workbench main interface appears, showing available investigation tools and recent activity."

Transition sentence: "In the next milestone, you'll initiate an investigation by clicking an alert annotation."
```

## Review and development (human-in-the-loop)

This phase requires human expertise to validate the AI-generated outline before development begins.

1. Secure Sign-off: Copy and paste the final learning journey proposal into a Google Doc and send it to the domain expert (a developer or Product Manager).
   - Goal: Ask for comments and suggestions to improve the proposal and potentially refine the input we use for future AI-driven journey development.

   {{< admonition type="note" >}}
   The domain expert should be familiar with the purpose of a Learning Journey. If they need context, set up a discussion and feel free to include Chris for additional support.
   {{< /admonition >}}

1. AI development: Once the outline is finalized and approved in the Google Doc, return to Copilot/Cursor to build the learning journey files.
   1. Select Directory: Choose the `learning-journey` directory in the website repo so the AI can build the files in the correct location.

   1. Switch Mode: Select **Agent** mode.

   1. Enter this prompt:

      `I want you to write a learning journey and I will provide you with the milestones one at a time.`

   1. To ensure proper processing and reduce the chances AI drifts from its task, copy and paste content from the Google docs _one milestone at a time_.

      Copilot/Cursor will process the request and automatically generate the necessary files for the learning journey, starting with the landing page.

   1. After Copilot/Cursor builds the draft, enter the following prompt:

      `Update the learning journey landing page to take into account all milestones you have written.`

      Copilot/Cursor adds the landing page first, but some of the content on that page depends on the milestones, which haven't been written yet. This prompt trues up the landing page to take into account all milestones.

## Post-processing prompts (link validation)

After the draft is built, use AI to correct common issues, such as broken links in side journeys. Copilot/Cursor hallucinates and creates side journey links that don't exist.

1. Add Context: Provide the AI with the `side-journey-verification.md` file as context.

1. Validation Prompt: Instruct the AI to:

"Use the attached side-journey-verification.md file to test the links you added to the side journey. Where appropriate, add a valid link to replace the incorrect link."

This ensures that the final output is accurate and adheres to established internal documentation standards.


# Create new product or feature documentation

Creating documentation for a new feature from scratch can be a time-consuming process. AI can accelerate this workflow by taking your raw notes, product plans, and style guides and generating a complete, well-structured first draft.

This best practice covers how to use AI to turn source materials into a high-quality initial draft for either a new feature or product.

## Prompt

```markdown
I have uploaded three files: a style guide, a document template, and my raw notes.

Act as an expert technical writer. Your task is to generate a complete first draft of the documentation based on my notes using the attached style guide and template.

Audience: This document is for [admins].

Goal: After reading this document, [an admin] should be able to [xxx].

Scope & Structure: Define the feature or product. Add an introduction, how it works, and a workflow section. Create task topics for how users would use the product or feature.

[Adapt this section according to the type of documentation you are writing. Product DNAs, screenshots help with context. The more context, the better the initial draft.]

Tone: Keep the tone friendly and encouraging.
```


# Continuous quality evaluation using AI personas

Set up a continuous feedback loop to keep your docs improving by using AI agents that act like a team of reviewers with different specialties.

These agents help spot and fix issues by checking your docs against important quality criteria for different types of users.

Instead of waiting for occasional human reviews, you get ongoing, consistent, and scalable feedback from AI agents, each with their own focus.

Your virtual review team has three main roles:

## The Critic

To analyze the fundamental quality of the writing. It acts as a meticulous peer reviewer, checking for clarity, conciseness, tone, and structural integrity.

### Prompt

```markdown
Act as a senior technical writer. Review the attached document for clarity, conciseness, and adherence to our style guide. Identify any confusing sentences, undefined jargon, or structural issues.
```

## The Questioner

To determine if the documentation answers the questions users are likely to have. It reads a document and generates a list of questions that a curious user would ask.

### Prompt

```markdown
Based on the attached document about 'Adaptive Logs', generate 10 distinct questions a new user is likely to have. The questions should cover what the feature is, why it's useful, and how to get started.
```

## The User

To test if a specific user type can succeed with the documentation. You create multiple flavors of this agent, each modeled on one of your key user profiles (e.g., beginner, admin, etc.). This agent is given the questions from "The Questioner" and tries to find the answers in the documentation.

### Prompt

```markdown
Act as a beginner user who is not a developer. I will provide a document and a list of 10 questions. For each question, state whether you found a 'Clear Answer', 'Partial Answer', or 'No Answer' within the provided text. If you found the language too technical to understand, note that.
```

You can track the results in a spreadsheet and use them as benchmarks. If the success rate falls below 80%, you can flag an area as a candidate for improvement and open an issue or discuss priorities with the PM.


# Develop release notes

Release notes provide a curated list of the most important changes for a new release of a versioned product. The PRs included in the release notes are a subset of the CHANGELOG full list. The CHANGELOG is the authoritative source of all changes for a release.

Release notes are primarily used for versioned products like the database products, Mimir, Tempo, Loki, and Pyroscope. Individual feature releases may be announced in a "What's new" entry and included in the release notes.

## Background on the release notes process

The process outlined in this document is based on the Tempo release notes.
Each new major version has a comprehensive CHANGELOG, release notes that provide a curated list of changes that impact users, and a blog post that highlights the top three to five features.

When creating Tempo release notes, we use the following process:

1. Curate a subset of CHANGELOG entries that have user impact to include in release notes. Coordinate with your team to determine the most important changes.
1. Group these entries into categories, for example, Features, Enhancements, Bug fixes, Upgrade considerations, and (sometimes) Security updates. These features are further grouped by topic, for example, TraceQL metrics and performance improvements.
1. Identify the top 3-5 capabilities that are the featured changes in the release. These high visibility features are also featured in the release blog post and video.

After you prioritize the entries, you can create the release notes.

1. For each entry, use the GitHub MCP to look up the PR and provide a concise summary (no more than two to three sentences) of the change. Where appropriate, add an example.
1. Use the headings in the sample to group the entries.
1. Validate the changes to the documentation with the code changes.
1. Correct linting errors, remove any leftovers from the CHANGELOG (like user names).

## Before you begin

Before starting the workflow, ensure you have the following tools and context loaded:

- Tool: Use your favorite AI IDE. Steps in this example use Cursor in Plan mode.
- Model: Select the Claude Sonnet 4.0 Large Language Model (LLM) or ChatGPT 5.
- GitHub MCP server enabled in Cursor or VS Code to allow the AI to read PRs and files. Refer to [Using the GitHub MCP server](https://docs.github.com/en/copilot/how-tos/provide-context/use-mcp/use-the-github-mcp-server).
- Access to the codebase and PRs. Database/backend projects often lack UI screenshots; authoritative details come from PR descriptions, diffs, and code.
- Style/linting setup (for example, Vale) to catch tone and style issues.
- Example release notes to use as a template. For example, for Tempo, use `/tempo/docs/sources/tempo/release-notes/v2-9.md`.

## Workflow overview

1. Start with a concise, outcome‑focused prompt that includes a small "sample" section to anchor structure and tone. Provide a sample of the release notes to use as a template.
1. Expand with a production prompt that instructs the AI to include PR links, groupings, upgrade considerations, and examples.
1. Iterate with follow‑ups to ensure all PRs are covered, add examples, and place breaking changes into "Upgrade considerations."
1. Validate via MCP lookups: open PRs and code to confirm details and capture configuration or migration steps.
1. Polish with sentence‑case headings and run your linter.

### Input sources to curate first

- CHANGELOG unreleased section for the upcoming version, or
- A curated list of PRs you want included in the release notes.

## Create initial draft for curated PR list

Use this when you have a curated list of PRs and a sample of the release notes to use as a template.

If you are using a curated list of PRs, paste the entire PR list under "Sample" or reference the PR list as a source. You can also use the unreleased or version section of the CHANGELOG as a source.

When you run this prompt using Plan mode, the AI creates a plan that you can review and update before executing.

```markdown
As an experienced technical writer and tracing expert, generate a release notes document using the information included in this prompt.

Each entry has a PR number and link. Use the GitHub MCP server to look up more information about the PR and provide a concise summary (no more than 2–3 sentences) of each change. Where appropriate, add an example.

Use `/tempo/docs/sources/tempo/release-notes/v2-6.md` as an example of what to produce.

The next release is vX.Y.

You need to include the PR number and the link to the PR for each entry at the end of the entry.

Use the headings in the template: @/tempo/docs/sources/tempo/release-notes/v2-6.md.

Sample:
Features (feature in blog post)

[FEATURE] Add MCP Server support. [#5212](https://github.com/grafana/tempo/pull/5212)
[FEATURE] Add query hints sample=true and sample=0.xx which can speed up TraceQL metrics queries by sampling a subset of the data to provide an approximate result. [#5469](https://github.com/grafana/tempo/pull/5469)

[New Parquet v5 (experimental) - what’s coming up next](https://github.com/grafana/tempo/issues/4694) Not ready to be run. In active development.
[FEATURE] New block encoding vParquet5-preview1 with low-resolution timestamp columns for better TraceQL metrics performance. This format is in development and breaking changes are expected before final release. [#5495](https://github.com/grafana/tempo/pull/5495)
[FEATURE] New block encoding vParquet5-preview2 with dedicated attribute columns for integers. This format is in development and breaking changes are expected before final release. [#5639](https://github.com/grafana/tempo/pull/5639)
Operational Improvements - also in blog - 2nd
Set of improvements for people who are running multi-tenant situations. These improvements also help with
[CHANGE] Do not count cached querier responses for SLO metrics such as inspected bytes. [#5185](https://github.com/grafana/tempo/pull/5185)
[CHANGE] Adjust the definition of tempo_metrics_generator_processor_service_graphs_expired_edges to exclude edges that are counted in the service graph. [#5319](https://github.com/grafana/tempo/pull/5319)
Changes that help people with multi-tenants – help improve trace quality and the data that they get out
[ENHANCEMENT] Add counter query_frontend_bytes_inspected_total, which shows the total number of bytes read from disk and object storage [#5310](https://github.com/grafana/tempo/pull/5310)
[ENHANCEMENT] Add histograms spans_distance_in_future_seconds / spans_distance_in_past_seconds that count spans with end timestamp in the future / past. While spans in the future are accepted, they are invalid and may not be found using the Search API. [#4936](https://github.com/grafana/tempo/pull/4936)
[ENHANCEMENT] Add support for scope in cost-attribution usage tracker. [#5646](https://github.com/grafana/tempo/pull/5646)
[ENHANCEMENT] Improve logging and tracing in the write path to include tenant info. [#5436](https://github.com/grafana/tempo/pull/5436)
[ENHANCEMENT] Measure bytes received before limits and publish it as tempo_distributor_ingress_bytes_total. [#5601](https://github.com/grafana/tempo/pull/5601)
```

Tips

- Keep the "Sample" short; the goal is to lock formatting and tone.
- If needed, ask the model to continue; see the Iteration checklist.

### Iterate on the initial draft

After you have the initial output, iterate to improve structure, correct issues, validate output, and make sure all entries have the PR number and link. Use this to enforce structure, links, upgrade considerations, and cross‑references to docs.

For Tempo, use an existing release notes file, for example,`/tempo/docs/sources/tempo/release-notes/v2-6.md`, as a template; for other products, use the equivalent release notes file in your repository.

The prompts in this section provide an example of how to review and iterate the output.

```markdown
Put the breaking changes into an "Upgrade considerations" section, similar to v2-6.md.

Look in `docs/sources/tempo` for any documentation for the MCP server. Add a doc link to that section of the vX.Y release notes.

Looking at https://github.com/grafana/tempo/pull/5495, is there a configuration option that needs to be set to enable this feature? Include the answer in the release notes if applicable.

Apply the [Writer's Toolkit](https://grafana.com/docs/writers-toolkit/). Use sentence case for headings. Fix linting issues.
```

Optional follow‑ups to drive accuracy and completeness

- "Continue and summarize the remaining PRs (list any missed by number)."
- "Group entries under Features, Enhancements, Bug fixes, Upgrade considerations."
- "Add a minimal example where helpful."
- "Verify configuration flags or migrations by opening the linked PRs and changed files."
- "Validate the description of <FEATURE> using the codebase and PR <NUMBER>."

## Use a CHANGELOG as a starting point

Use this when your CHANGELOG has the authoritative unreleased section and you don't have a curated list of PRs.

```markdown
As an experienced technical writer, generate release notes using the information in the `main / unreleased` section of `CHANGELOG.md`.

Each entry has a PR number. Use the GitHub MCP server to look up more information about the change and provide a concise summary (no more than 2–3 sentences) of each change. Where appropriate, add an example.

Use `/tempo/docs/sources/tempo/release-notes/v2-6.md` as a template.

The next release is vX.Y.
```

### Create release notes for an API

This longer prompt is useful when you create release notes for a product with an API/migration focus.

```markdown
You are an expert technical writer and developer advocate. Draft a changelog for developers, summarizing all changes in the last release.

Instructions:

1. For each change (added/removed/modified endpoint, schema, parameter, or response), write a clear, concise, actionable summary.
2. For API changes, include HTTP method and path and specify exactly what changed.
3. Explicitly call out breaking changes.
4. Mention motivation/context where useful.
5. Analyze commit diffs to extract developer‑impacting changes and group related changes.
6. Link to relevant docs/issues/code where possible and recommend follow‑ups.
7. Use bullet points under clear headings; highlight breaking items; keep entries specific.
```

## Suggestions for improving the output

Here are some suggestions for improving the output.

### Iteration checklist

- **Coverage**: Did the AI process all PRs? If not, ask it to continue and list missed PRs explicitly.
- **Grouping**: Are items organized (Features, Enhancements, Bug fixes, Upgrade considerations)?
- **Links**: Every item includes the PR number and link.
- **Examples**: Add short examples where they clarify usage.
- **Breaking changes**: Migrated to "Upgrade considerations" with steps and flags.
- **Doc cross‑links**: Add links to relevant docs for new capabilities (for example, MCP server docs).
- **Style**: Sentence‑case headings; run Vale/linters.

### When to add supporting docs

If a change introduces a new metric, flag, or configuration:

- Ask the AI (with MCP) to locate the exact code or docs, then draft a short page or subsection.
- Place the content in the most findable location (for example, a metrics page under Monitoring) and cross‑link from the release notes.

### Common follow‑up prompts

- "Scan the linked PRs and include any configuration flags users must set."
- "Move any breaking changes into 'Upgrade considerations' and provide migration steps."
- "Add a one‑line example for the new metric/query/flag."
- "Fix linting issues and convert headings to sentence case."

### Notes

- Backend/database projects may not have screenshots; rely on PR descriptions and code via MCP.
- Some models summarize only the first few PRs by default; explicitly request processing of all remaining PRs.
- Keep the sample short to steer format without overwhelming the model.


# Continuous feedback and sentiment analysis

Focus your attention and improve the quality of documentation by directly acting on high-signal, actionable feedback from the communities where your users are having organic conversations.

You can structure your prompt to do two or more things. The prompt below lets you set up the following alert cadences:

1. **A focused daily alert**: Create a primary daily alert that targets a specific list of high-value sites where your users are most active. This gives you your core, actionable feedback.
1. **A broad weekly, monthly, or quarterly alert**: Keep a secondary, broader search that runs weekly. Its purpose is to catch mentions from new blogs or forums you might want to add to your primary list.

```markdown
Set up a daily, weekly, monthly, or quarterly alert to search (site:reddit.com/r/grafana OR site:reddit.com/r/devops OR site:stackoverflow.com OR site:community.grafana.com) for new conversations from [the past 24 hours] about Grafana's [Adaptive Telemetry] products.

For each conversation, perform a detailed analysis on:

- Overall sentiment
- Usability and UX feedback
- Documentation feedback

Summarize the findings and flag any urgent issues.
```


# Content optimization and SEO with AI

To make your documentation more easily findable by AI, you must shift your thinking from traditional keyword SEO to a more holistic approach focused on semantic context and structured data.

AI doesn't "read" your website like a human. Instead, systems like Retrieval-Augmented Generation (RAG) break your documentation into smaller "chunks," convert them into numerical representations of their meaning (embeddings), and store them in a database. When a user asks a question, the AI finds the most semantically relevant chunks to construct its answer.

## Key principles for high retrievability

We do a lot of this already, but there are some improvements we can look at as people shift to using AI as the main starting point for searching for information.

What we do well:

- Consistently use a logical heading structure (H1 for the main title, followed by H2s for major sections, then H3s for sub-sections)
- Write clear, unambiguous language
- Interlinking between related topics

Things we could do better:

- High quality metadata
- Atomic pages

## Use high quality metadata

AI aims to identify the most relevant and trustworthy topic that precisely addresses the user's question. By enhancing our documentation with effective SEO practices and comprehensive metadata, we provide clear signals that help AI systems accurately assess and surface the best content for each query.

### Add better descriptions

Clear, detailed descriptions bridge the gap between a user's question and your feature's solution. Well-crafted descriptions boost the AI's confidence in your content, making it more likely to be selected as a reliable source. To improve your metadata:

- **Include a review date:** Signals the topic’s freshness and reliability to both AI and human readers.
- **Specify the intended audience:** Helps AI tailor responses to the appropriate knowledge level.
- **Define relevant personas:** Adds context, enabling AI to provide answers that align with different user roles and goals.

## Create shorter, atomic pages

Dividing lengthy documentation into concise, atomic topics,each centered on a single concept, improves AI retrieval, even if your heading structure is already well-organized.
| | **Long page with good headings** | **Short, focused pages** |
| :--- | :--- | :--- |
| **AI retrieval confidence** | Moderate to high. The AI can identify a relevant section using headings and extract the associated content, but may be less certain about the overall page relevance. | Very high. The AI recognizes that the entire page directly addresses the user's query, allowing it to return the whole document as a highly relevant answer. |

## Use AI as your SEO consultant

You can use AI to act as your SEO consultant and give feedback on how to optimize your content for AI.

### Prompt

```markdown
Act as an SEO specialist reviewing this documentation topic in Grafana. My primary target keyword is "Adaptive Logs."

Analyze the following topic (set of topics) and provide feedback on these four points:

- Keyword usage: Is the primary keyword present and used naturally in the main title (H1) and at least one subheading (H2)?
- Meta description: Write a compelling, SEO-friendly meta description (under 160 characters) that includes the primary keyword.
- Heading structure: Is the heading structure logical (i.e., no H3s without an H2 above it)?
- Readability: Are there any sentences that are overly long or complex that could be simplified for better readability?
```


# Use AI for fewer (and shorter) demos and meetings with SMEs

Use AI to shorten meetings with your SMEs to make your interviews more productive by handling the preparation and analysis. The key is to use AI before the interview to do your homework and prepare targeted questions, and after the interview to extract key information and action items.

## Before the interview

Use AI to analyze all the existing source material (product dna, design docs, engineering notes, even code comments).

### Prompt

```markdown
Act as a technical writer. You´re preparing to interview an engineer about the new 'xxx' feature. Attached is the Product DNA, design doc, and some engineering notes. Based on this, generate a list of 5-7 specific, technical questions I should ask to understand how to document this feature for a developer audience. Focus on potential points of confusion or gaps in the existing material.
```

## After the interview

Record the interview and use AI to process the transcript. Pull the most important information, structure it, and identify any remaining questions.

### Prompt

```markdown
Analyze this interview transcript. My goal is to understand the configuration process. Extract the key, step-by-step instructions the SME provided into a numbered list. Create a list of any action items, so I can follow up asynchronously.
```


# Use Grafana Assistant to help with documentation

Grafana Assistant is an AI-powered tool integrated into Grafana that can help technical writers draft, refine, and update documentation. It has access to the official Grafana documentation, can search for relevant information, and can help structure content in the appropriate format.

Using Assistant, you have access to an AI that is embedded in the context of Grafana products.

Use Grafana Assistant to:

- Update procedures: Refresh outdated workflows with current steps by asking Assistant to compare tasks and screenshots with the current UI.
- Create examples: Generate code samples and configuration examples.
- Create initial drafts: Create first drafts of new documentation pages or sections.
- Research existing content: Find related documentation and ensure consistency.
- Create multi-step examples: Prompt Assistant to explain a workflow step by step.

For more information about the Assistant, refer to the [Grafana Assistant documentation](https://grafana.com/docs/grafana-cloud/machine-learning/assistant/).

## Before you begin

Before you begin, you need to have access to a Grafana Cloud instance with Grafana Assistant enabled.
The Grafana Ops instance has the latest version of Assistant and the latest versions of Grafana products.

## Workflows

You can use Assistant as you would any AI tool to help you write documentation. The major advantage is that it's embedded in the context of Grafana products, so the concepts, procedures, and examples are higher quality.
Caveats:

- The Assistant can't directly modify Markdown files.
- Consider working with the Assistant and your editor together. Use Assistant to evaluate and produce content and then use your editor to format and edit the content.

### Create new documentation

Start with a clear prompt. For example, this prompt creates documentation for a topic written in the Grafana documentation style. Update this prompt to fit your specific needs.

```markdown
As an experienced technical writer, create documentation for [topic]. Use @[RESOURCE_NAME] as a template.
Include:

- An overview section
- Before you begin
- Step-by-step instructions
- Examples
- Troubleshooting tips

Write this in Grafana documentation style:

- Use sentence case for headings
- Include code blocks with appropriate language tags
- Use numbered lists for procedures
- Use bullet points for feature lists
- Keep tone professional but approachable
```

Iterate and refine the draft. At some point, move your draft to your editor. Assistant can't directly modify Markdown files.

Example:

```markdown
As an experienced technical writer, create documentation for how to configure the Tempo data source. Use https://grafana.com/docs/grafana/latest/datasources/tempo/configure-tempo-data-source/ as a template.

Include these sections:

- An overview section
- Before you begin
- Step-by-step instructions
- Examples (if applicable)
- Troubleshooting tips (if applicable)

Write this in Grafana documentation style:

- Use sentence case for headings
- When code blocks are needed, include appropriate language tags
- Use numbered lists for procedures
- Use bullet points for feature lists
- Keep tone professional but approachable
```

Next, ask the Assistant to validate the documentation against the UI by pointing to a feature or UI element for comparison. In this case, the prompt refers to a specific Tempo data source.

```markdown
Verify the steps you created against the Tempo data source, for example, @tempo-test
```

### Assess doc accuracy

Assistant can assess accuracy by comparing the documentation to the current UI, identifying out of date information, and providing feedback on how to address any identified problems.

You can refer to specific features by name or use `@` to select a specific Grafana feature, data source, or other capabilities from a pop-up menu.

Example: Compare UI elements to published docs with screenshots.

```markdown
Are the screenshots in this doc up to date with the latest Traces Drilldown UI? Compare the Traces Drilldown UI to the screenshot on this page: https://grafana.com/docs/grafana/latest/visualizations/simplified-exploration/traces/ui-reference/
```

Example: Validate documentation against the UI.

```markdown
Validate this page against the Tempo data source (for example, @tempo-ops-test-lbac ) Documentation to evaluate: https://grafana.com/docs/grafana/latest/datasources/tempo/configure-tempo-data-source/
```

You can ask Assistant to perform updates to the documentation, with the caveat that you'll have to transfer the updated content to your editor.

### Create a walkthrough

You can ask Assistant to create a walkthrough of a workflow by providing a description of the workflow and the UI elements that need to be clicked.

In this example, there was a major update to the Traces Drilldown UI. The Get started with Traces Drilldown documentation needed to be updated to reflect these changes.
Here are the steps used to update the walkthrough:

1. Navigate to the Traces Drilldown in a Cloud instance with Assistant enabled.
2. Ask Assistant to identify any current errors and explain how to address them. Select an error to explore that users might encounter.
3. Perform the investigation as guided by Assistant.
   - In your editor, open the Markdown file and make updates using some of the explanation text provided by Assistant.
   - Use the AI tools in your editor to help automate the updates.
   - After each update, ask Assistant to validate the changes against the current UI.
4. Ask Assistant to clarify any concepts or steps.
5. Validate the steps yourself.

