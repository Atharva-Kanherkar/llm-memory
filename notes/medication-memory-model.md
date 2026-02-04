# How Your Medications Affect Memory (With Math)

Personal notes on modeling medication effects on memory. Based on my actual prescription and research into how each drug hits the memory system.

---

## My Current Stack

| Drug | Dose | Time | What It Does |
|------|------|------|--------------|
| **Sertraline** (Lupisert) | 150mg (50+100) | morning + evening | SSRI - serotonin reuptake inhibitor |
| **Zolpidem** (Sove IT) | 6.25mg | bedtime | Z-drug sedative - GABA-A α1 agonist |
| **Oxcarbazepine** (Oxetol XR) | 450mg | evening | Anticonvulsant - sodium channel blocker |
| **Tofisopam** (Toficalm) | 50mg | morning | Atypical anxiolytic - PDE inhibitor |
| **Sompraz D** | 40mg | evening | PPI + prokinetic (gut, not brain) |

Three of these hit memory. One actually helps. Let me break it down.

---

## Part 1: The Baseline Memory Model (No Meds)

First, let's establish what "normal" looks like mathematically.

### Encoding Strength (M₀)

When you experience something, how strongly it gets encoded:

```
M₀ = α₁·A + α₂·E + α₃·N + α₄·ΔP

where:
  A  = Attention (0-1) — were you focused?
  E  = Emotional arousal (0-1) — amygdala activation
  N  = Novelty (0-1) — how different from recent experience?
  ΔP = Prediction error (0-1) — was it unexpected?

  α₁, α₂, α₃, α₄ = weights (typically sum to ~1)
```

**Typical healthy weights:**
```
α₁ (attention)  = 0.30
α₂ (emotion)    = 0.35  ← emotional tagging is powerful
α₃ (novelty)    = 0.20
α₄ (pred.error) = 0.15
```

### Memory Strength Over Time

The Ebbinghaus-style decay with reactivation:

```
M(t) = M₀ · e^(-λt) + Σᵢ Rᵢ · e^(-λ(t - tᵢ))

where:
  λ  = decay constant (~0.1 per hour for unconsolidated memories)
  Rᵢ = boost from i-th reactivation (~0.1-0.3)
  tᵢ = time of reactivation
```

### Consolidation Efficiency

During sleep, memories transfer from hippocampus to cortex:

```
C_eff = f(SWS_time, spindle_density, REM_time)

Simplified:
C_eff = β₁·SWS + β₂·σ + β₃·REM

where:
  SWS = slow-wave sleep time (hours)
  σ   = spindle density (events/minute during NREM)
  REM = REM sleep time (hours)

Typical healthy values:
  SWS ≈ 1.5-2 hours/night
  σ   ≈ 2-3 spindles/minute
  REM ≈ 1.5-2 hours/night
  C_eff ≈ 0.7-0.8 (70-80% of salient memories consolidate)
```

### Post-Sleep Decay

Consolidated memories decay MUCH slower:

```
λ_consolidated ≈ λ_fresh / 10

Fresh memory: λ = 0.1/hour → half-life ≈ 7 hours
Consolidated:  λ = 0.01/hour → half-life ≈ 70 hours (≈3 days)
```

---

## Part 2: What Each Medication Does

### Sertraline (SSRI) — The Emotional Dampener

**Mechanism:**
- Blocks serotonin reuptake → more 5-HT in synaptic cleft
- Reduces amygdala reactivity to emotional stimuli
- Inhibits hippocampal LTP (long-term potentiation)
- Suppresses REM sleep
- Long-term: may increase hippocampal neurogenesis

**Research findings:**
- SSRIs inhibit hippocampal LTP in majority of studies ([PMC5002481])
- Emotional blunting = reduced amygdala activation ([CUNY Academic Works])
- SSRI responders show decreased amygdala activity to negative stimuli ([ScienceDirect])
- May accelerate cognitive decline in late-life, but I'm 21, different context ([Psychiatry Research])

**Mathematical effect:**

```
ENCODING IMPAIRMENT:

E_effective = E × δ_sertraline

where δ_sertraline ≈ 0.5-0.7 (30-50% reduction in emotional encoding)
```

The amygdala normally "flags" emotional events for priority encoding. With sertraline, this flag is dimmer.

```
         NORMAL                          ON SERTRALINE

    Emotional event                   Emotional event
          │                                 │
          ▼                                 ▼
    ┌───────────┐                    ┌───────────┐
    │  AMYGDALA │                    │  AMYGDALA │
    │  response │                    │  response │
    │    0.8    │                    │    0.5    │ ← dampened
    └─────┬─────┘                    └─────┬─────┘
          │                                │
          ▼                                ▼
    Strong encoding                  Weaker encoding
    M₀ = 0.75                        M₀ = 0.55
```

**LTP inhibition:**

```
LTP_efficiency = LTP_baseline × δ_ltp

where δ_ltp ≈ 0.7-0.8 (SSRIs reduce LTP in hippocampus)
```

This means even non-emotional memories encode slightly weaker.

**REM suppression:**

```
REM_time = REM_baseline × δ_rem

where δ_rem ≈ 0.6-0.8 (SSRIs reduce REM by 20-40%)
```

REM is important for emotional memory consolidation. Less REM = emotional memories consolidate worse.

---

### Zolpidem (Z-Drug) — The Sleep Architect

This one is complicated. Research shows mixed effects.

**Mechanism:**
- GABA-A receptor agonist, specifically at α1 subunit
- INCREASES slow-wave sleep
- INCREASES spindle-slow wave coupling
- DECREASES REM sleep
- Can cause anterograde amnesia if awake after taking
- Next-day cognitive effects (attention, verbal memory)

