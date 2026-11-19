# Workspace

This folder is a personal AI workspace. Work is split into **workstations**: one folder per area of life, each with its own instructions (`CLAUDE.md`), memory (`MEMORY.md`) and reference files (`Resources/`). The user is not a developer. Never make them deal with git, files or folders unless they ask.

## First session

If `00_Resources/profile.md` says `status: not-configured`, greet the user warmly in their language (guess from their first message; Portuguese if unsure), explain in two sentences what this workspace does, and offer to start the setup interview with `/onboarding`. If `00_Resources/onboarding.md` shows an unfinished step, offer to continue from that step instead.

## Every session

1. Read `MEMORY.md` and `00_Resources/profile.md` before answering. Use what you find; don't announce it.
2. Pick the workstation from the Routing Map below. Read its `CLAUDE.md` and `MEMORY.md`, then only the resources its table says to read.
3. If nothing fits, use Inbox. If the same kind of request keeps arriving with no home, suggest a new workstation with `/new-workstation`.
4. Reply in the user's language (`language` in the profile). File headings stay in English; facts are written in the user's language.
5. Before writing anything on the user's behalf, read `00_Resources/voice.md`.

## Saving

Files save themselves: after every answer a hook commits and pushes, and GitHub merges everything into `main` within a minute. Never run git commands yourself. If a hook message says the changes are NOT saved, tell the user in one sentence. Never edit anything under `.github/`.

## Routing Map

| Workstation | Route here when the user... |
| :---- | :---- |
| Inbox | ...drops a loose thought, idea, reminder or task and doesn't want to think about where it goes |
| Life Admin | ...deals with tasks, appointments, bills, documents, errands or the house |
| Communication | ...needs to write, reply to or review an email or a message |
| Money | ...talks about spending, budgets, bank statements, bills or savings |

## Creating a workstation

Use `/new-workstation`. Every workstation is a folder with:

1. `CLAUDE.md` with these sections, in order: **Identity** (who you are here, what routes here, what doesn't), **Resources** (table: Resource, Read when...), **Workflow** (how the main task gets done and how to check the result), **Editorial Rules** (always opens with "Follow the voice in 00_Resources/voice.md").
2. `MEMORY.md` with the sections **Contacts** and **Key Decisions**.
3. `Resources/` with at least one starter file.

Then add a row to the Routing Map above. Folder names use plain letters, numbers and spaces only (no accents or symbols), so they work on Windows.
