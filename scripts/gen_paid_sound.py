#!/usr/bin/env python3
"""Wave 10 / B4 — synthesizes the ONE application sound.

Article IV.3 (DESIGN_CONSTITUTION.md): "Sound is saffron. The application
budget is one sound, reserved for the operator's true win moment — a deal
closing as paid." Owner-ratified direction (2026-07-13, FABLE_WAVE10_SPEC
B4): synthesize, don't license — a small script producing a two-tone low
settle, warm attack, fast decay, <1s, <=50KB. No downloaded/licensed samples.

Character: a *settle*, not a fanfare — a soft wooden "tunk-tmm", not a
chime arpeggio. Two tones a soft fifth apart:
  - TONE_A ("tunk"): the fifth, brighter and shorter, hits first.
  - TONE_B ("tmm"): the root, lower and warmer, follows and carries the
    decay — this is the "settle."
Both are pure sine partials (no harsh harmonics) blended with a touch of a
soft second partial for warmth, each shaped by a short linear attack and an
exponential decay — no sustain, no loop, nothing that reads as a chime.

Re-roll guide for the owner (all named constants below):
  - ROOT_FREQ_HZ / FIFTH_RATIO: pitch + interval character (raise/lower the
    whole sound, or change the interval from a fifth to something else).
  - TONE_A_*/TONE_B_*: independently retime, re-pitch, or re-gain either
    half of the "tunk-tmm" (e.g. bigger gap = more of a distinct two-hit
    settle; smaller gap = a single soft thud).
  - *_DECAY_RATE: higher = snappier/shorter perceived tail, lower = longer
    warmer tail (bounded by TOTAL_DURATION_S regardless).
  - WARMTH_PARTIAL_RATIO / WARMTH_PARTIAL_GAIN: amount of the soft upper
    partial mixed under each tone (the "warm" in "warm attack, fast decay").
  - MASTER_GAIN: overall loudness headroom before the 16-bit clip.
  - SAMPLE_RATE_HZ: 22050 keeps the file tiny for a clip this short; raise
    to 44100 for a "less lo-fi" character (roughly doubles file size).

Output: frontend/src/assets/sounds/paid-settle.wav
Mono, 16-bit PCM, must stay <1s and <=50KB (hard constraints from spec B4).
"""

import math
import os
import struct

# ---------------------------------------------------------------------------
# RE-ROLL PARAMETERS — tweak these, then `python scripts/gen_paid_sound.py`
# ---------------------------------------------------------------------------

SAMPLE_RATE_HZ = 22050  # mono, 16-bit — keeps the file small for a <1s clip
TOTAL_DURATION_S = 0.42  # hard ceiling is 1.0s; this settle is well under it

# Pitch: a low root with a soft fifth above it (per spec: "low root
# ~200-320Hz plus a soft fifth").
ROOT_FREQ_HZ = 208.0  # TONE_B ("tmm") — the settle
FIFTH_RATIO = 1.5  # perfect fifth
FIFTH_FREQ_HZ = ROOT_FREQ_HZ * FIFTH_RATIO  # TONE_A ("tunk") = 312 Hz

# TONE_A — "tunk": hits immediately, brighter, short, decays fast.
TONE_A_FREQ_HZ = FIFTH_FREQ_HZ
TONE_A_START_S = 0.0
TONE_A_ATTACK_S = 0.008
TONE_A_DECAY_RATE = 20.0  # exponential decay constant (1/s), higher = snappier
TONE_A_GAIN = 0.55

# TONE_B — "tmm": follows with a small overlap, lower, warmer, carries the
# audible tail of the settle.
TONE_B_FREQ_HZ = ROOT_FREQ_HZ
TONE_B_START_S = 0.065
TONE_B_ATTACK_S = 0.018
TONE_B_DECAY_RATE = 8.5
TONE_B_GAIN = 0.55

