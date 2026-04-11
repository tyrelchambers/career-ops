# Mode: tracker — Application Tracker

Read and display `data/applications.md`.

**Tracker format:**
```markdown
| # | Date | Company | Role | Score | Status | PDF | Report |
```

Possible statuses: `Evaluated` → `Applied` → `Responded` → `Interview` → `Offer` / `Rejected` / `Discarded` / `SKIP`

- `Applied` = the candidate submitted their application
- `Responded` = a recruiter/company reached out and the candidate replied (inbound)
- `Interview` = actively in the interview process
- `Offer` = offer received
- `Rejected` = rejected by the company
- `Discarded` = discarded by candidate or offer closed
- `SKIP` = doesn't fit, don't apply

If the user asks to update a status, edit the corresponding row.

Show also statistics:
- Total applications
- By status
- Average score
- % with PDF generated
- % with report generated
