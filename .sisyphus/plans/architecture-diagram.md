# Model Inference Platform — System Architecture Diagram

## TL;DR

> **Quick Summary**: Generate a 7-layer SVG architecture diagram (960×1400px) for the Model Inference Cloud Platform, based on Section 5.1 of the product spec document.
> 
> **Deliverables**: 
> - `model-inference-platform-architecture.svg` (960×1400px SVG)
> - `model-inference-platform-architecture.png` (1920px width PNG export)
> 
> **Estimated Effort**: Short
> **Parallel Execution**: NO - single task (sequential steps within)
> **Critical Path**: Task 1 → Validation → Export PNG

---

## Context

### Original Request
User invoked `/fireworks-tech-graph` with: `@模型推理云平台-产品方案.md 画出系统架构图`

### Interview Summary
**Key Decisions**:
- Diagram type: Architecture Diagram (7-layer microservice architecture)
- Source: Section 5.1 "总体架构分层" from product spec
- Style: Style 1 "Flat Icon" (default white background, `#2563eb` blue arrows)
- Output: SVG + PNG export at 1920px width
- ViewBox: `0 0 960 1400` (tall diagram with 7 layers + legend + key features)

**Research Findings**:
- 7 layers identified from doc:
  1. Access Layer (Web Console, APIs, SDK, CLI, 3rd Party)
  2. Gateway & Routing (API GW, Adapter, Auth, Rate Limit, Smart Router)
  3. Control Plane (User/Team, Billing, Endpoints, Model Registry, Load Balancer, A/B Testing, Model Eval, Prompt Templates, Budget Alerts, Data Quality, Semantic Cache, Fine-tune Scheduler, RBAC)
  4. Data Plane (MultiEngineRouter with vLLM/SGLang/Custom/Mock, Dedicated Endpoints, Batch/Fine-tune Clusters, Agentic RL)
  5. Orchestration (Volcano Scheduler, HAMi Plugin, K8s Native Components)
  6. Storage & Data (PostgreSQL, Redis, S3, ClickHouse, Kafka, LanceDB)
  7. Observability (Prometheus, Grafana, OpenTelemetry, Loki)

- Arrow flow colors per Style 1: Blue `#2563eb` (request flow), Green `#16a34a` (resource schedule), Purple `#9333ea` (data/async), Orange `#ea580c` (GPU cluster), Gray `#6b7280` (monitoring)

### Metis Review
**Identified Gaps** (addressed):
- None critical — diagram spec is clear from product document Section 5.1
- Auto-resolved: Layer dimensions calculated from document structure (140px per layer average, 7 layers = 980px + title 80px + legend 50px + key features 120px + footer 30px = ~1400px total height)

---

## Work Objectives

### Core Objective
Generate a production-quality SVG architecture diagram showing the 7-layer microservice architecture of the Model Inference Cloud Platform, with proper arrow flows between layers and internal component connections.

### Concrete Deliverables
- `model-inference-platform-architecture.svg` — valid SVG, 960×1400px ViewBox
- `model-inference-platform-architecture.png` — PNG export at 1920px width

### Definition of Done
- [ ] SVG file exists and passes `rsvg-convert -o /dev/null` validation
- [ ] PNG file generated at 1920px width
- [ ] All 7 layers present with correct components per Section 5.1
- [ ] Arrow flows show L1→L2→L3→L4→L5→L6→L7 with proper colors
- [ ] Internal arrows within L4 (Data Plane) show MultiEngineRouter → Dedicated → Batch/Fine-tune → Cross-domain
- [ ] Legend present (bottom-left) explaining arrow colors
- [ ] Key architecture features listed (bottom section)

### Must Have
- All 7 architecture layers from Section 5.1
- Proper arrow markers (arrow-blue, arrow-green, arrow-purple, arrow-orange, arrow-gray)
- Layer containers with dashed borders (`stroke-dasharray="4,2"`)
- Node boxes with shadow filter
- Chinese + English bilingual labels per document