# Warmth: a quiet soft upper partial blended under each tone so it reads as
# a wooden "settle" rather than a bare sine beep.
WARMTH_PARTIAL_RATIO = 2.0  # one octave above the tone's own frequency
WARMTH_PARTIAL_GAIN = 0.12  # relative to the tone's own gain

MASTER_GAIN = 0.85  # final headroom before 16-bit clipping

OUTPUT_PATH = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "frontend", "src", "assets", "sounds", "paid-settle.wav",
)

# ---------------------------------------------------------------------------
# Synthesis
# ---------------------------------------------------------------------------


def tone_sample(t: float, start_s: float, freq_hz: float, attack_s: float,
                 decay_rate: float, gain: float) -> float:
    """One shaped sine partial (+ a soft warmth partial) at time t."""
    local_t = t - start_s
    if local_t < 0:
        return 0.0

    # Linear attack, then exponential decay (no sustain/loop).
    if local_t < attack_s:
        envelope = local_t / attack_s
    else:
        envelope = math.exp(-decay_rate * (local_t - attack_s))

    fundamental = math.sin(2 * math.pi * freq_hz * local_t)
    warmth = WARMTH_PARTIAL_GAIN * math.sin(
        2 * math.pi * freq_hz * WARMTH_PARTIAL_RATIO * local_t
    )
    return gain * envelope * (fundamental + warmth)


def synthesize() -> bytes:
    num_samples = int(SAMPLE_RATE_HZ * TOTAL_DURATION_S)
    samples = []
    peak = 0.0
    for n in range(num_samples):
        t = n / SAMPLE_RATE_HZ
        value = tone_sample(
            t, TONE_A_START_S, TONE_A_FREQ_HZ, TONE_A_ATTACK_S,
            TONE_A_DECAY_RATE, TONE_A_GAIN,
        ) + tone_sample(
            t, TONE_B_START_S, TONE_B_FREQ_HZ, TONE_B_ATTACK_S,
            TONE_B_DECAY_RATE, TONE_B_GAIN,
        )
        samples.append(value)
        peak = max(peak, abs(value))

    # Normalize to MASTER_GAIN headroom, then quantize to 16-bit PCM.
    norm = (MASTER_GAIN / peak) if peak > 0 else 1.0
    pcm = bytearray()
    for value in samples:
        v = max(-1.0, min(1.0, value * norm))
        pcm += struct.pack("<h", int(v * 32767))
    return bytes(pcm)


def write_wav(pcm_data: bytes, path: str) -> None:
    num_channels = 1
    bits_per_sample = 16
    byte_rate = SAMPLE_RATE_HZ * num_channels * bits_per_sample // 8
    block_align = num_channels * bits_per_sample // 8
    data_size = len(pcm_data)

    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(b"RIFF")
        f.write(struct.pack("<I", 36 + data_size))
        f.write(b"WAVE")
        f.write(b"fmt ")
        f.write(struct.pack("<I", 16))  # PCM fmt chunk size
        f.write(struct.pack("<H", 1))  # PCM format tag
        f.write(struct.pack("<H", num_channels))
        f.write(struct.pack("<I", SAMPLE_RATE_HZ))
        f.write(struct.pack("<I", byte_rate))
        f.write(struct.pack("<H", block_align))
        f.write(struct.pack("<H", bits_per_sample))
        f.write(b"data")
        f.write(struct.pack("<I", data_size))
        f.write(pcm_data)


def main() -> None:
    pcm = synthesize()
    write_wav(pcm, OUTPUT_PATH)
    size_bytes = os.path.getsize(OUTPUT_PATH)
    duration_s = len(pcm) / 2 / SAMPLE_RATE_HZ
    print(f"Wrote {OUTPUT_PATH}")
    print(f"  size: {size_bytes} bytes ({size_bytes / 1024:.2f} KB)")
    print(f"  duration: {duration_s:.3f} s at {SAMPLE_RATE_HZ} Hz")
    assert size_bytes <= 50 * 1024, "asset exceeds 50KB budget"
    assert duration_s < 1.0, "asset exceeds 1s budget"


if __name__ == "__main__":
    main()
