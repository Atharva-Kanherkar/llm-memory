# Architecture Decision: Algorithms vs LLM Responsibilities

**Decision Date:** February 2025
**Status:** Accepted

---

## Context

The Mnemosyne system involves mathematical formulas for memory decay, retrieval, and consolidation. A key architectural question: should these computations be handled by algorithms or delegated to LLMs?

## Decision

**Algorithms do the math. LLMs do the understanding.**

---

## What Algorithms Handle (Fast, Deterministic, Cheap)

All core mathematical operations:

| Formula | Purpose | Why Algorithmic |
|---------|---------|-----------------|
| `M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))` | Memory strength decay | Pure exponential decay, microseconds to compute |
| `P(recall mᵢ) = [Mᵢ(t) · Cᵢ(t)] / [Σⱼ Mⱼ(t) · Cⱼ(t) + β]` | Retrieval with competition | Vector similarity + weighted multiplication |
| `M₀ = α₁·N + α₂·E + α₃·A + α₄·ΔP` | Initial encoding strength | Weighted sum of signals |
| Forgetting risk | Proactive surfacing | Arithmetic on existing values |
| Context similarity | Contextual recall | Cosine similarity between embeddings |
| Temporal linking | Memory chains | Timestamp comparisons |

### Requirements for algorithmic components:
- **Fast** — microseconds, not seconds
- **Deterministic** — same input → same output
- **Free** — no API calls, no cost per operation
- **Offline-capable** — works without network

These run constantly: on every capture, on every retrieval, on decay updates.

---

## What LLMs Handle (Slow, Expensive, Nuanced)

Operations requiring semantic understanding:

| Task | Why LLM |
|------|---------|
| **Summarization** | Turning raw captures into coherent memory text |
| **Consolidation merging** | Combining multiple similar memories into one meaningful summary |
| **Reconstruction** | Rebuilding rich context from sparse fragments |
| **Entity extraction** | Identifying people, projects, concepts from content |
| **Sentiment detection** | Determining emotional valence of captured content |
| **Query understanding** | Parsing natural language queries into retrieval intent |
| **Response generation** | Natural conversation in TUI/voice interfaces |
| **Mirror/Advisor modes** | Cognitive twin personality responses |

### Characteristics of LLM tasks:
- Require nuance and context
- Benefit from world knowledge
- Output is text, not numbers
- Called sparingly, not on every operation

---

## Hybrid Flow

```
CAPTURE FLOW
────────────────────────────────────────────────────────────
Event → [Algorithm: compute M₀, salience, embedding] → Store
        └─ fast, every capture

RETRIEVAL FLOW
────────────────────────────────────────────────────────────
Query → [Algorithm: vector search, competition scoring, ranking]
      → [LLM: reconstruct/summarize if sparse] → Display
        └─ algorithm first, LLM only if needed

CONSOLIDATION FLOW (nightly)
────────────────────────────────────────────────────────────
[Algorithm: cluster similar weak memories by embedding distance]
      → [LLM: generate merged summary text] → Store
        └─ algorithm finds clusters, LLM writes summary
```

---

## Consequences

### Positive
- Daemon runs efficiently with minimal resource usage
- System works offline for core functionality
- Predictable, testable memory behavior
- LLM costs are bounded (only on explicit interactions + nightly consolidation)

### Negative
- Two systems to maintain (algorithmic core + LLM integration)
- Need to carefully define the handoff points
- Some edge cases may need LLM but won't get it (offline mode)

---

## Implementation Notes

The daemon is **mostly algorithmic**. LLM calls happen:
1. On explicit user queries (TUI/voice)
2. During nightly consolidation
3. When reconstruction is needed for sparse memories
4. For entity extraction on high-salience captures

This keeps the system responsive and cost-effective while preserving the benefits of LLM understanding where it matters most.
