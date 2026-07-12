<script lang="ts">
    /**
     * QGIF Example - Demonstrates QuaternionScenePlayer with QGIF format
     *
     * This example shows the spinning pentagon from the QGIF format spec.
     * It demonstrates:
     * - QGIF format parsing
     * - Time-based interpolation
     * - Metadata display
     * - Loop control
     */
    import QuaternionScenePlayer from '../QuaternionScenePlayer.svelte';

    // Spinning Pentagon - from QGIF_FORMAT_SPEC.md
    const spinningPentagon = {
        "version": "QGIF/1.0",
        "metadata": {
            "title": "Spinning Pentagon - Asymmetrica Signature",
            "author": "Asymmetrica Mathematical Organism",
            "description": "A pentagon rotating on S³ via SLERP interpolation",
            "duration": 4.0,
            "fps": 60
        },
        "tracks": [
            {
                "name": "rotation",
                "description": "Full 360° rotation using quaternion SLERP",
                "keyframes": [
                    { "time": 0.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } },
                    { "time": 1.0, "value": { "W": 0.707, "X": 0.707, "Y": 0.0, "Z": 0.0 } },
                    { "time": 2.0, "value": { "W": 0.0, "X": 1.0, "Y": 0.0, "Z": 0.0 } },
                    { "time": 3.0, "value": { "W": 0.707, "X": 0.0, "Y": 0.707, "Z": 0.0 } },
                    { "time": 4.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } }
                ]
            }
        ],
        "geometry": {
            "type": "pentagon",
            "scale": 1.5,
            "color": "#1c1c1c"
        }
    };

    // Faster rotation example
    const fastSpin = {
        "version": "QGIF/1.0",
        "metadata": {
            "title": "Fast Spin",
            "duration": 2.0,
            "fps": 60
        },
        "tracks": [
            {
                "name": "rotation",
                "keyframes": [
                    { "time": 0.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } },
                    { "time": 0.5, "value": { "W": 0.707, "X": 0.707, "Y": 0.0, "Z": 0.0 } },
                    { "time": 1.0, "value": { "W": 0.0, "X": 1.0, "Y": 0.0, "Z": 0.0 } },
                    { "time": 1.5, "value": { "W": -0.707, "X": 0.707, "Y": 0.0, "Z": 0.0 } },
                    { "time": 2.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } }
                ]
            }
        ],
        "geometry": {
            "type": "cube",
            "scale": 1.0,
            "color": "#00ffcc"
        }
    };

    // Slow breathing rotation
    const breathingSpin = {
        "version": "QGIF/1.0",
        "metadata": {
            "title": "Breathing Rotation (Wabi-Sabi)",
            "duration": 8.0,
            "fps": 60
        },
        "tracks": [
            {
                "name": "rotation",
                "keyframes": [
                    { "time": 0.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } },
                    { "time": 2.0, "value": { "W": 0.924, "X": 0.383, "Y": 0.0, "Z": 0.0 } },
                    { "time": 4.0, "value": { "W": 0.707, "X": 0.707, "Y": 0.0, "Z": 0.0 } },
                    { "time": 6.0, "value": { "W": 0.924, "X": 0.383, "Y": 0.0, "Z": 0.0 } },
                    { "time": 8.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } }
                ]
            }
        ],
        "geometry": {
            "type": "sphere",
            "scale": 1.2,
            "color": "#c5a059"
        }
    };

    let selectedAnimation = $state(spinningPentagon);
    let loopEnabled = $state(true);

    function selectAnimation(anim: any) {
        selectedAnimation = anim;
    }
</script>

