## Repository Skills

This repository defines reusable skill instructions in `.agents/skills/`.
Each skill is a subdirectory containing a `SKILL.md` file with detailed workflow instructions.

When the user invokes a command that matches a skill name (e.g. `/gvb-ghissue-to-speckit #239`),
read `.agents/skills/<skill-name>/SKILL.md` and follow its instructions completely before doing any work.

### Discovery

At the start of each session, scan `.agents/skills/` for available skill directories.
Do not maintain a hardcoded list — discover skills dynamically from the filesystem so that
additions, removals, and renames are picked up automatically.