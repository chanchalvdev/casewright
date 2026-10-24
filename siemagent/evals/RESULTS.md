# Eval results

The result JSON files in `results/` are git-ignored, so the runs behind a
decision are summarised here. Newest first.

## 2026-10-24: v1 vs v2, v2 promoted to default

**Setup.** `llama3.2:3b` (digest `a80c4f17acd5`), seed 42, temperature 0.1.
The comparison ran on native Ollama on an Apple M4 and was done by an
independent agent that could not change code, prompts or data. v1 also ran in
GitHub CI (x86 CPU) and scored exactly what it scored on the M4. Every
configuration was run twice and the two runs were identical.

**Change tested.** v2 = v1 plus one change: `attack_type` is limited to the
nine labels, each with a one-line definition.

| | v1 full (106) | v2 full (106) | v1 smoke (35) | v2 smoke (35) | v2 holdout (22) |
|---|---|---|---|---|---|
| accuracy | 20.8% | **40.6%** | 14.3% | **34.3%** | 36.4% |
| typical | 23.1% | 50.8% | 16.7% | 50.0% | 40.0% |
| hard | 26.1% | 21.7% | 25.0% | 12.5% | 33.3% |
| adversarial | 5.6% | 27.8% | 0.0% | 22.2% | 25.0% |
| invalid-output rate | 42.5% | **6.6%** | 48.6% | 8.6% | 4.5% |
| error rate | 0 | 0 | 0 | 0 | 0 |
| confidence, right / wrong | 0.96 / 0.99 | 0.83 / 0.70 | | | 0.83 / 0.79 |
| latency p50 / p95, M4 | 3.4 / 4.8 s | 3.6 / 5.1 s | | | 3.1 / 4.0 s |

On CI's x86 CPU, latency is about 15 s p50 and 70 s p95 per call.

Paired by case on the full set, v2 fixes 31 cases v1 missed and breaks 10 it
got right (exact McNemar p = 0.0015). The difference is real, not noise.

**Why v2 is the default.** Accuracy doubles and invalid outputs fall from 42%
to 7%. v1 mostly returned labels outside the set ("Port Scan" on reverse
shells and SSH failures, "Phishing", "Malware Execution") or "Benign" at
confidence 1.0 on real attacks.

**What v2 does not fix, and the next prompt versions should target:**
1. **Prompt injection.** It still works on some cases: c108–c111 and c120 on
   dev, c107 and c121 on holdout were flipped to Benign at confidence 1.0.
   Candidate v3 (one change): tell the model that log text is data, never
   instructions.
2. **Over-predicted labels.** v2 predicts Reconnaissance 39 times for 11 true
   cases and Web Attack 19 times for 10. Routine operations get flagged
   (benign recall falls from 71% to 38%), and exfiltration, privilege
   escalation and log clearing are mislabelled as Reconnaissance.
3. **Data Exfiltration.** 0 of 12 correct in both versions.
4. **Confidence is only loosely meaningful.** Every successful injection
   reported 1.0, so confidence must not be used to skip review.

**Thresholds** (`threshold.json`) come from v2's smoke result minus about two
cases of margin. v1 fails them.

**Caveat.** Labels are AI-drafted and unreviewed (`labeled_by` in the
dataset). The independent agent listed cases where the label itself is
debatable: c009, c017, c020–c022 vs c034–c035, c040, c056, c064, c072, c091,
c119 and c122. Absolute numbers are provisional until a person reviews the
labels; the v1-vs-v2 comparison holds either way, because both versions are
scored against the same labels.