### Must NOT Have (Guardrails)
- NO external `@import` in SVG (breaks rsvg-convert)
- NO text overflow (estimate: text.length × 7px ≤ shape_width - 16px)
- NO arrows passing through component interiors (route around)
- NO missing arrow labels background rects
- NO jump-over arcs missing for crossing arrows

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO (this is SVG generation, not code)
- **Automated tests**: YES (SVG validation + PNG export)
- **Framework**: N/A — use `rsvg-convert` for validation

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **SVG Validation**: Use `rsvg-convert file.svg -o /dev/null 2>&1` to check syntax
- **PNG Export**: Use `rsvg-convert -w 1920 file.svg -o file.png`
- **Visual Check**: Not required (agent cannot "see" but file existence + size can be verified)

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - single task):
├── Task 1: Generate SVG Architecture Diagram [deep]
└── Task 1 continues: Validate SVG → Export PNG

Critical Path: Task 1 (SVG gen) → Validate → PNG Export
Parallel Speedup: N/A (single diagram generation task)
Max Concurrent: 1 (sequential steps within task)
```

### Dependency Matrix
- **1**: - - SVG generation, validation, PNG export

### Agent Dispatch Summary
- **1**: **1** - Task 1 → `deep` (complex SVG generation with 7 layers, multiple node types, arrow routing)

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.
> **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**

- [ ] 1. Generate Model Inference Platform Architecture Diagram (SVG + PNG)

  **What to do**:
  - Read Style 1 reference: `references/style-1-flat-icon.md` from fireworks-tech-graph skill
  - Read product spec Section 5.1 for 7-layer architecture definition
  - Generate SVG with Python list method (prevents truncation):
    - SVG header with `viewBox="0 0 960 1400"`
    - Style block with font-family, layer-title, node-label, node-sub, arrow-label classes
    - Defs block with 5 arrow markers (blue/green/purple/orange/gray) + drop shadow filter
    - White background rect
    - Title: "模型推理云平台 — 系统架构图" at y=32
    - 7 Layer containers (dashed borders, light backgrounds):
      - L1 Access Layer (y=65, h=85): Web Console, Responses+Chat API, SDK, CLI, 3rd Party
      - L2 Gateway & Routing (y=165, h=90): API Gateway, Responses Adapter, Auth, Rate Limit, Smart Router
      - L3 Control Plane (y=270, h=180, 2 rows): User/Team, Billing, Endpoints, Model Registry, Load Balancer, A/B, Eval, Templates, Alerts, Quality, Cache, Scheduler, RBAC
      - L4 Data Plane (y=465, h=155): MultiEngineRouter (vLLM/SGLang/Custom/Mock), Dedicated Endpoints, Batch/Fine-tune, Cross-domain, Agentic RL
      - L5 Orchestration (y=635, h=140): Volcano Scheduler, HAMi Plugin, K8s Native
      - L6 Storage & Data (y=790, h=100): PostgreSQL, Redis, S3, ClickHouse, Kafka, LanceDB
      - L7 Observability (y=905, h=90): Prometheus, Grafana, OpenTelemetry, Loki
    - Inter-layer arrows (L1→L2→L3→L4→L5→L6→L7) with proper colors
    - Internal L4 arrows: MultiEngineRouter → Dedicated, → Batch/Fine-tune, Dedicated → Cross-domain
    - Side arrows: Data Plane → Storage (semantic cache, async log, model weights)
    - Legend box (y=1015, h=50) with 5 arrow color explanations
    - Key features callout (y=1085, h=120) with 6 architecture highlights
    - Footer text (y=1230)
  - Write SVG to: `/Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg`
  - Validate: `rsvg-convert file.svg -o /dev/null`
  - Export PNG: `rsvg-convert -w 1920 file.svg -o file.png`
  - Both files in same directory as product spec

  **Must NOT do**:
  - Do NOT use `@import` in SVG (breaks rsvg-convert)
  - Do NOT let arrows pass through component interiors
  - Do NOT forget arrow label background rects (opacity=0.95)
  - Do NOT skip jump-over arcs for crossing arrows in L4

  **Recommended Agent Profile**:
  > Select category + skills based on task domain. Justify each choice.
  - **Category**: `deep` - Complex SVG generation requiring 7-layer architecture, multiple node types, arrow routing, legend, and visual formatting. This is NOT a simple single-file task.
    - Reason: The task requires thorough research (read spec + style guide), structural planning (layer dimensions, node positions), and careful implementation (7 layers × ~10 nodes each + arrows + legend + key features).
  - **Skills**: `fireworks-tech-graph` (loaded), `read` (to read style reference and product spec)
    - `fireworks-tech-graph`: Required to follow the skill's workflow (Python list method, validation, export)
    - `read`: Required to load `references/style-1-flat-icon.md` and verify product spec section 5.1
  - **Skills Evaluated but Omitted**:
    - `websearch`: Not needed — all info from local documents
    - `drawio-skill`: Not needed — generating SVG directly per fireworks-tech-graph skill

  **Parallelization**:
  - **Can Run In Parallel**: NO — single diagram generation task
  - **Parallel Group**: Wave 1 (only task)
  - **Blocks**: None (first task)
  - **Blocked By**: None (start immediately)

  **References** (CRITICAL - Be Exhaustive):

  > The executor has NO context from your interview. References are their ONLY guide.
  > Each reference must answer: "What should I look at and WHY?"

  **Pattern References** (existing code to follow):
  - `/Users/apple/.agents/skills/fireworks-tech-graph/references/style-1-flat-icon.md` — Style 1 color tokens, arrow markers, box shapes, legend format, SVG template. WHY: Need exact color codes (#2563eb, #16a34a, etc.), arrow marker definitions, and font settings.
  - `/Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/模型推理云平台-产品方案.md:Section 5.1` — 7-layer architecture definition with component lists. WHY: Source of truth for which components go in each layer, layer names (Chinese + English), and arrow flow descriptions.

  **API/Type References** (contracts to implement against):
  - N/A (SVG generation, not API)

  **Test References** (testing patterns to follow):
  - `/Users/apple/.agents/skills/fireworks-tech-graph/references/style-1-flat-icon.md:Legend section` — Legend format with arrow markers + text. WHY: Need to create matching legend at bottom of diagram.

  **External References** (libraries and frameworks):
  - N/A (pure SVG, no external libraries)

  **WHY Each Reference Matters** (explain the relevance):
  - Style file: Without the exact color tokens and arrow marker definitions, the SVG won't match Style 1 "Flat Icon" and may fail validation.
  - Product spec Section 5.1: Without the layer definitions, components may be placed in wrong layers or omitted entirely.

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** - No human action permitted.
  > Every criterion MUST be verifiable by running a command or using a tool.

  **If TDD (tests enabled):**
  - [ ] SVG file created: `model-inference-platform-architecture.svg` exists in correct directory
  - [ ] `rsvg-convert model-inference-platform-architecture.svg -o /dev/null 2>&1` → "Valid" output (no XML errors)
  - [ ] PNG exported: `model-inference-platform-architecture.png` exists, file size > 0
  - [ ] `identify -format '%w %h' model-inference-platform-architecture.png` → "1920 [some height]" (width verified)

  **QA Scenarios (MANDATORY - task is INCOMPLETE without these):**

  > This is NOT optional. A task without QA scenarios WILL BE REJECTED.
  >
  > Write scenario tests that verify the ACTUAL BEHAVIOR of what you built.
  > Minimum: 1 happy path + 1 failure/edge case per task.
  > Each scenario = exact tool + exact steps + exact assertions + evidence path.
  >
  > **The executing agent MUST run these scenarios after implementation.**
  > **The orchestrator WILL verify evidence files exist before marking task complete.**

  ```
  Scenario: Happy path - SVG validates and PNG exports correctly
    Tool: Bash (rsvg-convert)
    Preconditions: SVG file written to output directory
    Steps:
      1. Run `rsvg-convert /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg -o /dev/null 2>&1`
      2. Run `rsvg-convert -w 1920 /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg -o /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.png`
      3. Run `ls -la /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.png`
    Expected Result: 
      - Step 1: Output contains "Valid" or no error messages
      - Step 2: PNG file created successfully
      - Step 3: PNG file exists, size > 0 bytes
    Failure Indicators: 
      - Step 1: "XML parsing error" or "syntax error"
      - Step 2: File not created
      - Step 3: File size = 0 or file not found
    Evidence: `.sisyphus/evidence/task-1-svg-validation.txt`

  Scenario: Happy path - SVG contains all 7 layers
    Tool: Bash (grep)
    Preconditions: SVG file exists
    Steps:
      1. Run `grep -c "layer-title" /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg`
    Expected Result: 
      - Output ≥ 7 (7 layer-title elements for each layer)
    Failure Indicators: 
      - Output < 7 (missing layers)
    Evidence: `.sisyphus/evidence/task-1-layer-count.txt`

  Scenario: Failure case - Arrow labels have background rects
    Tool: Bash (grep)
    Preconditions: SVG file exists
    Steps:
      1. Run `grep -c "arrow-label" /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg`
      2. Run `grep -A2 "arrow-label" /Users/apple/Documents/Obsidian Vault/X-Product-Design/Model-Inference-Platform-Design/model-inference-platform-architecture.svg | grep "rect.*opacity=\"0.95\""`
    Expected Result: 
      - Step 1: Output ≥ 10 (multiple arrow labels)
      - Step 2: At least one background rect found for arrow labels
    Failure Indicators:
      - Step 1: 0 arrow labels (missing all arrow labels)
      - Step 2: No background rects found (violates critical rule)
    Evidence: `.sisyphus/evidence/task-1-arrow-labels.txt`
  ```

  **Evidence to Capture:**
  - [ ] `task-1-svg-validation.txt`: rsvg-convert output
  - [ ] `task-1-layer-count.txt`: grep -c output
  - [ ] `task-1-arrow-labels.txt`: grep output for arrow labels

  **Commit**: NO (generation task, not code repo)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.

- [ ] F1. Plan Compliance Audit — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, check content). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. Code Quality Review — `unspecified-high`
  Run `rsvg-convert file.svg -o /tmp/test.png 2>&1 && echo "Valid"`. Review SVG for: missing arrow markers, text overflow, arrow-component collisions, missing label background rects, jump-over arcs for crossings. Check all 7 layers present with correct components per Section 5.1.
  Output: `SVG [Valid/Invalid] | Layers [N/7] | Arrows [N issues] | Text [N overflow] | VERDICT`

- [ ] F3. Real Manual QA — `unspecified-high` (+ `fireworks-tech-graph` skill if needed)
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (layers connect correctly, arrows flow proper directions). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. Scope Fidelity Check — `deep`
  For each task: read "What to do", read actual diff (SVG content). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's content. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `type(scope): desc` - file.svg, npm test

---

## Success Criteria

### Verification Commands
```bash
rsvg-convert model-inference-platform-architecture.svg -o /dev/null 2>&1 | grep -i "valid\|error"
identify -format '%w %h' model-inference-platform-architecture.png
grep -c "layer-title" model-inference-platform-architecture.svg
```

### Final Checklist
- [ ] All "Must Have" present (7 layers, all components per Section 5.1)
- [ ] All "Must NOT Have" absent (no @import, no text overflow, no arrow collisions)
- [ ] All tests pass (SVG valid, PNG exports, layer count ≥ 7)
