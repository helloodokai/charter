Generate a complete, unambiguous software specification document (SPEC.md) that
an autonomous coding agent can execute without any additional context or
clarification from a human.

You are given a charter (structured intent) and a transcript (the Socratic
dialogue that produced it). Use BOTH to write the spec. The transcript
contains nuances, clarifications, and constraints the human stated verbally
that may not appear in the charter's structured fields.

## Absolute Rules

1. DO NOT suggest implementation approaches, architectures, libraries, or
   code. The spec describes WHAT, never HOW.
2. DO NOT invent requirements not stated in the charter or transcript.
3. Preserve the user's exact terminology and phrasing where possible.
4. Every requirement must be testable — if you cannot write a test for it,
   rephrase until you can.
5. If the charter or transcript contradicts itself, flag the contradiction
   explicitly rather than silently choosing one side.

## Success Criteria

1. An autonomous coding agent given ONLY this document could build the
   correct system without asking a single clarifying question.
2. Every section is precise enough to have a clear pass/fail test.
3. No section contains implementation details, technology choices, or
   architectural decisions.
4. All ambiguities from the counter-spec are explicitly resolved or
   documented as open decisions.
5. The document is self-contained — no external references, no "see issue
   #42", no "ask the PM".

## Required Output Structure

Produce a SPEC.md with EXACTLY these sections, in this order. If a section
has no content (e.g., no edge cases were identified), write "None
identified." under that heading — never omit the section.

# SPEC: <goal — one sentence>

## Overview

<1–2 paragraph explanation of what this system does and why, based on the
charter's goal and context. Use the human's own words from the transcript
where possible.>

## Requirements

### Functional Requirements

<numbered list of FRs, each with: FR-N: <statement>
- Each must be independently testable
- Use RFC 2119 language (MUST, SHOULD, MAY) precisely>

### Non-Functional Requirements

<numbered list of NFRs with the same format, covering performance,
security, compatibility, and style constraints from the charter>

## Acceptance Criteria

<list each criterion from the charter with its verification method.
Rewrite for agent clarity: state exactly what must be true for the
criterion to pass. Format: AC-N: <criterion> [verification: test|manual|metric]>

## Scope

### In Scope

<what this system MUST do, derived from goal and acceptance criteria>

### Out of Scope

<what this system MUST NOT do, from non-goals and transcript clarifications>

## Boundaries

### Affected Files / Modules

<from blast_radius.files — list patterns or descriptions>

### Affected Services

<from blast_radius.services>

### Affected Data

<from blast_radius.data>

## Constraints

<list all constraints from the charter. Performance, security,
compatibility, style, dependencies — each as a precise, testable
statement.>

## Edge Cases

<list each edge case. For each: EC-N: <description>
- State expected behavior precisely>

## Open Questions

<list each unknown from the charter. For each: OQ-N: <question>
  - Blocking: yes|no
  - Resolution needed before: <phase or "any">>

## Risk Assessment

- Risk Level: <low|medium|high|critical>
- Rationale: <one sentence>
- Rollback Plan: <from charter, or "None specified">

## Counter-Spec Resolution

<for each misinterpretation or ambiguity from the counter-spec:
- Ambiguity: <what was ambiguous>
- Resolution: <how this spec resolves it, or "Unresolved — requires
  human decision">>

CRITICAL: Output ONLY the SPEC.md content. No preamble, no commentary,
no explanations outside the document structure.

BEGIN