**Research findings:**
- Greater memory improvement after zolpidem vs placebo for declarative memory ([SLEEP Oxford])
- More SWS, less REM sleep compared to placebo ([PMC8064806])
- Enhances hippocampal-prefrontal coupling during NREM ([Nature])
- Dose-dependent anterograde amnesia ([PMC3657033])
- Next-day verbal memory and attention deficits ([PMC3280925])
- 6.25mg is low dose, but effects still present

**The paradox:**

```
ZOLPIDEM EFFECTS ON MEMORY

    POSITIVE                         NEGATIVE
    ────────                         ────────
    ↑ SWS time                       ↓ REM time
    ↑ Spindle density                Anterograde amnesia window
    ↑ Spindle-SO coupling            Next-day attention deficit
    ↑ Declarative consolidation      Next-day verbal memory deficit
                                     No tolerance develops
```

**Mathematical model:**

```
CONSOLIDATION EFFECTS:

C_eff_zolpidem = β₁·(SWS × 1.3) + β₂·(σ × 1.2) + β₃·(REM × 0.7)
                     ↑ increased      ↑ increased    ↑ decreased

Net effect on DECLARATIVE memory consolidation: may be POSITIVE
Net effect on EMOTIONAL memory consolidation: NEGATIVE (less REM)
```

**Anterograde amnesia window:**

```
P(encoding_failure) = f(time_since_dose, blood_concentration)

For 6.25mg zolpidem:
  Peak concentration: ~1.5 hours post-dose
  Half-life: ~2.5 hours

  P(amnesia) at 1 hour post-dose: ~0.3
  P(amnesia) at 2 hours post-dose: ~0.2
  P(amnesia) at 8 hours (wake): ~0.05

  DANGER ZONE: 0-3 hours post-dose
  anything experienced in this window may not encode
```

```
TIMELINE OF A NIGHT

    10pm        11pm        2am         6am         8am
      │           │           │           │           │
      ▼           ▼           ▼           ▼           ▼
    ┌───────────────────────────────────────────────────┐
    │ TAKE      │ AMNESIA   │           │           │   │
    │ ZOLPIDEM  │ WINDOW    │  SLEEP    │ RESIDUAL  │   │
    │           │ (danger)  │ (consol.) │ EFFECTS   │   │
    └───────────────────────────────────────────────────┘

    If something important happens at 11pm and you're
    still awake → may not remember it

    Morning: attention and verbal memory slightly impaired
```

**Next-day effects:**

```
A_morning = A_baseline × δ_next_day

where δ_next_day ≈ 0.85-0.95 (5-15% attention reduction)

This compounds with other effects.
```

---

### Oxcarbazepine (Anticonvulsant) — The Neural Dampener

**Mechanism:**
- Blocks voltage-gated sodium channels
- Reduces neuronal excitability
- Slows neural transmission speed

**Research findings:**
- Generally considered cognitively benign ([PMC3229254])
- Some studies show improved attention, processing speed ([PubMed 8405007])
- 20% of patients report mild memory issues ([PMC2686935])
- Better than carbamazepine and valproate for cognition

**Mathematical effect:**

```
PROCESSING SPEED:

τ_processing = τ_baseline × (1 + δ_oxc)

where δ_oxc ≈ 0.05-0.15 (5-15% slower processing)
```

This affects encoding indirectly:

```
A_effective = A × (1 - δ_oxc/2)

Slower processing → slightly lower effective attention
But effect is SMALL compared to other meds
```

---

### Tofisopam (Atypical Anxiolytic) — The Surprise Helper

**Mechanism:**
- 2,3-benzodiazepine (different from typical 1,4-benzos)
- Does NOT bind to classic benzodiazepine site
- Inhibits PDE-4, PDE-10, possibly PDE-2
- No sedative, amnestic, or muscle relaxant properties
- Mild cognitive stimulant effects

**Research findings:**
- ANTI-amnesic effects demonstrated in rat studies ([PubMed 31981560])
- Improved hippocampal synaptogenesis and neurogenesis
- Enhanced logical memory and verbal reasoning in anxious subjects
- Does NOT impair psychomotor or intellectual performance ([ScienceDirect])
- Mild cognitive stimulatory activity

**This is the good news:**

```
TOFISOPAM EFFECTS:

A_effective = A × (1 + δ_tofisopam)

where δ_tofisopam ≈ 0.05-0.10 (5-10% attention BOOST)

Plus: reduced anxiety → less interference from anxious thoughts
Plus: hippocampal neurogenesis support
```

This partially compensates for other meds. You're lucky it's in the stack.

---

## Part 3: The Combined Model

Now let's put it all together.

### Modified Encoding Strength

```
BASELINE:
M₀ = α₁·A + α₂·E + α₃·N + α₄·ΔP

ON YOUR MEDICATION STACK:
M₀_med = α₁·A_eff + α₂·E_eff + α₃·N + α₄·ΔP

where:
  A_eff = A × δ_attention_combined
  E_eff = E × δ_emotion_combined
```

**Calculating combined attention effect:**

```
δ_attention_combined = δ_sertraline_a × δ_zolpidem_a × δ_oxc_a × δ_tofisopam_a

                     = 1.0 × 0.90 × 0.95 × 1.08

                     ≈ 0.92

(sertraline doesn't directly hit attention much)
(zolpidem: 10% reduction next-morning)
(oxcarbazepine: 5% reduction)
(tofisopam: 8% BOOST)

NET: ~8% attention reduction
```

**Calculating combined emotion effect:**

```
δ_emotion_combined = δ_sertraline_e × δ_tofisopam_e

                   = 0.60 × 0.95

                   ≈ 0.57

(sertraline: 40% reduction in emotional tagging)
(tofisopam: 5% reduction from anxiety dampening)

NET: ~43% emotional encoding reduction
```

**Final encoding equation:**

```
M₀_med = 0.30·(A × 0.92) + 0.35·(E × 0.57) + 0.20·N + 0.15·ΔP

Simplified:
M₀_med = 0.28·A + 0.20·E + 0.20·N + 0.15·ΔP
                    ↑
            emotional weight dropped from 0.35 to 0.20
```

