# CLIPS Pipeline Convention

This document describes a convention for creating composable CLIPS rulebases that can be chained together in pipelines. Following this convention makes it easier to connect rulebases where one's output becomes another's input.

## Overview

A **pipeline** is a sequence of CLIPS rulebases where:
1. Each stage processes input facts and produces output facts
2. Output facts from stage N become input facts for stage N+1
3. Stages can be composed dynamically at runtime

## Core Concepts

### 1. Pipeline Envelope

Every pipeline-compatible rulebase should use a standard **envelope** template that wraps the domain-specific payload:

```clips
(deftemplate pipeline-item
    "Standard envelope for pipeline-compatible facts"
    (slot item-id (type STRING))              ; Unique identifier for tracking
    (slot item-type (type SYMBOL))            ; Domain-specific type (e.g., order, incident)
    (slot stage (type SYMBOL))                ; Current pipeline stage
    (slot status (type SYMBOL)                ; Processing status
        (allowed-symbols pending processing completed failed routed))
    (slot route-to (type SYMBOL) (default nil))  ; Next stage hint (for branching)
    (slot created-at (type STRING))           ; ISO 8601 timestamp
    (slot source-stage (type SYMBOL) (default nil)))  ; Which stage produced this
```

### 2. Stage Markers

Each rulebase should define its stage identity:

```clips
(deftemplate stage-info
    "Identifies this rulebase's role in a pipeline"
    (slot stage-name (type SYMBOL))           ; e.g., validation, pricing, fulfillment
    (slot stage-order (type INTEGER))         ; Position in pipeline (1, 2, 3...)
    (slot accepts-types (type STRING))        ; Comma-separated item-types accepted
    (slot produces-types (type STRING))       ; Comma-separated item-types produced
    (slot can-route-to (type STRING)))        ; Comma-separated next stages (for branching)
```

### 3. Naming Conventions

#### Templates
- **Input templates**: `<domain>-input` or domain-specific names
- **Output templates**: `<domain>-result` or `<domain>-<action>`
- **Envelope**: Always include `pipeline-item` for tracking

#### Stage Names
Use verb-noun or descriptive names:
- `validation`, `pricing`, `fulfillment`
- `detection`, `classification`, `response`
- `inspection`, `grading`, `routing`

#### Status Values
Standard status progression:
- `pending` → Item received, not yet processed
- `processing` → Currently being evaluated by rules
- `completed` → Successfully processed, ready for next stage
- `failed` → Processing failed, includes error info
- `routed` → Branching decision made, check `route-to`

### 4. Routing Patterns

#### Linear Pipeline (A → B → C)
```
stage-order: 1 → 2 → 3
route-to: nil (implicit next stage)
```

#### Branching Pipeline
```
Classification stage sets route-to:
  - route-to: security-response
  - route-to: operations-response
  - route-to: escalation
```

#### Conditional Skip
```
If validation passes with no issues:
  - route-to: fulfillment (skip pricing)
Otherwise:
  - route-to: pricing (normal flow)
```

## Template Compatibility

### Rule: Output Must Match Input

For stage B to accept output from stage A:
1. A's output templates must match B's expected input templates
2. Slot names and types must be compatible
3. The `pipeline-item` envelope must be present

### Compatibility Matrix Example

| Stage | Accepts | Produces |
|-------|---------|----------|
| order-validation | order-request | validated-order |
| order-pricing | validated-order | priced-order |
| order-fulfillment | priced-order | fulfillment-decision |

### Adapter Pattern

When rulebases aren't directly compatible, create an adapter rulebase:

```clips
;;; adapter-pricing-to-legacy.clp
(defrule adapt-priced-order
    (priced-order (order-id ?id) (total ?t))
    =>
    (assert (legacy-order (id ?id) (amount ?t))))
```

## Implementation Guidelines

### 0. Self-Contained Rule Files

Each pipeline stage rule file must be **self-contained** with all necessary template definitions. The CLIPS provider loads each rule file independently, so templates referenced in rules must be defined in that file.

**Include in every pipeline stage file:**
```clips
;;; ==========================================================================
;;; Pipeline Templates (required for pipeline compatibility)
;;; ==========================================================================

(deftemplate pipeline-item
    "Standard envelope for pipeline-compatible facts"
    (slot item-id (type STRING))
    (slot item-type (type SYMBOL))
    (slot stage (type SYMBOL))
    (slot status (type SYMBOL)
        (allowed-symbols pending processing completed failed routed skipped)
        (default pending))
    (slot route-to (type SYMBOL) (default nil))
    (slot created-at (type STRING) (default ""))
    (slot source-stage (type SYMBOL) (default nil)))

(deftemplate stage-info
    "Identifies this rulebase's role in a pipeline"
    (slot stage-name (type SYMBOL))
    (slot stage-order (type INTEGER) (default 0))
    (slot accepts-types (type STRING) (default ""))
    (slot produces-types (type STRING) (default ""))
    (slot can-route-to (type STRING) (default "")))
```

