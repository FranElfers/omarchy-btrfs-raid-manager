---
name: commit
description: Create git commits. Use ONLY when the user explicitly prompts "commit".
---

# Git Commit Guidelines

1. **Trigger:** Commit ONLY on explicit user prompt `"commit"`. NEVER commit automatically.
2. **Identity:** Use the global OS Git identity (`git config --global user.name` / `user.email`). Do not override author credentials.
3. **Format:** Follow **Conventional Commits 1.0.0**:
   - `<type>(<scope>): <description>`
   - Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`.
   - Imperative, lowercase, no trailing period.
