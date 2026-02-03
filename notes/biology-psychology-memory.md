# How Memory Actually Works (And How We Steal Its Ideas)

Personal notes on building an external hippocampus. Not formal, just me figuring this out.

---

## The Brain Doesn't Store Files

First thing to unlearn: memory isn't storage. There's no `memories/` folder in your skull.

When you experience something, multiple brain regions light up simultaneously:

```
         EXPERIENCING A CONVERSATION

    ┌──────────────────────────────────────────────────────────────┐
    │                                                              │
    │   Visual Cortex          Auditory Cortex       Amygdala     │
    │   "Ravi's face"          "his voice"           "slight      │
    │   "office lighting"      "AC hum"               anxiety"    │
    │        │                      │                    │        │
    │        │                      │                    │        │
    │        └──────────┬──────────┴──────────┬─────────┘        │
    │                   │                      │                  │
    │                   ▼                      ▼                  │
    │              ┌─────────────────────────────┐                │
    │              │                             │                │
    │              │        HIPPOCAMPUS          │                │
    │              │                             │                │
    │              │   "these all happened       │                │
    │              │    together at 2:30pm       │                │
    │              │    on Tuesday"              │                │
    │              │                             │                │
    │              │   creates INDEX, not copy   │                │
    │              └─────────────────────────────┘                │
    │                            │                                │
    │        Insula              │           Prefrontal           │
    │        "gut feeling"       │           "meaning:            │
    │        "heart racing"      │            deadline            │
    │             │              │            discussion"         │
    │             └──────────────┴──────────────┘                 │
    │                                                              │
    └──────────────────────────────────────────────────────────────┘
```

The hippocampus is like a librarian who doesn't photocopy books — it writes index cards that say "the face is in aisle 3, the voice is in aisle 7, the feeling is in aisle 12, and they all go together."

**This is why damage to the hippocampus doesn't erase old memories** — the books are still on the shelves. You just can't form new index cards.

---

## The Binding Problem

Here's the wild part: these brain regions are physically far apart. Visual cortex is in the back. Auditory is on the sides. Emotional is deep inside. How do they know they're part of the same experience?

```
    BRAIN GEOGRAPHY (very simplified top-down view)

              front
                │
       ┌────────┴────────┐
       │                 │
       │   PREFRONTAL    │  ← planning, meaning, "what does this mean?"
       │    (meaning)    │
       │                 │
    ┌──┴──┐           ┌──┴──┐
    │     │           │     │
    │ L   │           │   R │
    │     │           │     │
    │  TEMPORAL       │     │  ← auditory processing, language
    │  (hearing)      │     │
    │     │           │     │
    └──┬──┘           └──┬──┘
       │    DEEP         │
       │  ┌──────────┐   │
       │  │HIPPOCAMPUS│  │     ← the binder, the indexer
       │  │ AMYGDALA │   │     ← emotional flagging
       │  └──────────┘   │
       │                 │
       │    VISUAL       │     ← seeing, back of head
       │    CORTEX       │
       └────────────────-┘
              back
```

The hippocampus solves this with **theta rhythms** — it creates a temporal binding window. Everything that fires within ~100-500ms gets tagged as "same event." It's like a camera shutter for consciousness.

**Implication for us:** when we capture data, we need to bind signals that happen close in time. A timestamp window of ~1 second is probably fine. Everything in that window = same moment.

---

## Memory Has Stages (Not Just "Saved" or "Not Saved")

