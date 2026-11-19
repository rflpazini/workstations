---
name: onboarding
description: Setup interview for a new workspace. Gets to know the user (work, home, routine, friction, tools, writing voice), maps where Claude can take work off their plate, creates their workstations and delivers a first win. Use on first use, when 00_Resources/profile.md says status not-configured, when onboarding.md has an unfinished step, or when the user asks to set up, restart or redo the setup.
---

# Onboarding

The goal is a workspace that already knows this person when the interview ends, plus one real task done for them before they leave.

## How to talk

- Speak the user's language. Warm, plain, no jargon: never mention git, files, folders, markdown or "skills".
- **One question per message.** Short. Offer multiple choice when the answers are predictable; the user can always answer freely.
- Every question can be skipped ("prefer not to say" is a fine answer). Health, family and money details are optional.
- The user may be on a phone: keep each message to a few lines.

## Save as you go

After each block, show a 2-3 line summary, ask "Did I get that right?", and once confirmed:
1. Write the facts into `00_Resources/profile.md` under the matching heading, in the user's language.
2. Set `step:` in `00_Resources/onboarding.md` to the next block number and list the blocks already done.

This is what lets the interview resume on another device. **On start, read `onboarding.md`: if a step is recorded, say where you left off and continue from there.**

## Blocks

1. **Welcome.** Explain in 3 lines: this is their private assistant that remembers; everything stays in their own private space; never paste passwords or card numbers. Ask what they'd like to be called and confirm their language. Fill `preferred-name` and `language`.
2. **Who they are.** City and time zone, who lives with them (people, pets), the devices they use.
3. **Work.** What they do, a typical week, who they work with, recurring deliverables, tools.
4. **Home and life.** House, family routines, hobbies, anything they manage for others.
5. **Yesterday.** "Walk me through yesterday, from waking up to going to bed." Ask one follow-up on the busiest part. This block finds the real tasks; give it time.
6. **Friction.** What takes too long, what they forget, what they postpone, what they dislike doing, what people keep asking them.
7. **Tools.** Email, calendar, messaging, notes, cloud storage, banks, apps they live in.
8. **How to answer.** Short or detailed, formal or casual, and anything Claude must never do. Put the "never" items in `## Never do`.
9. **Voice.** Ask them to paste 2-3 messages or emails they wrote (any topic). Fill `00_Resources/voice.md`: tone, greetings, expressions, what they avoid, plus the samples.

## Map the actions

Run the `life-mapper` agent with the full profile. It writes `00_Resources/action-map.md`. Present its result in a short message: the 3 quick wins and the workstations it suggests.

## Workstations

The workspace ships with Inbox, Life Admin, Communication and Money. Propose 1-3 more from the map (for their work, a hobby, health, travel, studies, a side business...). Create the ones they accept with `/new-workstation`, tailored to what they told you.

## First win

Pick quick win #1 and do it now, end to end, following the always-works rule. Show the result.

## Close

1. Set `status: configured` in the profile and `step: done` in `onboarding.md`.
2. Add a short "About the user" summary to the root `MEMORY.md`.
3. Leave a cheat sheet in their language, 5 lines max:
   - Dump anything into the chat, anytime; it goes to the Inbox and gets sorted.
   - Say "remember that..." and it's saved.
   - Ask for a weekly review on Sundays (`/weekly-review`).
   - Correct me whenever I get something wrong; I'll learn it.
   - Works the same on phone, tablet and computer.
