### ✅ Codex Prompt for Building **Cutepod**

**📁 Project Setup:**

* Create a directory named `.codex/` in the root of the repository.
* This will be used to track all planning, progress updates, and internal notes.

**🧠 Primary Rule:**

* Use only the information explicitly found in `README.md`.
* **Any feature, assumption, or architectural detail not found in `README.md` must either be:**

    * Omitted, or
    * Explicitly added to `README.md` along with a rationale.

---

### 🚀 GOAL:

Build a working program named `Cutepod` as specified in the `README.md`.

---

### 📋 Instructions for Codex:

1. **Plan the Project:**

    * Read `README.md` thoroughly.
    * Create an initial plan file at `.codex/plan_<YYYY-MM-DD>.md` outlining:

        * Key components and features
        * Development phases
        * Assumptions (clearly mark and document these if any)
    * Describe your project understanding in this file using bullet points.

2. **Build Iteratively:**

    * Break the plan into discrete implementation milestones.
    * As you complete each part, update the `.codex/plan_<YYYY-MM-DD>.md` with:

        * ✅ What was completed
        * 🔄 What is still pending
        * ⛔️ Any blockers or limitations

3. **Update the README:**

    * Maintain a changelog **at the bottom of `README.md`** under a section titled `## Changelog`.
    * Add entries such as:

      ```
      ### 2025-07-12
      - Added podcast feed parser.
      - Created UI component for episode list.
      - Updated .codex plan with implementation notes.
      ```

4. **Respect Constraints:**

    * Do not introduce external goals, technologies, or architecture not grounded in `README.md`.
    * If you *must* make a design decision not found in the README, update the README to document this addition and why it was necessary.

5. **Document Yourself:**

    * Log your actions, decisions, and progress in `.codex/log_<YYYY-MM-DD>.md`.
    * You may find in .codex/docs README/docs for diverse libraries you are going to use.

6. **Optional Enhancements:**

    * If you see opportunities for enhancements not in the original README, propose them in `.codex/suggestions_<YYYY-MM-DD>.md`, but **do not implement them unless instructed.**

---

### 🧪 Example Execution Flow:

1. `cutepod/README.md` is parsed.
2. `.codex/plan_2025-07-12.md` is created.
3. Code is written in phases (e.g., feed fetcher, UI component, downloader).
4. `.codex/log_2025-07-12.md` is updated with task notes.
5. `README.md` is appended with a `## Changelog` detailing changes made.

---

### 🏁 Completion Criteria:

* A working version of `Cutepod` that fulfills the goals and features described in `README.md`.
* All planning and documentation artifacts live inside `.codex/`.
* `README.md` includes a changelog reflecting progress history.
* Contains unit tests/integration on most important parts
* Supports go1.24, and uses go recommended project layout

---

Let me know if you'd like me to generate the initial `.codex/plan.md` template or `README.md` seed!