```
    THE MEMORY PIPELINE

    EXPERIENCE
        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │  SENSORY MEMORY                                               │
    │  Duration: milliseconds                                       │
    │  Capacity: everything you're sensing right now               │
    │  Loss: immediate decay, you don't even notice                │
    │                                                               │
    │  (most stuff never gets past here)                           │
    └───────────────────────────┬───────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │ ATTENTION GATE        │
                    │ "is this relevant?"   │
                    │ "is this novel?"      │
                    │ "is this emotional?"  │
                    └───────────┬───────────┘
                                │
                                ▼
    ┌───────────────────────────────────────────────────────────────┐
    │  WORKING MEMORY                                               │
    │  Duration: seconds to minutes                                 │
    │  Capacity: ~4-7 items (less on sedatives tbh)                │
    │  Loss: displacement (new stuff pushes out old)               │
    │                                                               │
    │  this is what you're "thinking about" right now              │
    └───────────────────────────┬───────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │ ENCODING GATE         │
                    │ (hippocampus decides) │
                    │ "worth remembering?"  │
                    └───────────┬───────────┘
                                │
                                ▼
    ┌───────────────────────────────────────────────────────────────┐
    │  SHORT-TERM MEMORY (hours)                                    │
    │  Duration: hours to ~1 day                                    │
    │  Storage: hippocampus (the index cards)                      │
    │  State: FRAGILE — not yet consolidated                       │
    │  Loss: decay + interference + failed consolidation           │
    │                                                               │
    │  ⚠️  THIS IS WHERE MEDICATION HITS YOU                       │
    │  sedatives impair consolidation during sleep                 │
    │  memories get stuck here and fade                            │
    └───────────────────────────┬───────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │ CONSOLIDATION         │
                    │ (happens during SLEEP)│
                    │ hippocampus replays   │
                    │ to cortex             │
                    └───────────┬───────────┘
                                │
                                ▼
    ┌───────────────────────────────────────────────────────────────┐
    │  LONG-TERM MEMORY (days to lifetime)                         │
    │  Duration: days, years, lifetime                             │
    │  Storage: distributed across cortex                          │
    │  State: STABLE but can still change on recall                │
    │                                                               │
    │  episodic: "what happened" — specific events                 │
    │  semantic: "what I know" — facts, concepts, merged patterns  │
    │  procedural: "how to do things" — skills, habits             │
    └───────────────────────────────────────────────────────────────┘
```

---

## The Forgetting Curve (Ebbinghaus, 1885)

German psychologist Hermann Ebbinghaus memorized nonsense syllables and tested himself. He found:

```
MEMORY RETENTION OVER TIME (classic Ebbinghaus curve)

100% │▓▓▓▓
     │▓▓▓▓▓
     │ ▓▓▓▓▓▓
 75% │  ▓▓▓▓▓▓▓
     │    ▓▓▓▓▓▓▓▓
     │      ▓▓▓▓▓▓▓▓▓
 50% │         ▓▓▓▓▓▓▓▓▓▓
     │            ▓▓▓▓▓▓▓▓▓▓▓
     │               ▓▓▓▓▓▓▓▓▓▓▓▓▓▓
 25% │                  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
     │                     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
     │                        ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  0% └─────────────────────────────────────────────────────────────
        20m   1h    9h   1d   2d    6d    31d

        RAPID DROP INITIALLY, THEN LONG TAIL
```

The formula:

```
R(t) = e^(-t/S)

where:
  R(t) = retention at time t
  S    = stability (higher = slower forgetting)
  t    = time since encoding
```

But this is too simple. It assumes:
- No reactivation (but we recall things!)
- Same decay for everything (but emotional stuff lasts longer!)
- No interference (but similar memories compete!)

---

## What Actually Affects Encoding Strength?

Not all memories are born equal. The initial strength `M₀` depends on:

```
M₀ = f(attention, emotion, novelty, prediction_error)

or more specifically:

M₀ = α₁·A + α₂·E + α₃·N + α₄·ΔP
```

Let's see this in action:

```
ENCODING STRENGTH EXAMPLES

                          Attention  Emotion  Novelty  Pred.Error  →  M₀
                          ─────────  ───────  ───────  ──────────     ───
Random lecture               0.2       0.1      0.1       0.0        0.10
                             (zoning   (bored)  (heard    (expected)
                              out)              before)

First day at new job         0.9       0.6      0.9       0.7        0.78
                             (hyper-   (anxious (every-   (nothing
                              alert)   +excited) thing     is as
                                        new)     expected)

Getting fired                0.7       0.95     0.8       0.95       0.85
                             (shocked  (fear,   (never    (didn't
                              focus)   anger)   happened)  see it
                                                          coming)

Routine standup #47          0.3       0.1      0.05      0.05       0.13
                             (half     (meh)    (same     (same
                              there)            as always) as always)

Bug that took 4 hours        0.8       0.7      0.3       0.6        0.60
                             (deep     (frust-  (bugs     (weird
                              focus)   ration)  happen)   behavior)
```

**Implication for the system:** we can estimate these signals:
- Attention: session duration, typing patterns, lack of tab switches
- Emotion: sentiment analysis, keywords ("fuck", "finally!", "error")
- Novelty: embedding distance from recent memories
- Prediction error: anomalies, error messages, unexpected git states

