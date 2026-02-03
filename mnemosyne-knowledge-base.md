# MNEMOSYNE: A Cognitive Prosthesis System

## Complete Knowledge Base & Architecture Document

**Project Codename:** Mnemosyne (Greek goddess of memory, mother of the Muses)

**Purpose:** A personal memory prosthesis system for someone experiencing medication-induced memory difficulties. Not a note-taking app—a genuine external hippocampus that captures, encodes, decays, consolidates, and retrieves memories with human-like characteristics.

**Author's Context:** Built for a programmer on SSRIs and sedatives who experiences forgetting of small but important things—decisions made, context before meetings, conversations had. The system must work *despite* impaired memory, not demand more from it.

---

# Part 1: Conceptual Foundation

## 1.1 The Core Problem with Existing Tools

Everything that exists (Notion, Obsidian, voice notes, etc.) requires you to **remember to capture**. That's the cruel irony—they assume the thing you're lacking.

What's needed is something that works *despite* impaired memory:
- Zero-capture friction (passive, ambient)
- Anticipatory, not reactive (surfaces before you realize you've forgotten)
- Sparse capture, rich reconstruction (stores seeds, not forests)
- Human-like forgetting (not everything deserves permanence)

## 1.2 Design Principles

| Principle | Rationale |
|-----------|-----------|
| **Zero-capture friction** | You can't rely on remembering to remember. Everything passive. |
| **Anticipatory, not reactive** | Don't wait for queries. Surface before the need is conscious. |
| **Sparse capture, rich reconstruction** | Store seeds, not forests. LLM regrows context from fragments. |
| **Graceful degradation** | Works offline. Works without LLM. Layers add intelligence. |
| **Privacy by design** | All raw data stays local. Only embeddings touch the cloud (optional). |
| **Beautiful objects** | If hardware is added later, it should look like a sci-fi artifact. |

## 1.3 The Vision: Cognitive Twin

Not just memory storage—a **model of yourself** that the system builds over time:

- How do you write? (code style, comment style, commit messages)
- How do you decide? (what factors do you weigh?)
- How do you speak? (verbal tics, favorite phrases)
- What do you value? (what do you prioritize in tradeoffs?)
- When do you struggle? (what patterns precede frustration?)

**Twin Modes:**

1. **Memory Mode** (default): "What was I doing yesterday?" → Retrieves and reconstructs YOUR memories
2. **Mirror Mode**: "What would I think about this?" → Responds AS you would, based on learned patterns
3. **Advisor Mode**: "What should I do?" → Your patterns + external knowledge, can disagree with "you"
4. **Dialogue Mode**: Talk to "yourself" → Rubber duck debugging, but the duck knows you

---

# Part 2: Mathematical Foundations of Memory

## 2.1 Memory Strength as a Function of Time

The classic Ebbinghaus forgetting curve:

```
M(t) = M₀ · e^(-λt)
```

Where:
- `M₀` = initial intensity (emotion, novelty, shock)
- `λ` = decay rate (higher = forget faster)
- `t` = time

This explains why:
- Random lectures vanish fast
- Emotionally loaded moments stick longer

But this alone is too clean. Real brains are messier.

## 2.2 Reactivation Model (Why Some Memories Don't Decay Smoothly)

Reality looks more like:

```
M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))
```

Where:
- Each `Rᵢ` = a reactivation (thinking, recalling, dreaming, trauma trigger)
- `tᵢ` = time of that reactivation

Every recall re-writes the memory.

**Key insight:** Memory is not read-only storage. It's more like a Git repo with force-push enabled.

## 2.3 Memory Merging (Why Details Blur but Vibes Stay)

Memories live as vectors in feature space:

```
m⃗ = [emotion, place, people, body-state, meaning]
```

Over time, similar memories undergo attractor dynamics:

```
m⃗_new = α·m⃗₁ + (1-α)·m⃗₂ + ε
```

Where:
- Similar events collapse into a prototype
- `ε` = noise (why details get weird or wrong)

That's why:
- You forget what exactly was said
- But remember how it felt

**Emotion has the highest weight in the vector.**

## 2.4 What Memories Survive Long-Term?

Mathematically, memories persist if:

```
dM/dt ≈ 0
```

That happens when at least one is true:

1. **High emotional gradient**: Strong amygdala activation → low decay constant (grief, fear, love, shame)

2. **Identity relevance**: If a memory updates your internal model of who you are:
   ```
   M → self-schema
   ```
   These decay logarithmically, not exponentially.

3. **Repeated unpredictable recall**: Irregular reactivation > spaced repetition (dreams, smells, random cues)

## 2.5 Why "Forgotten" Memories Randomly Pop Up

Memory recall probability:

```
P(recall) ∝ M(t) · C(t)
```

Where `C(t)` = contextual overlap.

A smell, lighting, posture, song, or emotional state can spike `C(t)` even if `M(t)` is low.

That's why:
- A random auto ride
- A hospital smell
- 3am silence

...can unlock stuff you swore was gone. They weren't erased—just below threshold.

## 2.6 The Uncomfortable Truth

Memories don't fade uniformly. They:
- Compress
- Merge
- Mutate
- Reappear out of order

Mathematically, your brain optimizes for **meaning over accuracy**.

**One-line summary:** Forgetting isn't deletion—it's loss of access. And remembering is rewriting.

---

# Part 3: Extended Mathematical Model for Implementation

## 3.1 Interference — Memories Compete

Similar memories don't just merge—they compete for retrieval.

```
P(recall mᵢ) = [Mᵢ(t) · Cᵢ(t)] / [Σⱼ Mⱼ(t) · Cⱼ(t) + β]
```

Where:
- The denominator is ALL memories competing for activation
- `β` = baseline noise (prevents division by zero, models "tip of tongue" failures)

This explains:
- Learning new things can make old things harder to recall (retroactive interference)
- Similar memories block each other (you remember "a conversation about Redis" but not which one)

**Implementation:** When retrieving, we don't just find the best match. We model the competition. If many similar memories exist, we might surface the *merged prototype* instead of individuals.

## 3.2 Encoding Strength — Not All Moments Are Born Equal

What determines initial strength `M₀`?

```
M₀ = f(novelty, emotion, attention, prediction_error)
```

More precisely:

```
M₀ = α₁·N + α₂·E + α₃·A + α₄·ΔP
```

Where:
- `N` = novelty (how different from recent context)
- `E` = emotional arousal (detected from content, physiological signals if available)
- `A` = attention (was the user focused? long session? or distracted?)
- `ΔP` = prediction error (did something unexpected happen? an error? a surprise?)

**Computational approximations:**
- **Novelty:** embedding distance from recent memories
- **Emotion:** sentiment analysis + keywords (error, fuck, yes!, finally)
- **Attention:** session duration, typing speed, lack of context switches
- **Prediction error:** error messages, unexpected git conflicts, anomalies

## 3.3 Sleep Consolidation — Batch Processing

Real memory consolidation happens in **batches**—primarily during sleep.

The hippocampus replays memories during sleep, and only some make it to cortical long-term storage:
- Similar memories merge into schemas
- Weak memories get pruned
- Important memories get strengthened

**Implementation:** A "consolidation daemon" runs periodically (nightly, or when idle):
1. Finds clusters of similar weak memories
2. Generates merged summaries (the "schema")
3. Archives raw memories, keeps summaries active
4. Boosts high-salience memories that survived the day

## 3.4 Temporal Contiguity — Memories Form Chains

Memories that occur close in time get linked—recalling one activates neighbors.

```
P(recall mⱼ | recalled mᵢ) ∝ e^(-|tᵢ - tⱼ| / τ)
```

Where `τ` = temporal binding window (~minutes to hours).

This is why "what was I doing before the meeting?" is a valid query—temporal chains exist.

**Implementation:** Every memory stores:
- `previous_memory_id` — what came immediately before
- `next_memory_id` — what came after
- `session_id` — all memories from a continuous activity block

## 3.5 The Forgetting/Remembering Asymmetry

A memory can have:
- High `M(t)` (it exists, it's strong)
- Low `C(t)` (current context doesn't match)

Result: you can't access it NOW, but it's not forgotten. A different context unlocks it.

**Implementation:** TWO retrieval modes:
1. **Contextual retrieval** — "what's relevant to what I'm doing NOW?" (uses M × C)
2. **Semantic retrieval** — "what do I know about X?" (uses M alone, ignores current context)

---

# Part 4: System Architecture

## 4.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MNEMOSYNE SYSTEM                                  │
│                        "Your Cognitive Twin"                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         CAPTURE LAYER                                │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │   │
│  │  │  Screen  │ │   Git    │ │ Clipboard│ │  Audio   │ │  Files   │  │   │
│  │  │  Watcher │ │  Monitor │ │  History │ │ (opt-in) │ │ Activity │  │   │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │   │
│  │       └────────────┴────────────┴────────────┴────────────┘        │   │
│  │                                  │                                  │   │
│  │                                  ▼                                  │   │
│  │                        ┌─────────────────┐                          │   │
│  │                        │  PRIVACY GATE   │                          │   │
│  │                        │  (redact/block) │                          │   │
│  │                        └────────┬────────┘                          │   │
│  └─────────────────────────────────┼───────────────────────────────────┘   │
│                                    │                                        │
│  ┌─────────────────────────────────┼───────────────────────────────────┐   │
│  │                         MEMORY LAYER                                 │   │
│  ├─────────────────────────────────┼───────────────────────────────────┤   │
│  │                                 ▼                                    │   │
│  │  ┌──────────────────────────────────────────────────────────────┐   │   │
│  │  │                      WORKING MEMORY                          │   │   │
│  │  │              (last ~10 minutes, full detail)                 │   │   │
│  │  └──────────────────────────┬───────────────────────────────────┘   │   │
│  │                             │ (salience filter)                     │   │
│  │                             ▼                                       │   │
│  │  ┌──────────────────────────────────────────────────────────────┐   │   │
│  │  │                     SHORT-TERM MEMORY                        │   │   │
│  │  │           (hours, embedded, strength decaying)               │   │   │
│  │  └──────────────────────────┬───────────────────────────────────┘   │   │
│  │                             │ (nightly consolidation)               │   │
│  │                             ▼                                       │   │
│  │  ┌──────────────────────────────────────────────────────────────┐   │   │
│  │  │                      LONG-TERM MEMORY                        │   │   │
│  │  │         (days+, consolidated, merged, abstracted)            │   │   │
│  │  └──────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  Storage: SQLite (metadata) + ChromaDB (vectors) + Files (raw)     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│  ┌─────────────────────────────────┼───────────────────────────────────┐   │
│  │                       COGNITION LAYER                                │   │
│  ├─────────────────────────────────┼───────────────────────────────────┤   │
│  │                                 ▼                                    │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │   │
│  │  │  RETRIEVER  │  │  PREDICTOR  │  │ PERSONALITY │  │    LLM     │  │   │
│  │  │             │  │             │  │   MODEL     │  │  PROVIDER  │  │   │
│  │  │ context →   │  │ "you'll    │  │             │  │            │  │   │
│  │  │ relevant    │  │  forget    │  │ "how would  │  │ • Ollama   │  │   │
│  │  │ memories    │  │  this"     │  │  you say    │  │ • OpenAI   │  │   │
│  │  │             │  │             │  │  this?"     │  │ • Claude   │  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │ • Groq     │  │   │
│  │                                                      │ • Local    │  │   │
│  └──────────────────────────────────────────────────────┴────────────┴──┘   │
│                                    │                                        │
│  ┌─────────────────────────────────┼───────────────────────────────────┐   │
│  │                       INTERFACE LAYER                                │   │
│  ├─────────────────────────────────┼───────────────────────────────────┤   │
│  │       ┌─────────────────────────┼─────────────────────────┐         │   │
│  │       │                         │                         │         │   │
│  │       ▼                         ▼                         ▼         │   │
│  │  ┌─────────┐             ┌─────────────┐            ┌─────────┐    │   │
│  │  │   TUI   │             │   WEB UI    │            │  VOICE  │    │   │
│  │  │         │             │             │            │         │    │   │
│  │  │bubbletea│             │ htmx + templ│            │ Piper + │    │   │
│  │  │         │             │ :3333       │            │ Whisper │    │   │
│  │  └─────────┘             └─────────────┘            └─────────┘    │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 4.2 Data Flow

### Capture Flow
```
Event → Capture → Buffer → Privacy Gate → Encode → Link → Store
```

### Retrieval Flow
```
Context → Embed → Search → Rank (with competition) → Expand? → Format → Display
                                                        ↓
                                                   (if sparse)
                                                   LLM Rebuild
```

## 4.3 Interface Architecture

**Recommended approach (not Electron):**

1. **TUI** (bubbletea) — Daily interaction, terminal-native
2. **Web UI** (htmx + templ, localhost:3333) — Configuration, memory browsing
3. **Voice** (Piper TTS + Whisper STT) — Optional overlay

This gives:
- Terminal-native daily use
- Proper visual interface for complex configuration
- Voice as a layer on top of both

---

# Part 5: Data Structures

## 5.1 Memory Representation

```
Memory {
    // Identity
    id: UUID
    created_at: Timestamp
    updated_at: Timestamp
    
    // Classification
    type: MemoryType          // screen, git, window, clipboard, audio, file, explicit, consolidated
    stage: MemoryStage        // working, short_term, long_term, archived
    
    // Content
    raw_data_path: String     // Path to raw file (screenshot, audio, etc.)
    content: String           // Extracted text content
    summary: String           // LLM-generated summary
    embedding: Vector[384]    // Semantic embedding
    
    // Context (for computing C(t))
    context: MemoryContext {
        active_app: String
        active_window: String
        active_file: String
        active_url: String
        git_repo: String
        git_branch: String
        git_commit: String
        time_of_day: String   // "morning", "afternoon", "evening", "night"
        day_of_week: String
        is_work_hours: Bool
        activity_type: String // "coding", "reading", "meeting", "browsing"
        cognitive_load: Float // Estimated mental load (0-1)
        session_duration: Duration
        location_hash: String // Hashed, not actual coords
    }
    
    // Encoding strength factors (determine M₀)
    novelty: Float            // Embedding distance from recent memories
    emotional_intensity: Float // From sentiment + keywords
    attention_level: Float    // From context signals
    prediction_error: Float   // Anomaly score
    
    // Human-like memory properties
    M_0: Float                // Initial strength = f(novelty, emotion, attention, prediction_error)
    lambda: Float             // Decay rate (can vary per memory type)
    strength: Float           // Current computed strength M(t) (cached)
    emotional_valence: Float  // -1 (negative) to 1 (positive)
    
    // Reactivation history
    reactivations: [(timestamp, strength_boost)]
    access_count: Int
    last_accessed: Timestamp
    
    // Temporal links
    previous_memory_id: UUID?
    next_memory_id: UUID?
    session_id: UUID
    
    // Entity links
    linked_memory_ids: [UUID]
    entities: [Entity]        // People, projects, concepts extracted
    tags: [String]
    
    // Consolidation state
    consolidation_level: Int  // 0=raw, 1=summarized, 2+=abstracted
    consolidated_from: [UUID] // Source memories if merged
    
    // Predictive
    forgetting_risk: Float    // Predicted likelihood of being forgotten
    identity_relevance: Float // How much this relates to self-schema
}
```

## 5.2 Entity Representation

```
Entity {
    type: String   // "person", "project", "file", "concept", "error"
    name: String
    count: Int     // How often mentioned across memories
    first_seen: Timestamp
    last_seen: Timestamp
    related_memories: [UUID]
}
```

## 5.3 Personality Model (Cognitive Twin)

```
PersonalityModel {
    // Learned from code
    coding_style: {
        verbosity: Float
        comment_frequency: Float
        naming_conventions: String
        preferred_patterns: [String]
        common_mistakes: [String]
    }
    
    // Learned from text
    writing_style: {
        avg_sentence_length: Float
        vocabulary_complexity: Float
        frequent_phrases: [String]
        filler_words: [String]
        punctuation_habits: Map
    }
    
    // Learned from decisions
    decision_patterns: {
        risk_tolerance: Float
        time_of_day_bias: Distribution
        cognitive_load_threshold: Float
        procrastination_triggers: [Pattern]
    }
    
    // Learned from memory access patterns
    memory_patterns: {
        typical_forgetting_times: Distribution
        recall_triggers: [ContextPattern]
        consolidation_preferences: [TopicWeight]
        what_you_forget_most: [Category]
    }
    
    // Emotional patterns
    emotional_patterns: {
        frustration_indicators: [Signal]
        focus_indicators: [Signal]
        fatigue_indicators: [Signal]
    }
}
```

---

# Part 6: Core Algorithms

## 6.1 Strength Computation

```python
def compute_strength(memory, current_time, params):
    """
    Compute current memory strength using Ebbinghaus decay with reactivation.
    
    M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))
    
    Modified by salience, emotional intensity, and identity relevance.
    """
    if memory.stage == WORKING:
        return 1.0  # Working memory always at full strength
    
    t = (current_time - memory.created_at).total_hours()
    
    # Salience and emotional valence slow decay
    salience_factor = 1.0 + (memory.salience * params.salience_weight)
    emotional_factor = 1.0 + (abs(memory.emotional_valence) * params.emotional_weight)
    effective_lambda = memory.lambda / (salience_factor * emotional_factor)
    
    # Base decay
    base_strength = memory.M_0 * exp(-effective_lambda * t)
    
    # Reactivation boosts
    boost = 0
    for (reactivation_time, boost_amount) in memory.reactivations:
        dt = (current_time - reactivation_time).total_hours()
        boost += boost_amount * exp(-effective_lambda * 0.5 * dt)  # Boosts decay slower
    
    # Identity relevance: logarithmic instead of exponential decay
    if memory.identity_relevance > IDENTITY_THRESHOLD:
        identity_factor = 1 + log(1 + memory.identity_relevance * t)
    else:
        identity_factor = 1
    
    strength = (base_strength + boost) * identity_factor
    
    return clamp(strength, 0.0, 1.0)
```

## 6.2 Contextual Similarity

```python
def compute_context_similarity(memory, current_context):
    """
    Compute how similar the memory's context is to current context.
    Used for contextual recall (C(t) in the model).
    """
    weights = {
        'app': 0.15,
        'file': 0.20,
        'git_branch': 0.15,
        'time_of_day': 0.05,
        'activity_type': 0.25,
        'emotional_state': 0.20
    }
    
    similarity = 0
    for feature, weight in weights.items():
        mem_val = getattr(memory.context, feature)
        cur_val = getattr(current_context, feature)
        
        if mem_val == cur_val:
            similarity += weight
        elif is_similar(mem_val, cur_val):  # Fuzzy match
            similarity += weight * 0.5
    
    return similarity
```

## 6.3 Retrieval with Competition

```python
def retrieve(query_embedding, current_context, memory_store, top_k=5, beta=0.1):
    """
    Retrieve memories using the competition model:
    P(recall mᵢ) = [Mᵢ(t) · Cᵢ(t)] / [Σⱼ Mⱼ(t) · Cⱼ(t) + β]
    """
    candidates = []
    
    for memory in memory_store:
        # Semantic similarity to query
        semantic_sim = cosine_similarity(query_embedding, memory.embedding)
        
        # Current strength
        M_t = compute_strength(memory, now())
        
        # Context overlap
        C_t = compute_context_similarity(memory, current_context)
        
        # Combined score (unnormalized retrieval probability)
        score = M_t * (0.6 * semantic_sim + 0.4 * C_t)
        
        candidates.append((memory, score))
    
    # Normalize (competition)
    total = sum(score for _, score in candidates) + beta
    candidates = [(m, s/total) for m, s in candidates]
    
    # Sort and take top-k
    top = sorted(candidates, key=lambda x: x[1], reverse=True)[:top_k]
    
    # Check for high competition (many similar scores)
    # If so, might want to surface merged prototype instead
    scores = [s for _, s in top]
    if len(scores) > 1 and (max(scores) - min(scores)) < 0.1:
        # High competition - consider surfacing merged version
        pass
    
    # Record reactivation for retrieved memories
    for memory, _ in top:
        record_reactivation(memory, boost=0.15)
    
    return top
```

## 6.4 Salience Calculation

```python
def calculate_salience(context, content, memory_type):
    """
    Determine initial salience based on context and content.
    High salience = slower decay, higher priority.
    """
    salience = 0.3  # Base salience
    
    # Explicit captures are always high salience
    if memory_type == EXPLICIT:
        return 0.9
    
    # Activity-based boosts
    if context.activity_type == "coding":
        salience += 0.1
    
    # Cognitive load boost
    salience += context.cognitive_load * 0.2
    
    # Long session = engagement
    if context.session_duration > timedelta(minutes=30):
        salience += 0.1
    
    # Git commits are important
    if memory_type == GIT and context.git_commit:
        salience += 0.2
    
    # Content-based boosters
    indicators = {
        "TODO": 0.15, "FIXME": 0.15, "BUG": 0.2,
        "DECISION": 0.25, "IMPORTANT": 0.2,
        "error": 0.15, "Error": 0.15, "failed": 0.1,
        "fuck": 0.2, "finally": 0.15, "yes!": 0.15  # Emotional markers
    }
    
    for indicator, boost in indicators.items():
        if indicator.lower() in content.lower():
            salience += boost
    
    return clamp(salience, 0.0, 1.0)
```

## 6.5 Consolidation Algorithm

```python
def consolidate(memory_store, llm, params):
    """
    Nightly consolidation: merge weak similar memories into summaries.
    Like sleep consolidation in the brain.
    """
    # Find weak memories ready to merge
    weak_memories = [
        m for m in memory_store
        if compute_strength(m, now()) < params.consolidation_threshold
        and m.consolidation_level == 0
        and m.type != EXPLICIT  # Never auto-consolidate explicit captures
    ]
    
    # Cluster by semantic similarity
    clusters = cluster_embeddings(
        [m.embedding for m in weak_memories],
        min_cluster_size=3,
        threshold=0.7
    )
    
    for cluster_indices in clusters:
        cluster = [weak_memories[i] for i in cluster_indices]
        
        # Generate merged summary
        contents = [m.content for m in cluster]
        time_range = f"{min(m.created_at for m in cluster)} to {max(m.created_at for m in cluster)}"
        
        summary = llm.generate(f"""
            Merge these {len(cluster)} related memories into a single coherent memory.
            Time range: {time_range}
            
            Original memories:
            {chr(10).join(contents)}
            
            Create a summary that preserves:
            - Key decisions made
            - Important context
            - Emotional tone
            - Any action items or todos
            
            Speak as if this is the person's own memory.
        """)
        
        # Create consolidated memory
        merged = Memory(
            content=summary,
            embedding=mean([m.embedding for m in cluster]),
            M_0=max(m.M_0 for m in cluster),  # Inherit strongest encoding
            consolidation_level=1,
            consolidated_from=[m.id for m in cluster],
            context=merge_contexts([m.context for m in cluster]),
            emotional_valence=mean([m.emotional_valence for m in cluster])
        )
        
        memory_store.add(merged)
        
        # Archive (don't delete) originals
        for m in cluster:
            m.stage = ARCHIVED
    
    # Boost survivors (memories that stayed strong all day)
    for memory in memory_store:
        if memory.stage != ARCHIVED:
            strength = compute_strength(memory, now())
            if strength > params.survival_threshold:
                record_reactivation(memory, boost=params.survival_boost)
```

## 6.6 Forgetting Risk Prediction

```python
def compute_forgetting_risk(memory, params):
    """
    Predict how likely this memory is to be forgotten.
    Used for proactive surfacing.
    """
    strength = compute_strength(memory, now(), params)
    
    # Base risk from current strength
    risk = 1.0 - strength
    
    # Never accessed = higher risk
    if memory.access_count == 0:
        risk *= 1.3
    
    # High initial salience but decaying = important to preserve
    if memory.M_0 > 0.7 and strength < 0.4:
        risk *= 1.2
    
    # Recent explicit capture with no follow-up = might be forgotten
    if memory.type == EXPLICIT:
        hours_since = (now() - memory.created_at).total_hours()
        if hours_since > 24 and memory.access_count == 0:
            risk *= 1.4
    
    # Old but stable = low risk
    days_since = (now() - memory.created_at).total_seconds() / 86400
    if days_since > 7 and memory.access_count > 0 and strength > 0.5:
        risk *= 0.6
    
    return clamp(risk, 0.0, 1.0)
```

## 6.7 Proactive Surfacing

```python
def should_surface(current_context, memory_store, params):
    """
    Determine if we should proactively surface a memory.
    Triggers on context switches (potential forgetting moments).
    """
    # Detect context switch
    if not context_just_changed(current_context):
        return None
    
    previous_context = get_previous_context()
    
    # Find memories relevant to PREVIOUS context (what you were doing)
    query_embedding = embed(previous_context.summary)
    relevant = retrieve(
        query_embedding,
        previous_context,
        memory_store,
        top_k=3
    )
    
    # Check forgetting risk
    for memory, relevance_score in relevant:
        risk = compute_forgetting_risk(memory, params)
        
        if risk > params.risk_threshold and relevance_score > params.relevance_threshold:
            # This memory is relevant and at risk of being forgotten
            return memory
    
    return None
```

---

# Part 7: Capture Specifications

## 7.1 Screen Capture

```
Trigger: Every 60 seconds (configurable)
Process:
  1. Capture screenshot
  2. Check privacy rules (blocked apps, URLs)
  3. If blocked → discard
  4. OCR extract text
  5. Identify activity type from content
  6. Generate embedding
  7. Calculate salience
  8. Store with context
  
Storage: ~200KB per capture (compressed JPEG + text)
Daily estimate: ~100MB (pruned after embedding for long-term)
```

## 7.2 Git State Monitor

```
Trigger: Every 5 minutes + on branch change + on commit
Capture:
  1. Current repository
  2. Current branch
  3. Recent commits (last 5)
  4. Staged/unstaged diff summary
  5. Current file in editor
  
Salience boosters:
  - Commits with "fix", "bug", "important" → +0.2
  - Large diffs → +0.1
  - New branch creation → +0.15
```

## 7.3 Window Tracking

```
Trigger: Every 10 seconds + on window change
Capture:
  1. Active application
  2. Window title
  3. URL (if browser)
  4. File path (if editor)
  
Purpose: Context for other memories, not stored standalone
```

## 7.4 Clipboard History

```
Trigger: On clipboard change
Capture:
  1. Clipboard content (text only, skip images for now)
  2. Source application
  3. Timestamp
  
Privacy: Redact passwords, API keys, credit cards
Salience: Generally low (0.2) unless contains code or URLs
```

## 7.5 Audio Capture (Opt-in)

```
Trigger: Every 5 minutes (configurable)
Duration: 10 seconds per snippet
Process:
  1. Capture audio
  2. Run voice activity detection
  3. If speech → transcribe with Whisper
  4. If silence/ambient → store as "environment anchor"
  5. Speaker diarization if multiple voices
  
Storage: ~50KB per anchor (compressed)
Salience: Conversation → 0.6, Environment → 0.2
```

## 7.6 Manual Anchor (Explicit Capture)

```
Trigger: Hotkey or voice command
Action:
  1. Capture screenshot
  2. Capture 30 seconds audio (15 before, 15 after if available)
  3. Record any spoken note
  4. Mark as EXPLICIT type
  5. Set salience = 0.9
  
These NEVER auto-consolidate or auto-archive.
```

---

# Part 8: Privacy & Security

## 8.1 Privacy Rules

```yaml
privacy:
  # Blocked applications (won't capture when active)
  blocked_apps:
    - 1password
    - keepassxc
    - bitwarden
    - gnome-keyring
    - seahorse
  
  # Blocked URLs (won't capture if visible)
  blocked_urls:
    - "*bank*"
    - "*banking*"
    - "*.gov"
    - "mail.google.com"
    - "outlook.live.com"
  
  # Blocked window title patterns
  blocked_title_patterns:
    - "(?i)password"
    - "(?i)private"
    - "(?i)secret"
    - "(?i)incognito"
  
  # Redaction patterns (regex)
  redact_patterns:
    - '\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b'  # Credit cards
    - '(?i)api[_-]?key['":\s]*[a-zA-Z0-9_-]{20,}'   # API keys
    - '(?i)password['":\s]*\S+'                      # Passwords in config
    - '(?i)secret['":\s]*\S+'                        # Secrets
  
  # Sensitive hours (optional, no capture)
  sensitive_hours:
    start: "23:00"
    end: "06:00"
  
  # Encryption
  encrypt_raw_data: true
```

## 8.2 Data Storage

- **Raw data:** Encrypted at rest (AES-256), stored locally only
- **Embeddings:** Can optionally sync to VPS (meaningless without raw data)
- **Auto-delete:** Raw data older than 30 days (configurable), summaries kept
- **Export:** Full data export in open formats on demand
- **Deletion:** "Forget this" command with cryptographic erasure

## 8.3 Access Control

- No cloud sync by default
- No accounts, no telemetry
- Local-only by default
- Optional VPS for embeddings + query processing only
- Guest mode: pause all capture with one command

---

# Part 9: LLM Provider Abstraction

## 9.1 Supported Providers

```yaml
llm:
  default_provider: "ollama"  # or "openai", "anthropic", "groq"
  
  ollama:
    base_url: "http://localhost:11434"
    model: "llama3.1:8b"
    embedding_model: "nomic-embed-text"
  
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o-mini"
    embedding_model: "text-embedding-3-small"
  
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"
  
  groq:
    api_key: "${GROQ_API_KEY}"
    model: "llama-3.1-70b-versatile"
  
  # Task-specific provider selection
  tasks:
    embedding: "ollama"        # Fast, local
    chat: "ollama"             # Default conversations
    summary: "ollama"          # Consolidation summaries
    reconstruction: "anthropic" # Complex memory reconstruction (optional upgrade)
```

## 9.2 Provider Interface

```go
type LLMProvider interface {
    // Text generation
    Generate(prompt string, opts GenerateOptions) (string, error)
    
    // Streaming generation
    GenerateStream(prompt string, opts GenerateOptions) (<-chan string, error)
    
    // Embeddings
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    
    // Model info
    ModelName() string
    EmbeddingDimension() int
}
```

---

# Part 10: Voice System

## 10.1 Speech-to-Text Options

| Engine | Latency | Accuracy | Privacy | Resource |
|--------|---------|----------|---------|----------|
| Whisper.cpp | ~1s | Excellent | Local | Medium |
| Vosk | ~200ms | Good | Local | Low |
| OpenAI Whisper API | ~500ms | Excellent | Cloud | None |

**Recommendation:** Whisper.cpp for quality, Vosk for speed.

## 10.2 Text-to-Speech Options

| Engine | Latency | Quality | Privacy | Cost |
|--------|---------|---------|---------|------|
| Piper | ~100ms | Good | Local | Free |
| Coqui TTS | ~200ms | Great | Local | Free |
| ElevenLabs | ~500ms | Excellent | Cloud | ~$5/mo |
| OpenAI TTS | ~300ms | Great | Cloud | Pay/use |

**Recommendation:** Piper for daily use, ElevenLabs for voice cloning (optional).

## 10.3 Voice Interaction Flow

```
1. Wake word detection ("hey mnemosyne") OR hotkey
2. Audio capture begins
3. User speaks query
4. Silence detection → end capture
5. STT transcription (Whisper)
6. Query processing (retrieve + LLM)
7. Response generation
8. TTS output (Piper)
9. Return to listening (optional follow-up mode)
```

---

# Part 11: Project Structure

```
mnemosyne/
├── cmd/
│   ├── mnemosyne/          # Main daemon
│   │   └── main.go
│   ├── mnemosyne-tui/      # TUI interface
│   │   └── main.go
│   └── mnemosyne-web/      # Web config server
│       └── main.go
│
├── internal/
│   ├── capture/            # All capture sources
│   │   ├── capture.go      # Interface & orchestration
│   │   ├── screen.go       # Screenshot + OCR
│   │   ├── git.go          # Repository state
│   │   ├── window.go       # Active window tracking
│   │   ├── clipboard.go    # Clipboard history
│   │   ├── audio.go        # Microphone capture
│   │   └── files.go        # File access patterns
│   │
│   ├── privacy/            # Privacy & filtering
│   │   ├── gate.go         # Main filter pipeline
│   │   ├── redact.go       # Pattern redaction
│   │   └── rules.go        # Block rules
│   │
│   ├── memory/             # Memory system
│   │   ├── store.go        # Storage abstraction
│   │   ├── models.go       # Memory data structures
│   │   ├── working.go      # Working memory (recent)
│   │   ├── shortterm.go    # Short-term with decay
│   │   ├── longterm.go     # Long-term consolidated
│   │   ├── consolidate.go  # Memory merging/fading
│   │   ├── strength.go     # Decay calculations
│   │   └── embed.go        # Embedding generation
│   │
│   ├── cognition/          # The "thinking" parts
│   │   ├── retrieval.go    # Context → relevant memories
│   │   ├── prediction.go   # Forgetting prediction
│   │   ├── personality.go  # Twin personality model
│   │   ├── reconstruction.go # Rebuild from fragments
│   │   └── surface.go      # Proactive surfacing logic
│   │
│   ├── llm/                # LLM provider abstraction
│   │   ├── provider.go     # Interface
│   │   ├── ollama.go
│   │   ├── openai.go
│   │   ├── anthropic.go
│   │   ├── groq.go
│   │   └── config.go
│   │
│   ├── voice/              # Voice interface
│   │   ├── stt.go          # Speech-to-text (Whisper)
│   │   ├── tts.go          # Text-to-speech (Piper)
│   │   ├── wake.go         # Wake word detection
│   │   └── conversation.go # Voice conversation loop
│   │
│   ├── tui/                # Terminal UI
│   │   ├── app.go
│   │   ├── views/
│   │   └── components/
│   │
│   └── web/                # Web UI
│       ├── server.go
│       ├── handlers/
│       └── templates/
│
├── pkg/
│   └── config/
│       └── config.go
│
├── db/
│   └── migrations/
│
├── configs/
│   └── default.yaml
│
└── scripts/
    ├── install.sh
    └── setup-voice.sh
```

---

# Part 12: Story Flows (User Experience)

## Story 1: The Interrupted Developer

> **9:00 AM** — Atharva sits down, coffee in hand. The display shows: *"Good morning. Yesterday you were debugging the webhook retry logic. Last state: investigating exponential backoff."*

> He didn't have to ask. It knew.

> **9:15 AM** — He's deep in code. A Slack notification pulls him into a meeting.

> **10:30 AM** — Meeting ends. He returns to his desk. The display has updated: *"Before the meeting: worker/retry.go:142. You were testing with max_retries=5. The failing case was network timeout."*

> **10:32 AM** — He's back in flow. The memory reconstructed his context in 2 minutes instead of 20.

## Story 2: The Lost Decision

> **Wednesday, 3:00 PM** — Atharva is reviewing a PR. Someone asks: "Why did we decide to use Postgres instead of CockroachDB?"

> He can't remember. It was weeks ago.

> **3:01 PM** — "Hey Mnemosyne, when did we decide on Postgres?"

> The system responds: *"On January 15th, during your call with Ravi. You discussed operational complexity. The deciding factor was... [plays 10 second anchor] '...CockroachDB's licensing change made us nervous. Let's stick with what we know.'"*

> The actual moment, preserved.

## Story 3: The Reconstructed Week

> **Sunday evening** — Atharva wonders what he accomplished this week. The medication fog makes it all blur together.

> "Mnemosyne, summarize my week."

> *"This week you: merged 3 PRs (auth refactor, queue retry, API versioning), had 4 meetings (2 with Rimo team, 1 with a potential customer, 1 with Rahul), debugged two production issues, and spent Thursday afternoon learning about vector databases. Your most intense focus was Wednesday morning on the auth refactor. You seemed frustrated on Thursday—the queue bug took longer than expected. Want details on any of these?"*

> His week, reconstructed from fragments. The fog lifts a little.

## Story 4: The Proactive Warning

> **Thursday, 2:00 PM** — Atharva finishes a long debugging session. He fixed a critical bug. He doesn't make a note.

> The system detects:
> - High cognitive load (long session, many context switches)
> - Important event (git commit with "fix: critical" in message)
> - Historical pattern: he forgets details of intense sessions within 48 hours

> **Thursday, 2:05 PM** — Soft notification: *"You just fixed the race condition in queue processing. High forgetting risk detected. Want to record why it happened?"*

> He speaks for 30 seconds explaining the bug. Future-proofed.

---

# Part 13: Implementation Phases

## Phase 1: The Daemon (Weeks 1-2)
**Goal:** Proactive context restoration for programming

Build:
- Screen capture daemon
- Git state monitor  
- Window tracking
- Context embedding pipeline
- Simple similarity search
- Desktop notification output

**Deliverable:** A daemon that tells you "before you got distracted, you were doing X in file Y"

## Phase 2: Memory Model (Weeks 3-4)
**Goal:** Human-like decay and consolidation

Build:
- Strength computation with decay
- Reactivation tracking
- Salience calculation
- Nightly consolidation daemon
- Memory competition model

**Deliverable:** Memories that fade, strengthen on access, and merge over time

## Phase 3: TUI + Web UI (Weeks 5-6)
**Goal:** Proper interfaces

Build:
- TUI for daily queries (bubbletea)
- Web UI for configuration (htmx + templ)
- Memory browser
- Privacy rule editor

**Deliverable:** Usable interfaces for daily interaction and configuration

## Phase 4: Voice (Weeks 7-8)
**Goal:** Hands-free interaction

Build:
- Whisper integration for STT
- Piper integration for TTS
- Wake word detection
- Conversation loop

**Deliverable:** "Hey Mnemosyne, what was I working on?"

## Phase 5: Cognitive Twin (Ongoing)
**Goal:** Learn and mimic the user

Build:
- Personality model learning
- Mirror mode responses
- Writing style analysis
- Decision pattern detection

**Deliverable:** A system that can answer "what would I think about this?"

---

# Part 14: Quick Start (Minimal Viable Version)

For immediate relief, build just this:

```
┌────────────────────────────────────────────┐
│           MNEMOSYNE v0.1                   │
│         "The Ugly One That Works"          │
├────────────────────────────────────────────┤
│                                            │
│   Your Laptop                              │
│   ├── Screen capture every 60s             │
│   ├── Git state monitor                    │
│   ├── Active window tracking               │
│   ├── Local embeddings (all-MiniLM)        │
│   ├── SQLite + simple vector search        │
│   └── Desktop notification output          │
│                                            │
│   That's it. No hardware. No voice.        │
│   Just context restoration.                │
│                                            │
│   Cost: ₹0                                 │
│   Time: 2 weeks                            │
│                                            │
└────────────────────────────────────────────┘
```

This solves 80% of the work-context problem immediately.

---

# Part 15: Future Hardware (Optional)

If the software works and you want to extend to life capture:

## The Shard (Wearable Pendant)
- ESP32-S3 + microphone + light sensor + IMU
- Captures ambient audio anchors passively
- ~₹2,000 to build

## The Hearth (Desk Display)
- ESP32-S3 + 2.9" e-ink display + LED ring
- Shows proactive memory surfacing
- Ambient, glanceable, doesn't demand attention
- ~₹3,500 to build

These are Phase 2+ additions. The software must work first.

---

# Appendix A: Key Formulas Summary

```
# Memory strength over time
M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))

# Retrieval probability with competition
P(recall mᵢ) = [Mᵢ(t) · Cᵢ(t)] / [Σⱼ Mⱼ(t) · Cⱼ(t) + β]

# Initial encoding strength
M₀ = α₁·Novelty + α₂·Emotion + α₃·Attention + α₄·PredictionError

# Temporal contiguity
P(recall mⱼ | recalled mᵢ) ∝ e^(-|tᵢ - tⱼ| / τ)

# Memory merging
m⃗_new = α·m⃗₁ + (1-α)·m⃗₂ + ε

# Forgetting risk
Risk = (1 - Strength) × AccessPenalty × SalienceBoost × AgeStability
```

---

# Appendix B: Configuration Reference

See `configs/default.yaml` in the project for full configuration options including:
- Capture intervals and sources
- Memory decay parameters
- Privacy rules
- LLM provider settings
- Voice configuration
- UI preferences

---

# Appendix C: Glossary

- **Salience:** How important/intense a memory is at encoding time
- **Strength:** Current accessibility of a memory (decays over time)
- **Reactivation:** Accessing a memory, which boosts its strength
- **Consolidation:** Merging similar weak memories into summaries
- **Contextual overlap (C(t)):** How similar current context is to a memory's context
- **Forgetting risk:** Predicted probability of losing access to a memory
- **Identity relevance:** How much a memory relates to your self-concept
- **Working memory:** Very recent captures, full detail, no decay
- **Short-term memory:** Hours old, embedded, actively decaying
- **Long-term memory:** Days+, consolidated, merged, abstracted
- **Archived:** Faded memories kept for potential reconstruction, not active retrieval

---

*Document generated from design conversation. Last updated: February 2025*
*Project: Mnemosyne - A Cognitive Prosthesis System*