**Example comparison:**

```
SCENARIO: Important conversation with deadline info

                              BASELINE        ON MEDS
                              ────────        ───────
Attention (A)                   0.7             0.7
Emotional arousal (E)           0.6             0.6
Novelty (N)                     0.3             0.3
Prediction error (ΔP)           0.4             0.4

ENCODING STRENGTH:

Baseline:
M₀ = 0.30(0.7) + 0.35(0.6) + 0.20(0.3) + 0.15(0.4)
   = 0.21 + 0.21 + 0.06 + 0.06
   = 0.54

On meds:
M₀ = 0.28(0.7) + 0.20(0.6) + 0.20(0.3) + 0.15(0.4)
   = 0.20 + 0.12 + 0.06 + 0.06
   = 0.44

REDUCTION: 0.54 → 0.44 = 18.5% weaker initial encoding
```

### Modified Decay

**Baseline decay:**
```
λ_baseline = 0.1 per hour (unconsolidated)
```

**On meds:**

The LTP inhibition from sertraline means memories don't stabilize as well:

```
λ_med = λ_baseline × (1 + δ_ltp_impairment)
      = 0.1 × (1 + 0.15)
      = 0.115 per hour

15% faster decay before consolidation
```

### Modified Consolidation

**Baseline:**
```
C_eff_baseline = 0.75 (75% of salient memories consolidate)
```

**On meds:**

```
Sleep architecture changes:

SWS:  baseline × 1.3 (zolpidem increases SWS)
σ:    baseline × 1.2 (zolpidem increases spindle density)
REM:  baseline × 0.6 (sertraline + zolpidem both suppress)

C_eff_med for DECLARATIVE memories:
  = 0.4·(1.3) + 0.3·(1.2) + 0.3·(0.6)
  = 0.52 + 0.36 + 0.18
  = 1.06 relative to baseline
  → slightly BETTER consolidation for facts/events

C_eff_med for EMOTIONAL memories:
  = 0.2·(1.3) + 0.2·(1.2) + 0.6·(0.6)
  = 0.26 + 0.24 + 0.36
  = 0.86 relative to baseline
  → 14% WORSE consolidation for emotional memories
```

**The split:**

```
┌─────────────────────────────────────────────────────────────────┐
│  MEMORY TYPE          │  ENCODING  │  CONSOLIDATION  │  NET    │
├───────────────────────┼────────────┼─────────────────┼─────────┤
│  Neutral facts        │    -8%     │     +6%         │   -2%   │
│  Emotional events     │   -43%     │    -14%         │  -50%   │
│  Routine/boring       │    -5%     │     +6%         │   +1%   │
└─────────────────────────────────────────────────────────────────┘

Your brain is BETTER at remembering boring stuff
and WORSE at remembering important emotional stuff

This is backwards from how memory should work.
```

---

## Part 4: Time-of-Day Effects

Your meds create a daily rhythm:

```
TIME-OF-DAY MEMORY CAPACITY

        6am   9am   12pm  3pm   6pm   9pm   11pm  2am
        │     │     │     │     │     │     │     │
Encoding│     │     │     │     │     │     │     │
Capacity│                                         │
   100% │      ████████████████                   │
        │     █              ██                   │
    80% │    █                 █                  │
        │   █                   █                 │
    60% │  █                     █                │
        │ █                       █               │
    40% │█                         ██             │
        │                           ██████        │
    20% │                                 ████████│← amnesia zone
        └─────────────────────────────────────────┘

LEGEND:
  6-9am:   Zolpidem residual effects wearing off
  9am-6pm: Best window (tofisopam active, others steady)
  6pm:     Sertraline evening dose kicks in
  9pm+:    Oxcarbazepine effects accumulate
  11pm+:   Zolpidem taken → danger zone
```

**Mathematical model of daily encoding capacity:**

```python
def encoding_capacity(hour):
    """
    Returns encoding efficiency multiplier (0-1) by hour of day.
    Based on medication timing.
    """
    # Tofisopam boost (morning, half-life ~6-8 hours)
    tofisopam = 1.08 * exp(-(hour - 8)**2 / 50) if hour >= 6 else 1.0

    # Zolpidem residual (morning impairment)
    zolpidem_residual = 0.90 + 0.10 * (1 - exp(-(hour - 6)**2 / 20))

    # Zolpidem active (evening amnesia window)
    if hour >= 23 or hour < 2:
        zolpidem_active = 0.3  # danger zone
    elif hour >= 22:
        zolpidem_active = 0.6
    else:
        zolpidem_active = 1.0

    # Sertraline (relatively constant, slight evening dip)
    sertraline = 0.95 if hour >= 20 else 1.0

    # Oxcarbazepine (evening accumulation)
    oxc = 0.95 if hour >= 21 else 1.0

    return tofisopam * zolpidem_residual * zolpidem_active * sertraline * oxc
```

---

## Part 5: Where The Damage Hits

Let me graph where the impairment actually hurts:

