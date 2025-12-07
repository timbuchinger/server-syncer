# 🧩 Skills

Skills are reusable, domain-specific workflows that the agent can invoke to
produce higher-quality, more consistent outputs.  
Because this environment does **not** have native support for Claude Code
Skills, the agent must emulate them through instructions.

## How the agent should use Skills

- Each Skill includes a **name**, **description**, and **implicit activation
  criteria** ("use when").  
- The *use when* component tells the agent **exactly when** the Skill should be
  invoked.  
- When a Skill applies, the agent **must activate it automatically** — the user
  does *not* need to call the Skill by name.  
- Each Skill may also include **supporting files** (shell scripts, templates,
  examples, workflows) located in the Skill's directory.  
  - The agent should reference and apply the logic or patterns contained in
    those files when producing an answer.  
  - If a supporting file would normally be executed (e.g., a Bash helper
    script), the agent should **simulate its effects**.

---

## 📦 Available Skills