Also include any templates from previous stages that this stage needs to read (e.g., if pricing reads `validated-order`, define it in `order-pricing.clp`).

### 1. Always Include Envelope

```clips
(defrule process-order
    ?item <- (pipeline-item (item-id ?id) (status pending))
    (order-request (order-id ?id) ...)
    =>
    (modify ?item (status processing))
    ;; ... processing logic ...
    (modify ?item (status completed) (source-stage validation)))
```

### 2. Preserve Item ID

The `item-id` should flow through all stages for traceability:

```clips
(defrule create-output
    (pipeline-item (item-id ?id) (status completed))
    (validated-order (order-id ?id) ...)
    =>
    (assert (pipeline-item
        (item-id ?id)  ; Same ID preserved
        (item-type order)
        (stage pricing)
        (status pending))))
```

### 3. Include Provenance

Output facts should reference their source:

```clips
(assert (priced-order
    (order-id ?id)
    (source-stage validation)  ; Where this came from
    ...))
```

### 4. Handle Errors Gracefully

```clips
(defrule handle-validation-error
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (validation-error (order-id ?id) (reason ?r))
    =>
    (modify ?item (status failed))
    (assert (pipeline-error
        (item-id ?id)
        (stage validation)
        (error-type validation-failed)
        (message ?r))))
```

## JSON Pipeline Execution

### Running a Pipeline

```rust
// Stage 1: Validation
let stage1_output = provider.chat(&ChatRequest::new("order-validation.clp")
    .with_message(Message::user(input_json))).await?;

// Extract conclusions for next stage
let stage2_input = extract_for_next_stage(&stage1_output, "pricing")?;

// Stage 2: Pricing
let stage2_output = provider.chat(&ChatRequest::new("order-pricing.clp")
    .with_message(Message::user(stage2_input))).await?;

// Stage 3: Fulfillment
let stage3_input = extract_for_next_stage(&stage2_output, "fulfillment")?;
let final_output = provider.chat(&ChatRequest::new("order-fulfillment.clp")
    .with_message(Message::user(stage3_input))).await?;
```

### Branching Execution

```rust
// Check route-to for branching
let route = get_route_decision(&classification_output)?;

let response_rulebase = match route.as_str() {
    "security" => "incident-response-security.clp",
    "operations" => "incident-response-ops.clp",
    _ => "incident-response-general.clp",
};

let response = provider.chat(&ChatRequest::new(response_rulebase)
    .with_message(Message::user(stage_input))).await?;
```

## Example Pipeline Structure

```
examples/
├── rules/
│   └── pipeline/
│       ├── common.clp                    # Shared templates reference
│       ├── order-validation.clp          # Stage 1: Validate order
│       ├── order-pricing.clp             # Stage 2: Apply pricing
│       ├── order-fulfillment.clp         # Stage 3: Route to fulfillment
│       ├── incident-detection.clp        # Stage 1: Detect incident
│       ├── incident-classification.clp   # Stage 2: Classify & route
│       ├── incident-response-security.clp    # Branch A: Security team
│       ├── incident-response-ops.clp         # Branch B: Operations team
│       └── incident-response-escalation.clp  # Branch C: Escalation path
├── data/
│   └── pipeline/
│       ├── orders.json               # Sample order inputs
│       └── incidents.json            # Sample incident inputs
└── clips_pipeline.rs                 # Pipeline runner example
```

## Running the Examples

```bash
# Run the pipeline example
cargo run --example clips_pipeline --features clips
```

This demonstrates:
- **Order Pipeline**: validation → pricing → fulfillment (linear)
- **Incident Pipeline**: detection → classification → security/ops/escalation (branching)

Sample output:
```
┌──────────────────────────────────────────────────────────────────┐
│ Scenario: standard-order                                         │
└──────────────────────────────────────────────────────────────────┘
  Description: Normal retail order - full 3-stage pipeline

  Stage: validation (order-validation.clp)
    Status: completed
    Route-to: pricing
    ✓ Order validated

  Stage: pricing (order-pricing.clp)
    Status: completed
    Route-to: fulfillment
    Pricing: $249.99 → $269.99 (discount: $0.00)

  Stage: fulfillment (order-fulfillment.clp)
    Status: completed
  ✓ Pipeline complete
```

## Benefits of This Convention

1. **Composability**: Mix and match stages from different domains
2. **Traceability**: Item IDs flow through entire pipeline
3. **Flexibility**: Support linear, branching, and conditional flows
4. **Debuggability**: Each stage's output can be inspected independently
5. **Reusability**: Stages can be used in multiple pipelines
6. **Testability**: Each stage can be tested in isolation

## Versioning

When evolving pipeline stages:
- Add new optional slots (backward compatible)
- Create new templates for breaking changes (e.g., `order-v2`)
- Use adapters to bridge incompatible versions