```
THE MEMORY PIPELINE - WHERE MEDS HIT

    EXPERIENCE
        │
        ▼
    ┌─────────────────────────────────────────────────────────┐
    │  ATTENTION GATE                                         │
    │                                                         │
    │  [OXCARBAZEPINE: -5%] [ZOLPIDEM MORNING: -10%]         │
    │  [TOFISOPAM: +8%]                                      │
    │                                                         │
    │  NET: -7% to -2% depending on time                     │
    └───────────────────────────┬─────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────┐
    │  EMOTIONAL TAGGING (Amygdala)                          │
    │                                                         │
    │  ██████████████████████████████████████████████████    │
    │  ██  SERTRALINE: -40% emotional encoding          ██   │
    │  ██  This is the BIG hit                          ██   │
    │  ██████████████████████████████████████████████████    │
    │                                                         │
    │  Important things don't FEEL important enough to stick │
    └───────────────────────────┬─────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────┐
    │  HIPPOCAMPAL ENCODING (LTP)                            │
    │                                                         │
    │  [SERTRALINE: inhibits LTP, -15% encoding efficiency]  │
    │                                                         │
    │  Synaptic strengthening is weakened                    │
    └───────────────────────────┬─────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────┐
    │  SHORT-TERM STORAGE                                     │
    │                                                         │
    │  [ZOLPIDEM EVENING: anterograde amnesia window]        │
    │                                                         │
    │  11pm-2am: anything not already encoded may be lost    │
    └───────────────────────────┬─────────────────────────────┘
                                │
                           (overnight)
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────┐
    │  CONSOLIDATION (Sleep)                                  │
    │                                                         │
    │  [ZOLPIDEM: +30% SWS, +20% spindles] ← actually helps  │
    │  [SERTRALINE + ZOLPIDEM: -40% REM] ← hurts emotional   │
    │                                                         │
    │  Facts consolidate OK. Emotional memories suffer.       │
    └───────────────────────────┬─────────────────────────────┘
                                │
                                ▼
    ┌─────────────────────────────────────────────────────────┐
    │  LONG-TERM STORAGE                                      │
    │                                                         │
    │  What makes it here is stable.                         │
    │  But less makes it here, especially emotional stuff.   │
    └─────────────────────────────────────────────────────────┘
```

---

## Part 6: How Mnemosyne Compensates

Now the good part. For each impairment, a compensation strategy.

### Compensation 1: External Emotional Tagging

**Problem:** Sertraline reduces amygdala-based emotional encoding by ~40%

**Solution:** Use external signals to detect salience

```
INTERNAL (impaired)              EXTERNAL (Mnemosyne)
───────────────────              ────────────────────
Amygdala activation       →      Heart rate spike (pendant)
"This feels important"    →      Keyword detection: "deadline",
                                 "important", "fuck", "remember"
Emotional arousal         →      Sentiment analysis of speech
Gut feeling               →      Context anomaly detection
```

**Modified salience calculation:**

```
S_internal = E × δ_sertraline  (impaired, ~60% of normal)

S_external = f(HR_spike, keywords, sentiment, anomaly)

S_combined = max(S_internal, S_external)  // external can rescue

OR weighted:
S_combined = 0.4·S_internal + 0.6·S_external  // trust external more
```

**Example:**

```
You're in a meeting. Someone says "the deadline is Friday."
Your amygdala: meh (sertraline blunting)
Your heart rate: slight spike
Keyword detector: "deadline" flagged
Mnemosyne: boosts salience to 0.7 despite low internal signal
```

### Compensation 2: Aggressive Capture During Danger Windows

**Problem:** Zolpidem creates amnesia window 11pm-2am

**Solution:**
1. Warn about the window
2. Capture more aggressively in danger zone
3. Surface consolidated version next morning

```
CAPTURE STRATEGY BY TIME:

Time          Capture Rate    Note
───────────────────────────────────────────────
6am-9am       Every 90s       Residual effects, capture more
9am-6pm       Every 120s      Best window, normal rate
6pm-10pm      Every 90s       Evening meds kicking in
10pm-11pm     Every 60s       Pre-zolpidem, important window
11pm-2am      Every 30s       DANGER ZONE: capture everything
                              Auto-flag as "low-confidence encoding"
2am-6am       Sleep           Consolidation happening
```

**Morning reconstruction prompt:**

```
"Last night after 10pm, you:
 - Had a conversation about [X] (captured at 10:23pm)
 - Looked at [Y] file (captured at 10:45pm)
 - Took zolpidem at 11pm
 - [Low confidence] May have thought about [Z]

 Do any of these need attention?"
```

### Compensation 3: External Consolidation

**Problem:** REM suppression impairs emotional memory consolidation

**Solution:** Run consolidation daemon that simulates what sleep should do

```python
def mnemosyne_consolidation():
    """
    Run nightly to compensate for impaired sleep consolidation.
    """
    # 1. Find memories from yesterday that didn't consolidate well
    yesterdays_memories = get_memories(timerange="yesterday")

    for memory in yesterdays_memories:
        # Check if it's emotionally tagged but low-strength
        if memory.emotional_valence > 0.5 and memory.strength < 0.4:
            # This should have consolidated but didn't (REM failure)
            # Artificially boost it
            memory.strength += 0.2
            memory.reactivations.append(("artificial_consolidation", 0.2))

    # 2. Cluster similar memories and merge
    weak_memories = [m for m in yesterdays_memories if m.strength < 0.3]
    clusters = cluster_by_embedding(weak_memories, threshold=0.7)

    for cluster in clusters:
        if len(cluster) >= 3:
            merged = llm_generate_summary(cluster)
            store(merged, strength=max(m.strength for m in cluster) + 0.1)
            archive(cluster)

    # 3. Surface high-forgetting-risk memories for morning review
    at_risk = [m for m in get_all_memories() if forgetting_risk(m) > 0.7]
    queue_for_morning_surface(at_risk[:5])
```

### Compensation 4: More Frequent Reactivation

**Problem:** Faster decay (λ +15%) due to LTP inhibition

**Solution:** More frequent memory surfacing = more reactivation boosts

```
BASELINE (healthy person):
  Needs reactivation every ~3-4 days to maintain memory

ON YOUR MEDS:
  Needs reactivation every ~2-3 days

MNEMOSYNE STRATEGY:
  - Track last-accessed time for all memories
  - Proactively surface memories approaching decay threshold
  - Spaced repetition with tighter intervals
```

**Modified decay with Mnemosyne intervention:**

```
WITHOUT MNEMOSYNE:
M(t) = M₀ × e^(-0.115t)  (faster decay)

Day 1: 100% → 65%
Day 2: 65%  → 42%
Day 3: 42%  → 27%
Day 7: 27%  → 5%   ← effectively lost

WITH MNEMOSYNE (reactivates on day 2):
Day 1: 100% → 65%
Day 2: 65%  → 42% → REACTIVATE → 62%
Day 3: 62%  → 47%
Day 4: 47%  → 35% → REACTIVATE → 55%
Day 7: 55%  → 28%  ← still accessible
```

