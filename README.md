# Workstations

A personal AI workspace for people who are not developers. You talk to Claude on your phone, tablet or computer, and it remembers: who you are, how you write, what is pending, what you decided. Your life is split into **workstations** (Inbox, Life Admin, Communication, Money and any others you need), each with its own instructions and memory, so Claude always knows the context.

The first conversation is an interview. Claude gets to know you, maps your routine, shows where it can take work off your plate, creates the workstations you need and does one real task for you before you leave.

## What you get

- **An interview that sets everything up.** It asks about your work, your home and your routine, and how you like to be answered, one question at a time. You can stop and continue on another device.
- **An action map.** Your recurring tasks, rated by what Claude can do alone, what it prepares for you to approve, and what it tracks for you.
- **Memory that follows you.** Say "remember that..." on the phone and it is there on the computer.
- **Your own voice.** Emails and messages are drafted the way you write, from samples you give it.
- **Always Works.** Claude checks before saying something is done: it re-reads what it saved, shows every draft before anything is sent, and cites a source and a date for prices and rules.
- **It learns from corrections.** When you correct it, it proposes a rule and follows it from then on.

## What you need

- A paid Claude plan (**Pro** or higher). Claude Code on the phone and on the web is not part of the free plan.
- A free **GitHub** account. It is where your files live, privately. You will not need to use it after setup.
- The **Claude app** on your phone and tablet, and a browser or the **Claude Desktop** app on your computer (Windows or Mac).

## Setup (about 15 minutes, once)

1. **Create a GitHub account** at [github.com/signup](https://github.com/signup) and turn on two-factor authentication with your phone.
2. **Make your own copy of this workspace.** On this page, click **Use this template**, then **Create a new repository**. Give it a name (for example `my-workspace`), choose **Private** and create it. Private matters: this is where your notes will live.
3. **Connect Claude to it.** On the computer, open [claude.ai/code](https://claude.ai/code) and sign in. When asked, connect GitHub and install the Claude app on GitHub. Give it access **only to the repository you just created**. ([Official guide](https://code.claude.com/docs/en/web-quickstart))
4. **Phone and tablet.** Install the Claude app, sign in with the same account, open the **Code** section and pick your repository.
5. **Say hi.** Start a new session on your repository and send any message. Claude will offer the setup interview. Doing the interview on the computer or tablet is easier, because there is some typing. You can also type `/onboarding` to start it.

### Helping someone else set it up

Do steps 1 to 3 with them, on their accounts. Install the app on their phone and tablet (step 4). Then let them do the interview alone. It is personal, and Claude asks for writing samples.

## Everyday use

| You say | What happens |
| :---- | :---- |
| "Buy dog food, call the plumber, Ana's birthday is on the 12th" | Saved to the Inbox in seconds and sorted into the right places |
| "Remember that I prefer short answers" | Saved, and it says where |
| "What do I have this week?" | Your dates and tasks, soonest first |
| "Reply to this email saying I can't make it on Tuesday" | A draft in your voice. Nothing is sent without your OK |
| (screenshot of a bank statement) "How did I do this month?" | Spending by category, with the insight first |
| "/weekly-review" | Inbox sorted, the next 7 days, overdue items, one new idea to save you time |
| "/new-workstation for my clients" | A new area set up just for that |

## How it works

```
my-workspace/
  CLAUDE.md          # the router: which workstation handles what
  MEMORY.md          # facts that span areas
  00_Resources/      # your profile, your voice, learned rules, the action map
  Inbox/             # each workstation: CLAUDE.md (how to work here),
  Life Admin/        #   MEMORY.md (what happened here) and
  Communication/     #   Resources/ (lists, trackers, logs)
  Money/
  .claude/           # rules, skills (/onboarding, /new-workstation, /weekly-review),
                     #   agents (life-mapper, researcher) and the save hooks
```

**Saving is automatic.** Every Claude session on the phone or the web works on its own copy. After each answer, a hook saves the changes and sends them to GitHub. A GitHub Action then merges every session into `main` within about a minute. The next session, on any device, starts from that. Notes written on two devices at the same time are merged line by line, so neither one is lost. You never see any of this. If a save fails, Claude tells you.

**Memory lives in files, not in Claude.** Claude's own memory does not travel between devices, so everything worth keeping goes into the `MEMORY.md` files and `00_Resources/`.

## Without GitHub (computer only)

Prefer not to use GitHub? Download this repository as a ZIP, unzip it into a folder on your computer and open that folder in the Claude Desktop app. Everything works the same on that computer, and the save hooks do nothing. To use it from the phone you then need [Remote Control](https://code.claude.com/docs/en/remote-control) or Dispatch, and the computer has to stay on. This mode is less tested.

## Status

Version 0.1. What has been tested so far:

- **Tested:** the save hooks and the merge to `main`, with 26 automated tests on real git repositories (concurrent edits from two devices, offline phone, a branch that cannot be merged, a race on `main`, among others) and a real run on GitHub Actions. Also a real Claude Code session on a Mac.
- **Not yet confirmed:** the same flow started from the phone and tablet apps, and the desktop-only mode on Windows. If you hit a problem, please [open an issue](../../issues).

## Contributing

Issues and pull requests are welcome. Anything executable comes with tests, written first:

```bash
cd tools
go test -race ./...                   # integration tests for the save hooks and the merge Action
SYNCSPEC_COVERAGE=1 go test ./...     # same, plus line coverage of the bash scripts (fails under 80%)
```

The rules Claude follows inside a workspace are in `.claude/rules/`. Always Works applies to them too: say what "done" means, then prove it.

## Credits

The idea of splitting a personal Claude workspace into workstations with their own instructions and memory was inspired by Jeff Su's [Cowork Toolkit](https://coworkacademy.ai/toolkit). Everything in this repository was written from scratch.

## License

[MIT](LICENSE)