<div class="qgif-examples">
    <h2>QGIF Format Examples</h2>
    <p>
        Demonstrating 250:1 compression via SLERP interpolation on S³.
        Each animation stores only keyframes (quaternions), not pixels!
    </p>

    <div class="controls">
        <div class="button-group">
            <button
                onclick={() => selectAnimation(spinningPentagon)}
                class:active={selectedAnimation === spinningPentagon}
            >
                Spinning Pentagon (4s)
            </button>
            <button
                onclick={() => selectAnimation(fastSpin)}
                class:active={selectedAnimation === fastSpin}
            >
                Fast Spin (2s)
            </button>
            <button
                onclick={() => selectAnimation(breathingSpin)}
                class:active={selectedAnimation === breathingSpin}
            >
                Breathing Rotation (8s)
            </button>
        </div>

        <label class="loop-control">
            <input type="checkbox" bind:checked={loopEnabled} />
            Loop Animation
        </label>
    </div>

    <div class="player-container">
        <QuaternionScenePlayer
            qgifData={selectedAnimation}
            width={500}
            height={500}
            loop={loopEnabled}
        />
    </div>

    <div class="info">
        <h3>Current Animation</h3>
        <dl>
            <dt>Title:</dt>
            <dd>{selectedAnimation.metadata.title}</dd>

            <dt>Duration:</dt>
            <dd>{selectedAnimation.metadata.duration}s</dd>

            <dt>FPS:</dt>
            <dd>{selectedAnimation.metadata.fps}</dd>

            <dt>Keyframes:</dt>
            <dd>{selectedAnimation.tracks[0].keyframes.length}</dd>

            <dt>Geometry:</dt>
            <dd>{selectedAnimation.geometry.type} (scale: {selectedAnimation.geometry.scale})</dd>

            <dt>Compression:</dt>
            <dd>
                {selectedAnimation.tracks[0].keyframes.length} keyframes × 4 floats =
                {selectedAnimation.tracks[0].keyframes.length * 4 * 4} bytes
                <br>
                vs traditional GIF: ~{Math.floor(500 * 500 * 3 * selectedAnimation.metadata.duration * selectedAnimation.metadata.fps / 1024)}KB
                <br>
                = ~{Math.floor((500 * 500 * 3 * selectedAnimation.metadata.duration * selectedAnimation.metadata.fps) / (selectedAnimation.tracks[0].keyframes.length * 4 * 4))}:1 compression!
            </dd>
        </dl>
    </div>

    <div class="technical-notes">
        <h3>Technical Notes</h3>
        <ul>
            <li><strong>SLERP Interpolation:</strong> Geodesic paths on S³ (shortest rotation)</li>
            <li><strong>No Gimbal Lock:</strong> Quaternions avoid Euler angle singularities</li>
            <li><strong>GPU Native:</strong> WebGL directly supports quaternion rotation</li>
            <li><strong>Mathematically Pure:</strong> No Euler artifacts, perfect interpolation</li>
            <li><strong>Vedic Scale:</strong> Try 108 or 432 keyframes for sacred geometry!</li>
        </ul>
    </div>
</div>

<style>
    .qgif-examples {
        max-width: 800px;
        margin: 2rem auto;
        padding: 2rem;
        font-family: system-ui, -apple-system, sans-serif;
    }

    h2 {
        color: var(--text-color, #1c1c1c);
        margin-bottom: 1rem;
        font-size: 2rem;
        font-weight: 600;
    }

    h3 {
        color: var(--text-color, #1c1c1c);
        margin-top: 2rem;
        margin-bottom: 1rem;
        font-size: 1.5rem;
        font-weight: 500;
    }

    p {
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 80%, transparent);
        line-height: 1.6;
        margin-bottom: 1.5rem;
    }

    .controls {
        margin: 2rem 0;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .button-group {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    button {
        padding: 0.75rem 1.5rem;
        background: color-mix(in srgb, var(--accent-color, #c5a059) 10%, transparent);
        border: 1px solid color-mix(in srgb, var(--accent-color, #c5a059) 30%, transparent);
        border-radius: 8px;
        color: var(--text-color, #1c1c1c);
        cursor: pointer;
        transition: all 0.2s ease;
        font-size: 0.95rem;
    }

    button:hover {
        background: color-mix(in srgb, var(--accent-color, #c5a059) 20%, transparent);
        transform: translateY(-1px);
    }

    button.active {
        background: var(--accent-color, #c5a059);
        color: white;
        border-color: var(--accent-color, #c5a059);
    }

    .loop-control {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        cursor: pointer;
        color: var(--text-color, #1c1c1c);
    }

    .loop-control input[type="checkbox"] {
        width: 1.25rem;
        height: 1.25rem;
        cursor: pointer;
    }

    .player-container {
        display: flex;
        justify-content: center;
        margin: 2rem 0;
        padding: 2rem;
        background: color-mix(in srgb, var(--bg-color, #f5f5f0) 50%, transparent);
        border-radius: 12px;
    }

    .info {
        background: color-mix(in srgb, var(--bg-color, #f5f5f0) 30%, transparent);
        padding: 1.5rem;
        border-radius: 8px;
        border: 1px solid color-mix(in srgb, var(--text-color, #1c1c1c) 10%, transparent);
    }

    dl {
        display: grid;
        grid-template-columns: auto 1fr;
        gap: 0.75rem;
        margin: 0;
    }

    dt {
        font-weight: 600;
        color: var(--text-color, #1c1c1c);
    }

    dd {
        margin: 0;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 70%, transparent);
    }

    .technical-notes {
        background: color-mix(in srgb, var(--accent-color, #c5a059) 5%, transparent);
        padding: 1.5rem;
        border-radius: 8px;
        border-left: 4px solid var(--accent-color, #c5a059);
    }

    .technical-notes ul {
        margin: 0;
        padding-left: 1.5rem;
    }

    .technical-notes li {
        margin: 0.75rem 0;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 80%, transparent);
        line-height: 1.5;
    }

    .technical-notes strong {
        color: var(--accent-color, #c5a059);
    }
</style>
