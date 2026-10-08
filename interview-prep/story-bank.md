# Story Bank — Master STAR+R Stories

This file accumulates your best interview stories over time. Each evaluation (Block F) adds new stories here. Instead of memorizing 100 answers, maintain 5-10 deep stories that you can bend to answer almost any behavioral question.

## How it works

1. Every time `/career-ops oferta` generates Block F (Interview Plan), new STAR+R stories get appended here
2. Before your next interview, review this file — your stories are already organized by theme
3. The "Big Three" questions can be answered with stories from this bank:
   - "Tell me about yourself" → combine 2-3 stories into a narrative
   - "Tell me about your most impactful project" → pick your highest-impact story
   - "Tell me about a conflict you resolved" → find a story with a Reflection

## Stories

<!-- Stories will be added here as you evaluate offers -->
<!-- Format:
### [Theme] Story Title
**Source:** Report #NNN — Company — Role
**S (Situation):** ...
**T (Task):** ...
**A (Action):** ...
**R (Result):** ...
**Reflection:** What I learned / what I'd do differently
**Best for questions about:** [list of question types this story answers]
-->

### [0→1 / Product Ownership] JobWatchCanada: Zero to Viral
**Source:** Report #099 — PostHog — Product Engineer
**S:** Saw a gap: no real-time, geospatial visualization of Canadian job market data existed.
**T:** Build and ship a full product solo within weeks — from idea to production.
**A:** Chose Go for concurrency (parallel scrapers, geospatial pipelines), React+TypeScript frontend, PostgreSQL + self-hosted Pelias geocoding; built AI content scheduling and admin dashboard; deployed and launched.
**R:** 20K+ unique visitors in first 2 months with zero paid acquisition; national media coverage, radio, cited by government officials and politicians.
**Reflection:** Viral adoption happens when you solve a real, underserved need. The architecture mattered less than the insight about what the market was missing.
**Best for questions about:** "Tell me about a product you owned end-to-end", "Tell me about your most impactful project", "Tell me about a time you showed initiative", "Tell me about a side project"

---

### [User Research / Iteration] Data Mapper Research Cycle
**Source:** Report #099 — PostHog — Product Engineer
**S:** Data Mapper had high support ticket volume; root cause was unclear from the tickets alone.
**T:** Understand the real friction points through users, not assumptions.
**A:** Ran 10 structured user research interviews; synthesized recurring themes; rewrote feature specs; reprioritized the backlog around the actual friction points rather than the loudest tickets.
**R:** 50% reduction in support tickets; customer retention improved.
**Reflection:** The most valuable signal was what users *didn't* say in their tickets — they'd normalized workarounds. Direct conversation exposed that.
**Best for questions about:** "Tell me about a time you used data/feedback to make a decision", "Tell me about a time you talked to users", "Tell me about improving a product metric"

---

### [Bias for Action / Shipping Fast] Stripe Apps: 1-Day Feature
**Source:** Report #099 — PostHog — Product Engineer
**S:** Team lead scoped a feature as "too costly to build this week" during planning.
**T:** Ship it anyway within the sprint by finding the smallest viable version.
**A:** Broke down the spec into the minimum shippable slice; cut scope to what users actually needed (not what the spec assumed they needed); shipped in one day.
**R:** Delivered before the next planning cycle; the approach was adopted by the team for similar estimation situations.
**Reflection:** Estimates are often wrong because they scope for perfection. Shipping the 80% version and proving it's enough is itself a skill.
**Best for questions about:** "Tell me about a time you moved fast", "Tell me about a time you pushed back on a timeline", "Tell me about a time you used good judgment"

---

### [Writing / Influence Without Authority] Hosting Proposal
**Source:** Report #099 — PostHog — Product Engineer
**S:** The company was on a hosting stack with growing overhead and cost; no one had formally proposed an alternative.
**T:** Identify a better solution, build the case, and get org-wide adoption without being in a leadership role.
**A:** Researched 4 hosting alternatives; documented tradeoffs and migration path; wrote a structured proposal; presented to leadership.
**R:** Company-wide adoption; reduced infrastructure overhead for the entire organization.
**Reflection:** Writing is a forcing function for thinking. Documenting the decision revealed weaknesses in options I'd initially favored.
**Best for questions about:** "Tell me about a time you influenced without authority", "Tell me about a time you identified and solved a problem proactively", "Tell me about your communication style"

---

### [Performance / Interactive Data Tools] JobWatchCanada: 500K Points in the Browser
**Source:** Report #308 — Waabi — Software Engineer, Commercial Software (Tools team)
**Provenance:** source: cv.md
**S:** JobWatchCanada needed to show hundreds of thousands of geocoded job-market points in the browser, on top of a Go + React app over millions of records.
**T:** Keep pan/zoom smooth at that scale without a paid mapping service.
**A:** Rendered the layer with MapLibre GL; used clustering and tile optimization for the map, and data pagination, caching, and query optimization on the Go + PostgreSQL side so the API kept up.
**R:** 500,000+ geospatial data points rendered interactively with smooth pan/zoom at scale.
**Reflection:** The rendering budget, not the framework, was the real constraint. Measure first, then decide what the browser should never receive.
**Best for questions about:** "Tell me about a performance problem you solved", "Tell me about building a data-heavy UI", "Tell me about a tool you built for a specific set of users"

---

### [Infrastructure / Self-Hosting] Self-Hosted Pelias Geocoding on Kubernetes
**Source:** Report #308 — Waabi — Software Engineer, Commercial Software (Tools team)
**Provenance:** source: cv.md
**S:** JobWatchCanada needed to geocode and reverse-geocode hundreds of thousands of full and partial addresses; a hosted geocoding API at that volume was not viable for a solo product.
**T:** Run geocoding in-house, reliably, on infrastructure I already operated.
**A:** Deployed a self-hosted Pelias geocoding server with multiple dataset integrations, alongside the Kubernetes stack I maintain for the product.
**R:** Reverse-geocoded hundreds of thousands of addresses at scale as part of the data pipeline.
**Reflection:** Operating your own infrastructure teaches you exactly which parts are worth paying someone else to run.
**Best for questions about:** "Tell me about your cloud/infra experience", "Tell me about a time you chose build over buy", "Tell me about running something in production"

---

### [Data Pipelines / Normalization] Government Data Normalization Pipelines
**Source:** Report #308 — Waabi — Software Engineer, Commercial Software (Tools team)
**Provenance:** source: cv.md
**S:** Canadian Open Government datasets arrive in heterogeneous formats and schemas.
**T:** Turn them into clean, queryable public data the product could render and search.
**A:** Built parsing and scraping pipelines that normalized the heterogeneous datasets into a consistent schema in PostgreSQL.
**R:** Millions of Open Government data points processed into a clean, queryable dataset powering the live product.
**Reflection:** Schema normalization is the unglamorous bulk of any data tool; the UI is only as good as the shape of the data underneath it.
**Best for questions about:** "Tell me about working with messy data", "Tell me about building a pipeline", "Tell me about dataset curation"