---

## Reactivation: Remembering Changes the Memory

Here's the weird part: every time you recall a memory, you **rewrite** it.

```
MEMORY AFTER MULTIPLE RECALLS

Original encoding (Monday)
        │
        │  strength = 1.0
        ▼
       [M]────────────────────────────────────────── decay
        │                                            ↓
        │                                         strength = 0.4
        ▼
    RECALL (Wednesday)
        │
        │  reconsolidation — memory becomes malleable again
        │  current emotional state colors the recall
        │  details get filled in (sometimes wrong)
        │  strength BOOSTED back up
        │
        ▼
       [M']───────────────────────────────────────── decay
        │                                            ↓
        │  this is now a memory OF THE RECALL       strength = 0.5
        │  not purely the original event
        ▼
    RECALL (Friday)
        │
        │  again: rewrite, boost, color
        ▼
       [M'']─────────────────────────────────────────
        │
        │  now it's a memory of a memory of a memory
        │  telephone game with yourself
```

The formula with reactivation:

```
M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))

where:
  M₀     = initial encoding strength
  λ      = decay rate
  t      = time since creation
  Rᵢ     = boost from i-th reactivation
  tᵢ     = time of i-th reactivation
```

Graph:

```
STRENGTH OVER TIME WITH REACTIVATIONS

1.0 │ ▓
    │ ▓▓
    │  ▓▓
    │   ▓▓▓
0.7 │    ▓▓▓       ▓▓  ← recall! boost!
    │      ▓▓▓    ▓▓▓▓
    │        ▓▓▓ ▓▓  ▓▓▓
0.4 │          ▓▓▓     ▓▓▓       ▓▓  ← another recall
    │                    ▓▓▓   ▓▓▓▓
    │                      ▓▓▓▓▓  ▓▓▓▓
0.1 │                              ▓▓▓▓▓▓▓▓▓▓▓▓▓▓
    └──────────────────────────────────────────────
      day1    day3    day5    day7    day14   day21

      WITHOUT RECALLS: would be at ~0.05 by day21
      WITH RECALLS: still at ~0.15, much more accessible
```

**Implication for the system:**
- Every time we surface a memory to you, we record that as a reactivation
- This boosts its strength, making it more likely to survive
- The system can do spaced repetition automatically — surface fading memories before they're gone

---

## Interference: Memories Compete

You don't forget in a vacuum. Similar memories **fight for activation**.

```
MEMORY INTERFERENCE

Query: "what did Ravi say about the deadline?"

Your brain:
                          ┌─────────────────────────┐
    ┌──────────────────┐  │                         │
    │ Meeting Monday   │──┼──┐                      │
    │ deadline: Friday │  │  │                      │
    └──────────────────┘  │  │   ┌──────────────┐   │
                          │  │   │              │   │
    ┌──────────────────┐  │  ├──►│  RETRIEVAL   │   │
    │ Meeting Tuesday  │──┼──┤   │  COMPETITION │   │
    │ deadline: Friday │  │  │   │              │   │
    └──────────────────┘  │  │   │  "which one  │   │
                          │  │   │   was it?"   │   │
    ┌──────────────────┐  │  │   │              │   │
    │ Meeting Thursday │──┼──┘   └──────────────┘   │
    │ deadline: Monday │  │             │           │
    └──────────────────┘  │             ▼           │
                          │      TIP OF TONGUE      │
                          │      or wrong answer    │
                          └─────────────────────────┘
```

The more similar memories you have, the harder it is to retrieve a specific one. This is called **proactive interference** (old blocks new) and **retroactive interference** (new blocks old).

Formula for retrieval probability with competition:

```
P(recall mᵢ) = Aᵢ / (Σⱼ Aⱼ + β)

where:
  Aᵢ = activation of memory i = strength × context_match
  β  = noise floor (prevents division by zero, models "tip of tongue")
```

Example:

```
Query: "deadline meeting with Ravi"

Memory A (Monday):    activation = 0.6 × 0.9 = 0.54
Memory B (Tuesday):   activation = 0.5 × 0.9 = 0.45
Memory C (Thursday):  activation = 0.7 × 0.7 = 0.49
Noise β = 0.1

P(recall A) = 0.54 / (0.54 + 0.45 + 0.49 + 0.1) = 0.54 / 1.58 = 34%
P(recall B) = 0.45 / 1.58 = 28%
P(recall C) = 0.49 / 1.58 = 31%
P(tip of tongue) = 0.1 / 1.58 = 6%

YOU'RE BASICALLY GUESSING.
High competition = low confidence.
```