### Compensation 5: Rich Context Binding

**Problem:** Emotional binding is impaired (can't "feel" connections)

**Solution:** Bind with explicit context instead

```
NORMAL BINDING:
  Memory + Emotion = Strong association
  "I remember that meeting because I was stressed"

ON MEDS (impaired):
  Memory + (weak emotion) = Weak association
  "I remember there was... a meeting?"

MNEMOSYNE BINDING:
  Memory + Time + Location + People + Activity + Preceding_event
  "Meeting at 2pm, office 3rd floor, with Ravi, after standup"

  Context becomes the binding, not emotion
```

**Retrieval changes:**

```
NORMAL RETRIEVAL:
  Query: "deadline"
  Brain: searches by emotional tag + semantic
  Result: "that stressful meeting"

MNEMOSYNE RETRIEVAL:
  Query: "deadline"
  System: searches by semantic + time + context
  Result: "Tuesday 2:30pm meeting with Ravi in office"

  More specific, less emotional, but MORE RELIABLE
```

---

## Part 7: The Complete Modified System

Here's the full picture:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MNEMOSYNE + YOUR BRAIN                           │
│                    (Compensated Memory System)                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  CAPTURE (Aggressive)                                               │
│  ├── Screen: every 60-120s (30s in danger window)                  │
│  ├── Audio: continuous ambient (pendant)                           │
│  ├── Context: window, git, location, time                          │
│  └── Physiology: heart rate (external emotional signal)            │
│                                                                     │
│  ENCODING (Compensated)                                             │
│  ├── Internal: M₀_med = 0.28A + 0.20E + 0.20N + 0.15ΔP            │
│  ├── External: S_ext = f(HR, keywords, sentiment, anomaly)         │
│  └── Combined: M₀_eff = M₀_med × (1 + boost_from_S_ext)            │
│                                                                     │
│  STORAGE                                                            │
│  ├── Working: last 10 minutes, full detail                         │
│  ├── Short-term: hours, embedded, decaying at λ=0.115/hr           │
│  ├── Danger-zone: 11pm-2am captures flagged, extra redundancy      │
│  └── Long-term: consolidated by daemon, λ=0.012/hr                 │
│                                                                     │
│  CONSOLIDATION (External Daemon)                                    │
│  ├── Runs nightly (2am-5am)                                        │
│  ├── Boosts emotional memories that should have consolidated       │
│  ├── Merges weak similar memories into summaries                   │
│  ├── Queues at-risk memories for morning surfacing                 │
│  └── Compensates for REM suppression                               │
│                                                                     │
│  RETRIEVAL (Context-Heavy)                                          │
│  ├── Semantic similarity (embedding distance)                      │
│  ├── Context similarity (time, location, activity, people)         │
│  ├── Recency weighting (decay-adjusted)                            │
│  ├── Competition modeling (interference)                           │
│  └── Reconstruction if sparse (LLM fills gaps)                     │
│                                                                     │
│  PROACTIVE SURFACING                                                │
│  ├── Morning review of overnight captures                          │
│  ├── Context-switch reminders ("before the meeting, you...")       │
│  ├── Forgetting-risk alerts (approaching decay threshold)          │
│  └── Spaced reactivation (tighter intervals than healthy brain)    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Part 8: Real-Life Impact (Coding, Learning, Math, Logic, Relationships)

Okay, enough theory. How does this actually fuck with my daily life?

---

### Impact on Coding

Coding uses several types of memory:

```
MEMORY TYPES IN PROGRAMMING

┌─────────────────────────────────────────────────────────────────┐
│  TYPE              │  WHAT IT'S FOR           │  MEDICATION HIT │
├────────────────────┼──────────────────────────┼─────────────────┤
│  Working memory    │  Hold variables, flow    │  -8% (minor)    │
│                    │  in head while coding    │                 │
├────────────────────┼──────────────────────────┼─────────────────┤
│  Episodic memory   │  "What was I doing       │  -50% emotional │
│                    │  before the meeting?"    │  -2% neutral    │
├────────────────────┼──────────────────────────┼─────────────────┤
│  Semantic memory   │  Syntax, APIs, patterns  │  -5% (okay)     │
│                    │  you've learned          │                 │
├────────────────────┼──────────────────────────┼─────────────────┤
│  Procedural memory │  How to type, IDE        │  Unaffected     │
│                    │  shortcuts, muscle mem   │                 │
└─────────────────────────────────────────────────────────────────┘
```

**The actual problems:**

1. **Context loss on interruption** — This is the big one
   ```
   You're debugging something complex
   Slack notification → 15 min meeting
   Come back: "...what was I doing?"

   The EMOTIONAL tag that would normally flag
   "this is important, remember where you are"
   is dampened by sertraline.

   Without the "this matters" feeling, your brain
   doesn't prioritize storing the context.
   ```

2. **Forgetting what you've already tried**
   ```
   Debugging a bug for 2 hours
   Try approach A, B, C, D
   Get distracted
   Come back: "Did I already try changing the timeout?"

   Each failed attempt should create a mild frustration
   memory. Sertraline blunts that. You forget what didn't work.
   ```

3. **Losing the "aha" moments**
   ```
   Finally figure out the bug at 10pm
   "Oh! It's a race condition in the queue handler"
   Take zolpidem at 11pm

   Next morning: "I fixed something... what was the insight?"

   The zolpidem amnesia window ate your eureka moment.
   ```

4. **Morning fog affecting code quality**
   ```
   Zolpidem residual effects: -10% attention until ~9am
   Code written at 7am: more bugs, missed edge cases

   Best coding window: 9am-6pm (tofisopam active, others stable)
   ```

**How the system helps with coding:**

```
DAEMON COMPENSATION FOR CODING

┌─────────────────────────────────────────────────────────────────┐
│  PROBLEM                    │  DAEMON SOLUTION                  │
├─────────────────────────────┼───────────────────────────────────┤
│  Context loss on            │  Proactive context surfacing      │
│  interruption               │  "Before the meeting, you were    │
│                             │  at worker/retry.go:142 testing   │
│                             │  exponential backoff"             │
├─────────────────────────────┼───────────────────────────────────┤
│  Forgetting what you tried  │  Git state monitoring             │
│                             │  Track uncommitted changes        │
│                             │  "You've tried: timeout=30s,      │
│                             │  timeout=60s, removing retry"     │
├─────────────────────────────┼───────────────────────────────────┤
│  Losing insights            │  Aggressive capture 10pm-11pm     │
│  to zolpidem window         │  Morning reconstruction:          │
│                             │  "Last night you discovered       │
│                             │  the race condition was in..."    │
├─────────────────────────────┼───────────────────────────────────┤
│  Morning fog                │  Hearth displays:                 │
│                             │  "Cognitive capacity: 80%"        │
│                             │  "Suggested: lighter tasks        │
│                             │  until 9am"                       │
└─────────────────────────────────────────────────────────────────┘
```

**The Hearth for coding:**

```
┌─────────────────────────────────────────┐
│  THE HEARTH (desk display)              │
│                                         │
│  Morning 7:30am:                        │
│  ┌─────────────────────────────────┐    │
│  │  ☀️ Good morning                │    │
│  │                                 │    │
│  │  Last night you were working on │    │
│  │  the webhook retry bug.         │    │
│  │                                 │    │
│  │  Insight captured at 10:47pm:   │    │
│  │  "Race condition in queue -     │    │
│  │   the worker picks up job       │    │
│  │   before status update commits" │    │
│  │                                 │    │
│  │  Focus capacity: ████████░░ 82% │    │
│  │  Best window starts: ~9am       │    │
│  └─────────────────────────────────┘    │
│                                         │
│  E-ink display — glanceable, ambient    │
│  LED ring: soft pulse if something      │
│  needs your attention                   │
│                                         │
└─────────────────────────────────────────┘
```

---

### Impact on Learning

Learning new things requires:
1. **Encoding** — getting it into short-term memory
2. **Consolidation** — sleep-dependent transfer to long-term
3. **Retrieval** — being able to access it later
4. **Spaced repetition** — reactivation to prevent decay

**Where meds hurt learning:**

```
LEARNING A NEW FRAMEWORK (e.g., learning Rust)

NORMAL BRAIN:
  Day 1: Learn ownership concepts (emotional: "this is cool/hard!")
         Strong encoding due to novelty + emotion
         Night: REM consolidates, ties to existing knowledge
  Day 2: Recall is good, build on it
  Day 7: Still remember core concepts

ON YOUR MEDS:
  Day 1: Learn ownership concepts (emotional: "meh, it's fine")
         Weaker encoding (sertraline blunts the "cool/hard" feeling)
         Night: Less REM → emotional aspects don't consolidate well
  Day 2: "Wait, what's the borrow checker again?"
  Day 7: "I read something about ownership... what was it?"
```

**The specific hit:**

```
LEARNING EFFICIENCY BY TYPE

                              BASELINE    ON MEDS     IMPACT
                              ────────    ───────     ──────
Exciting new concept           0.8         0.45       -44%
(high novelty, high emotion)

Dry documentation              0.5         0.48       -4%
(low emotion, just facts)

Learning by doing              0.7         0.60       -14%
(procedural + declarative)

Connecting concepts            0.75        0.40       -47%
(relies on emotional "aha!")

THE IRONY:
  Boring stuff → barely affected
  Exciting stuff → heavily impaired

  Your brain now optimizes for mediocrity.
```

**How the system helps with learning:**

```
LEARNING COMPENSATION STRATEGIES

┌─────────────────────────────────────────────────────────────────┐
│  1. EXTERNAL EMOTIONAL TAGGING                                  │
│                                                                 │
│     Learning Rust ownership:                                    │
│     - Daemon detects topic = "new concept"                     │
│     - Shard detects elevated heart rate (engaged)              │
│     - System flags as "learning moment" → high salience        │
│     - Even if YOU don't feel excited, system marks it          │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  2. ACCELERATED SPACED REPETITION                               │
│                                                                 │
│     Normal schedule: Day 1, Day 3, Day 7, Day 14               │
│     Your schedule:   Day 1, Day 2, Day 4, Day 7, Day 10        │
│                                                                 │
│     Daemon surfaces: "Yesterday you learned about borrowing.   │
│     Key insight: references can't outlive their referent."     │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  3. CONNECTION ASSISTANCE                                       │
│                                                                 │
│     When learning new concept:                                  │
│     Daemon searches existing memories for related patterns      │
│     Surfaces: "This is similar to RAII in C++ which you        │
│     learned about in [memory from 2 weeks ago]"                │
│                                                                 │
│     Helps build the connections your REM-deprived sleep isn't  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  4. OPTIMAL LEARNING WINDOWS                                    │
│                                                                 │
│     Hearth suggests: "Your encoding capacity is highest        │
│     10am-12pm. Consider learning new material now."            │
│                                                                 │
│     Avoid learning new things after 9pm (approaching           │
│     zolpidem window, won't consolidate well)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

### Impact on Math

Good news: math is less affected than emotional/episodic memory.

**Why math is relatively spared:**

```
MATH USES DIFFERENT SYSTEMS

1. Working memory — holding intermediate results
   Impact: -8% (minor)

2. Procedural memory — how to do operations
   Impact: unaffected

3. Pattern recognition — seeing structure
   Impact: -5% (minor)

4. Semantic memory — formulas, theorems
   Impact: -5% (minor)

Math doesn't rely heavily on emotional tagging.
The "aha!" moment of understanding a proof might
be dampened, but the mechanical execution is fine.
```

**Where it DOES hurt:**

```
MATHEMATICAL INTUITION

The "feel" for whether an answer is right:
  Normal: "That doesn't feel right, let me check"
  On meds: "Is 47 right? I guess?"

This is a subtle emotional signal — your gut
telling you something's off. Sertraline dampens it.

REMEMBERING WHY A PROOF WORKS:
  Normal: "I remember this proof because it was
          so elegant, the trick was beautiful"
  On meds: "There was a proof... I remember steps
          1, 2, 3, but why does it work?"

The emotional "elegance" tag that makes proofs
memorable is weakened.
```

**Compensation:**

```
FOR MATH:

1. External "sanity check" — calculator verification
   for anything important, don't trust intuition

2. Document the "why" explicitly — write down why
   a proof works, what the insight is, because you
   won't remember the "aha" feeling

3. More worked examples — procedural memory is
   unaffected, so practice compensates
```

---

### Impact on Logic

Similar to math — procedural/formal logic is relatively spared.

```
LOGIC BREAKDOWN

                              IMPACT
                              ──────
Formal deduction               -5%
(if A then B, A, therefore B)

Holding premises in mind       -8%
(working memory)

Intuitive reasoning            -30%
("something feels wrong here")

Remembering why an argument    -40%
was compelling
```

**The problem:**

```
REMEMBERING ARGUMENTS

Normal conversation:
  Friend: "We should use Postgres because..."
  You: *remembers the argument because it was compelling*

On meds:
  Friend: "We should use Postgres because..."
  You: "We decided Postgres... why again?"

The emotional "this is a good argument" tag is
what makes you remember WHY, not just WHAT.
```

**Compensation:**

```
DAEMON CAPTURES DECISIONS

When it detects decision-making conversation:
- Flags as high salience
- Extracts: decision + reasoning
- Stores explicit "why" alongside "what"

Later query: "Why did we choose Postgres?"
System: "On Jan 15th, Ravi said CockroachDB's
licensing made you nervous. Explicit quote:
'Let's stick with what we know.'"

The external system remembers the WHY.
```

---

### Impact on Relationships

This is where it hurts most. Relationships run on emotional memory.

```
RELATIONSHIP MEMORY

What relationships need:
├── Remember conversations (episodic)
├── Remember what matters to people (semantic + emotional)
├── Remember commitments/promises (episodic + emotional tag)
├── Remember shared experiences (episodic + emotional)
└── Emotional attunement (real-time emotional processing)

All of these are impacted by sertraline + zolpidem.
```

**The specific damage:**

```
SCENARIO: Partner tells you something important at 10pm

Normal brain:
  Input: "I'm worried about the project deadline"
  Processing: *empathy, emotional resonance*
  Encoding: high (emotional salience)
  Next day: remember conversation, remember feeling

On your meds:
  Input: "I'm worried about the project deadline"
  Processing: *dampened empathy, muted response*
  Encoding: medium (sertraline blunting)
  Zolpidem taken at 11pm: consolidation compromised
  Next day: "We talked about... something?"

  Partner: "I told you I was stressed!"
  You: "...did you?"

THIS HURTS RELATIONSHIPS.
```

**The forgetting patterns that damage relationships:**

```
┌─────────────────────────────────────────────────────────────────┐
│  PATTERN                      │  RELATIONSHIP IMPACT            │
├───────────────────────────────┼─────────────────────────────────┤
│  Forgetting conversations     │  Partner feels unheard          │
│                               │  "You never listen"             │
├───────────────────────────────┼─────────────────────────────────┤
│  Forgetting commitments       │  Breaking promises              │
│  (especially evening ones)    │  "You said you would..."        │
├───────────────────────────────┼─────────────────────────────────┤
│  Emotional blunting           │  Seeming distant, uncaring      │
│                               │  "You don't seem to care"       │
├───────────────────────────────┼─────────────────────────────────┤
│  Forgetting why you were      │  Unresolved conflicts           │
│  upset or why they were       │  Arguments that never close     │
├───────────────────────────────┼─────────────────────────────────┤
│  Forgetting special moments   │  Diminished shared history      │
│                               │  "Remember when we..."  "No?"   │
└─────────────────────────────────────────────────────────────────┘
```

**How the Shard helps with relationships:**

```
THE SHARD (pendant) FOR RELATIONSHIPS

When worn during conversations:
├── Captures audio snippets (with consent)
├── Detects elevated heart rate (emotional moment)
├── Flags conversations with people you care about
└── Creates richer memory entries for relationships

SCENARIO: Conversation with partner at 9pm

Shard captures:
├── Topic: partner worried about job
├── Emotional markers: concern in voice, your HR elevated
├── Key quotes: "I might not get the promotion"
├── Your response: [captured]
└── Commitment made: "I'll help you prep for the interview"

Next morning, Hearth shows:
┌─────────────────────────────────────────┐
│                                         │
│  Last night with [Partner]:             │
│                                         │
│  They're worried about the promotion.   │
│  You said you'd help them prep.         │
│                                         │
│  Suggested action: follow up today      │
│                                         │
└─────────────────────────────────────────┘

The system remembers what you might forget.
The system reminds you to follow through.
```

**Commitment tracking:**

```
PROMISE DETECTION

Shard + Daemon detect commitment language:
- "I'll do X"
- "I promise to..."
- "Let me handle that"
- "I won't forget"

Creates commitment entry:
├── What was promised
├── To whom
├── When
├── Context/why

Surfacing:
├── Reminder before deadline
├── Morning review of open commitments
└── Prompt to confirm completion

YOU CAN'T TRUST YOUR BRAIN TO REMEMBER PROMISES.
The system becomes your external promise-keeper.
```

**Relationship memory over time:**

```
BUILDING SHARED HISTORY

The system captures moments that build relationships:
├── First times ("first time we went to X")
├── Celebrations
├── Difficult conversations that got resolved
├── Inside jokes (detected by laughter + repeated phrases)
└── What they told you matters to them

Over time:
  Query: "What does [Partner] care about?"
  System: "Based on conversations:
           - Career growth (mentioned 23 times)
           - Time with family (15 times)
           - Your health (12 times)
           - The garden project (8 times)"

  This is knowledge your brain should hold but doesn't.
```

---

### Summary: Real-Life Impact

```
┌─────────────────────────────────────────────────────────────────┐
│  DOMAIN           │  IMPACT  │  MAIN PROBLEM                   │
├───────────────────┼──────────┼─────────────────────────────────┤
│  Coding           │  Medium  │  Context loss on interruption   │
│                   │          │  Forgetting insights to zolpidem│
├───────────────────┼──────────┼─────────────────────────────────┤
│  Learning         │  High    │  New concepts don't stick       │
│                   │          │  "Aha!" moments weakened        │
├───────────────────┼──────────┼─────────────────────────────────┤
│  Math             │  Low     │  Intuition dampened             │
│                   │          │  Procedural mostly fine         │
├───────────────────┼──────────┼─────────────────────────────────┤
│  Logic            │  Low-Med │  Formal fine, intuitive weaker  │
│                   │          │  Forgetting "why" of arguments  │
├───────────────────┼──────────┼─────────────────────────────────┤
│  Relationships    │  HIGH    │  Forgetting conversations       │
│                   │          │  Missing commitments            │
│                   │          │  Emotional distance             │
│                   │          │  Eroding shared history         │
└─────────────────────────────────────────────────────────────────┘
```

```
HOW EACH DEVICE HELPS

┌─────────────────────────────────────────────────────────────────┐
│  DAEMON (laptop/background)                                     │
│  ├── Screen capture for work context                           │
│  ├── Git state for coding history                              │
│  ├── Consolidation overnight                                   │
│  ├── Morning surfacing of what matters                         │
│  └── Spaced repetition for learning                            │
│                                                                 │
│  SHARD (pendant)                                                │
│  ├── Captures conversations (relationships!)                   │
│  ├── Heart rate for emotional moments                          │
│  ├── Ambient context when away from laptop                     │
│  ├── Commitment detection                                      │
│  └── Fills the gap when you're not at a screen                 │
│                                                                 │
│  HEARTH (desk display)                                          │
│  ├── Morning briefing (what matters today)                     │
│  ├── Cognitive capacity indicator                              │
│  ├── Relationship reminders (follow up with X)                 │
│  ├── Commitment due dates                                      │
│  └── Ambient, glanceable — doesn't demand attention            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Summary: The Numbers

```
┌──────────────────────────────────────────────────────────────────┐
│  PARAMETER          │  BASELINE  │  ON MEDS   │  WITH MNEMOSYNE │
├─────────────────────┼────────────┼────────────┼─────────────────┤
│  Attention weight   │    0.30    │    0.28    │      0.28       │
│  Emotion weight     │    0.35    │    0.20    │  0.20 + ext     │
│  Decay rate (λ)     │    0.10    │    0.115   │   compensated   │
│  Consolidation eff  │    0.75    │    0.65*   │      0.70       │
│  Retrieval accuracy │    0.80    │    0.55    │      0.75       │
│  Forgetting @ day 7 │    30%     │    50%     │      35%        │
└──────────────────────────────────────────────────────────────────┘

* Emotional memories specifically; declarative may be slightly better

THE GOAL:
  Not to restore you to baseline (can't without changing meds)
  But to get from 55% → 75% through external compensation

That 20% improvement is the difference between
"I forgot we talked about that" and "right, Tuesday with Ravi"
```

---

## References

Research used for this model:

**Zolpidem + Memory:**
- [The effect of zolpidem on memory consolidation over a night of sleep](https://pmc.ncbi.nlm.nih.gov/articles/PMC8064806/) - PMC8064806
- [Zolpidem augments hippocampal-prefrontal coupling during NREM sleep](https://www.nature.com/articles/s41386-022-01355-9) - Nature
- [In the Zzz Zone: Effects of Z-Drugs on Human Performance](https://pmc.ncbi.nlm.nih.gov/articles/PMC3657033/) - PMC3657033
- [Acute Effects of Zolpidem Extended-Release](https://pmc.ncbi.nlm.nih.gov/articles/PMC3280925/) - PMC3280925

**Sertraline + Memory:**
- [Effects of serotonin in the hippocampus: SSRIs and pyramidal cell function](https://pmc.ncbi.nlm.nih.gov/articles/PMC4825106/) - PMC4825106
- [SSRI effects on memory in older adults](https://pmc.ncbi.nlm.nih.gov/articles/PMC9112622/) - PMC9112622
- [SSRI-Induced Emotional Blunting](https://academicworks.cuny.edu/cgi/viewcontent.cgi?article=6077&context=gc_etds) - CUNY

**Oxcarbazepine:**
- [The cognitive impact of antiepileptic drugs](https://pmc.ncbi.nlm.nih.gov/articles/PMC3229254/) - PMC3229254
- [Cognitive effects of lamotrigine vs oxcarbazepine](https://pmc.ncbi.nlm.nih.gov/articles/PMC2686935/) - PMC2686935

**Tofisopam:**
- [Antiamnesic effects of tofisopam](https://pubmed.ncbi.nlm.nih.gov/31981560/) - PubMed
- [Tofisopam overview](https://www.sciencedirect.com/topics/pharmacology-toxicology-and-pharmaceutical-science/tofisopam) - ScienceDirect

**Emotional Memory + Amygdala:**
- [Neuronal activity in amygdala and hippocampus enhances emotional memory encoding](https://www.nature.com/articles/s41562-022-01502-8) - Nature

---

*Notes from building a memory system that accounts for my specific medication stack. February 2025.*