**Implication for the system:**
- We don't just find "best match" — we compute competition
- If competition is high, we surface the **merged gist** instead of a specific memory
- Or we surface distinguishing context: "Monday 2pm vs Tuesday 10am — which one?"

---

## Context-Dependent Retrieval (Why Smells Trigger Memories)

A memory isn't just content. It's content **bound to context**.

```
MEMORY STRUCTURE

    ┌─────────────────────────────────────────────────────────┐
    │  MEMORY: "debugging the webhook retry bug"              │
    ├─────────────────────────────────────────────────────────┤
    │                                                         │
    │  CONTENT (what happened)                                │
    │  ├── looking at worker/retry.go                        │
    │  ├── testing with max_retries=5                        │
    │  └── the timeout was 30s, should be 60s                │
    │                                                         │
    │  CONTEXT (the binding)                                  │
    │  ├── app: VSCode                                       │
    │  ├── file: worker/retry.go                             │
    │  ├── time: afternoon                                   │
    │  ├── location: home office                             │
    │  ├── music: lo-fi playlist                             │
    │  ├── emotional state: frustrated → satisfied           │
    │  ├── physical state: slouched, tense shoulders         │
    │  ├── who was around: alone                             │
    │  └── what came before: standup meeting                 │
    │                                                         │
    └─────────────────────────────────────────────────────────┘
```

Retrieval probability depends on **context overlap**:

```
P(recall) ∝ M(t) × C(t)

where C(t) = how similar current context is to memory's context
```

This is why:
- A **smell** can unlock a memory (olfactory context match)
- You **remember better in the same room** you learned in (location match)
- **Music** from a period brings back that period (auditory context)
- Being in **the same emotional state** helps (state-dependent memory)

```
CONTEXT OVERLAP EXAMPLE

Memory context:           Current context:         Overlap:
─────────────────────────────────────────────────────────────
app: VSCode               app: VSCode              ✓  0.15
file: worker/retry.go     file: api/handlers.go    ✗  0.00
time: afternoon           time: afternoon          ✓  0.05
location: home            location: home           ✓  0.05
music: lo-fi              music: lo-fi             ✓  0.10
emotional: frustrated     emotional: calm          ✗  0.00
physical: tense           physical: relaxed        ✗  0.00
                                          ─────────────────
                                          C(t) = 0.35

Low-medium overlap. Memory might surface weakly.
If you opened the same file → big context boost → memory pops up.
```

**Implication for the system:**
- We store rich context with every memory
- Retrieval uses context similarity, not just content similarity
- When you return to a similar context, relevant memories activate
- This is how we give you "retrieval cues" your brain isn't generating

---

## Sleep Consolidation (The Overnight Batch Job)

During slow-wave sleep, the hippocampus replays the day's memories to the cortex. Not all make it.

```
WHAT SLEEP DOES TO MEMORIES

        BEFORE SLEEP                         AFTER SLEEP
        ────────────                         ───────────

    ┌──────────────────┐                 ┌──────────────────┐
    │ standup meeting  │─────────────────│                  │
    │ (routine, low M₀)│   PRUNED        │                  │
    └──────────────────┘                 │                  │
                                         │                  │
    ┌──────────────────┐                 │  "meetings this  │
    │ meeting w/ Ravi  │─────┐           │   week were      │
    │ (important, M₀=.7)│     │  MERGED  │   about the      │
    └──────────────────┘      ├─────────►│   deadline"      │
    ┌──────────────────┐      │          │                  │
    │ meeting w/ client│─────┘           │  (gist survives, │
    │ (important, M₀=.6)│                │   details fade)  │
    └──────────────────┘                 │                  │
                                         └──────────────────┘
    ┌──────────────────┐                 ┌──────────────────┐
    │ fixed that bug!  │                 │ fixed that bug!  │
    │ (emotional, M₀=.8)│ ──BOOSTED────► │ (now in cortex)  │
    │                  │                 │ stronger, stable │
    └──────────────────┘                 └──────────────────┘

    ┌──────────────────┐
    │ random browsing  │─────────────────── GONE
    │ (low attention)  │   PRUNED
    └──────────────────┘
```

Consolidation does three things:
1. **Prunes** weak, irrelevant memories
2. **Merges** similar memories into schemas/gists
3. **Strengthens** emotionally significant or frequently-accessed memories

**Implication for the system:**
- We run a "consolidation daemon" nightly
- Clusters similar weak memories → LLM generates merged summary
- Archives raw memories, keeps summary active
- Boosts memories that survived the day with high strength

---

## My Specific Problem (Medication + Memory)

SSRIs and sedatives hit memory in specific ways:

```
WHERE MEDICATION INTERFERES

    ENCODING          STORAGE           RETRIEVAL
    ────────          ───────           ─────────
        │                 │                 │
        │                 │                 │
    attention ◄───┐       │                 │
    (sedatives    │       │                 │
     reduce this) │       │                 │
        │         │       │                 │
        ▼         │       │                 │
    ┌───────┐     │       │                 │
    │working│     │   ┌───────────┐         │
    │memory │─────┼──►│ consolida-│         │
    │       │     │   │ tion      │         │
    └───────┘     │   │           │         │
        │         │   │ sedatives │         │
        │         │   │ impair    │         │
        │         │   │ slow-wave │         │
        │         │   │ sleep     │         │
        │         │   └───────────┘         │
        │         │        │                │
        │         │        ▼                │
        │         │   memories get          │
        │         │   stuck, don't          │
        │         │   transfer to           │
        │         │   long-term             │
        │         │        │                │
        │         │        ▼                │
        │         │   details blur ◄────────┤
        │         │   together     (interference
        │         │   (interference  worse when
        │         │    increases)   retrieval
        │         │                 cues are weak)
        │         │                         │
        └─────────┴─────────────────────────┘

    THE RESULT:
    - encoding is OK (you experience things normally)
    - storage is impaired (consolidation suffers)
    - retrieval is impaired (weak cues, high interference)

    you have the memories. you just can't reach them.
```

---

## What Mnemosyne Does About This

```
COMPENSATION STRATEGY

    YOUR BRAIN (impaired)              MNEMOSYNE (external)
    ─────────────────────              ────────────────────

    hippocampus struggles    ────►     daemon captures continuously
    to form new bindings               binds context automatically
                                       timestamps everything

    consolidation impaired   ────►     nightly consolidation daemon
    during sleep                       clusters, merges, prunes
                                       simulates what sleep should do

    retrieval cues weak      ────►     context-based surfacing
    can't self-trigger                 "before the meeting you were..."
                                       gives you the cue you can't generate

    interference high        ────►     timestamps + specific context
    similar memories blur              "Monday 2pm" vs "Tuesday 10am"
                                       helps differentiate

    forgetting accelerated   ────►     proactive surfacing
    important stuff fades              "you're about to forget this"
                                       reactivates before it's gone
```

---

## The Pendant: Capturing What the Brain Captures

The hippocampus binds multiple streams. The pendant captures multiple streams:

```
HIPPOCAMPUS INPUTS                    PENDANT SENSORS
──────────────────                    ───────────────

Visual cortex (what you see)          [camera - maybe, privacy issues]
                                      [skip for v1, use phone/laptop]

Auditory cortex (what you hear)       [MEMS microphone]
                                      ambient audio, voices, environment

Spatial (where you are)               [BLE beacons, GPS via phone]
                                      room-level or city-level location

Temporal (when)                       [timestamp]
                                      automatic, precise

Emotional arousal (amygdala)          [heart rate sensor - MAX30102]
                                      elevated HR = flag for importance

Body state (insula)                   [IMU - accelerometer + gyro]
                                      sitting, walking, fidgeting, gestures

Environmental                         [light sensor, temperature]
                                      indoor/outdoor, time-of-day quality
```

The binding happens in software:

```
BINDING EXAMPLE

Timestamp: 2025-02-04 14:32:00 (± 1 second window)
├── audio: "...so the deadline is definitely Friday..."
├── motion: sitting, occasional gestures (engaged)
├── heart_rate: 82 bpm (slightly elevated)
├── light: indoor, artificial
├── temperature: 24°C
├── location: office-beacon-3rd-floor
└── phone_data: no active call, Slack in foreground

        │
        │  DAEMON BINDS THESE
        ▼

┌────────────────────────────────────────────────────────────────┐
│  MEMORY OBJECT                                                 │
│                                                                │
│  content: "conversation about Friday deadline"                 │
│  transcript: "...so the deadline is definitely Friday..."      │
│  embedding: [0.23, -0.41, 0.18, ...]                          │
│                                                                │
│  context:                                                      │
│    location: office/3rd-floor                                  │
│    time_of_day: afternoon                                      │
│    activity: conversation                                      │
│    physical_state: seated, engaged                             │
│    emotional_state: slightly_elevated (HR)                     │
│    environment: indoor, warm                                   │
│                                                                │
│  encoding_factors:                                             │
│    attention: 0.7 (engaged posture)                           │
│    emotion: 0.4 (elevated HR)                                  │
│    novelty: 0.3 (deadline discussed before)                   │
│    prediction_error: 0.2 (expected topic)                     │
│                                                                │
│  M₀: 0.45                                                      │
│  salience: 0.55 (deadline keyword boost)                      │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

---

## Retrieval Flow: Giving You Back Your Memories

```
YOU: "what was the deadline Ravi mentioned?"

        │
        ▼
    ┌─────────────────────────┐
    │ QUERY UNDERSTANDING     │  (LLM)
    │ extract: "deadline"     │
    │         "Ravi"          │
    │         "mentioned"     │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ EMBED QUERY             │  (algorithm)
    │ → [0.31, -0.28, ...]    │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ VECTOR SEARCH           │  (algorithm)
    │ find top-50 by cosine   │
    │ similarity              │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ COMPUTE ACTIVATION      │  (algorithm)
    │ for each candidate:     │
    │   A = M(t) × C(t)       │
    │   (strength × context)  │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ COMPETITION MODEL       │  (algorithm)
    │ P(mᵢ) = Aᵢ / (ΣAⱼ + β) │
    │                         │
    │ if competition high:    │
    │   → surface gist        │
    │ else:                   │
    │   → surface top match   │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ RECONSTRUCTION          │  (LLM, if needed)
    │ if memory is sparse:    │
    │   expand from fragments │
    │   fill in context       │
    └───────────┬─────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │ RECORD REACTIVATION     │  (algorithm)
    │ boost strength          │
    │ for surfaced memories   │
    └───────────┬─────────────┘
                │
                ▼

    RESPONSE: "On Tuesday at 2:30pm, Ravi said the deadline
               is Friday. You seemed a bit tense during this
               conversation (elevated heart rate). This was
               in the office, 3rd floor."
```

---

## Summary: The Pipeline

```
    ┌──────────────────────────────────────────────────────────────┐
    │                                                              │
    │  REAL WORLD ──► PENDANT ──► DAEMON ──► STORAGE              │
    │                                                              │
    │       │            │           │           │                │
    │       │            │           │           │                │
    │    senses       sensors     binding     memories            │
    │    (limited)    (extended)  (simulated  (persistent)        │
    │                             hippocampus)                    │
    │       │            │           │           │                │
    │       ▼            ▼           ▼           ▼                │
    │                                                              │
    │  YOUR BRAIN                 MNEMOSYNE                       │
    │  - impaired consolidation   - nightly consolidation daemon  │
    │  - weak retrieval cues      - context-based surfacing       │
    │  - high interference        - timestamps + differentiation  │
    │  - memories fade            - proactive reactivation        │
    │                                                              │
    │       │                         │                           │
    │       └────────────┬────────────┘                           │
    │                    │                                        │
    │                    ▼                                        │
    │                                                              │
    │              YOU, WITH BETTER MEMORY                        │
    │              (or at least, better access)                   │
    │                                                              │
    └──────────────────────────────────────────────────────────────┘
```

---

## References (for my own digging later)

- Ebbinghaus, H. (1885). Memory: A Contribution to Experimental Psychology.
- Anderson, J. R. (1990). The Adaptive Character of Thought. (ACT-R model)
- Squire, L. R. (2004). Memory systems of the brain.
- Walker, M. P. (2017). Why We Sleep. (sleep consolidation)
- Tulving, E. (1972). Episodic and semantic memory.
- Godden & Baddeley (1975). Context-dependent memory. (the diver study)

---

*Notes from figuring out how to build an external hippocampus. February 2025.*
