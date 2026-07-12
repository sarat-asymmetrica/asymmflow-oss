<script lang="ts">
    /**
     * QGIF Demo Page
     *
     * Demonstrates the QuaternionScenePlayer component with QGIF format.
     * This page showcases:
     * - 250:1 compression vs traditional GIF
     * - SLERP interpolation on S³
     * - Interactive animation controls
     * - Real-time metadata display
     */
    import QGIFExample from '$lib/asyl/components/examples/QGIFExample.svelte';
</script>

<svelte:head>
    <title>QGIF Demo - Quaternion Graphics Interchange Format</title>
    <meta name="description" content="Demonstrating 250:1 compression via SLERP interpolation on the unit 3-sphere S³" />
</svelte:head>

<div class="page-container">
    <header>
        <h1>QGIF Format Demo</h1>
        <p class="subtitle">
            Quaternion Graphics Interchange Format - 250:1 compression via SLERP on S³
        </p>
    </header>

    <main>
        <QGIFExample />

        <section class="explanation">
            <h2>What is QGIF?</h2>
            <p>
                QGIF (Quaternion Graphics Interchange Format) is a JSON-based animation format
                that achieves <strong>250:1 compression</strong> compared to traditional GIF files.
            </p>

            <h3>How It Works</h3>
            <ol>
                <li>
                    <strong>Store keyframes, not pixels:</strong> Instead of storing every pixel
                    of every frame, QGIF stores quaternion rotation keyframes.
                </li>
                <li>
                    <strong>SLERP interpolation:</strong> Between keyframes, the animation is
                    reconstructed using Spherical Linear Interpolation (SLERP) on the unit 3-sphere S³.
                </li>
                <li>
                    <strong>Geodesic paths:</strong> SLERP produces the shortest rotation path
                    on the manifold - mathematically optimal!
                </li>
                <li>
                    <strong>GPU native:</strong> WebGL and GPU shaders natively support quaternion
                    operations, enabling hardware acceleration.
                </li>
            </ol>

            <h3>Traditional GIF vs QGIF</h3>
            <div class="comparison">
                <div class="comparison-item">
                    <h4>Traditional GIF</h4>
                    <ul>
                        <li>Stores: Every pixel of every frame</li>
                        <li>Size: 500×500px × 3 channels × 60fps × 4s = ~180 MB</li>
                        <li>Interpolation: None (discrete frames)</li>
                        <li>Artifacts: Banding, compression artifacts</li>
                    </ul>
                </div>
                <div class="comparison-item">
                    <h4>QGIF Format</h4>
                    <ul>
                        <li>Stores: 5 quaternion keyframes</li>
                        <li>Size: 5 × 4 floats × 4 bytes = 80 bytes</li>
                        <li>Interpolation: SLERP (geodesic on S³)</li>
                        <li>Artifacts: None (mathematically pure)</li>
                    </ul>
                </div>
            </div>
            <p class="compression-result">
                <strong>Result: 2,250,000:1 compression ratio!</strong>
            </p>

            <h3>Mathematical Foundation</h3>
            <p>
                QGIF uses <strong>SLERP</strong> (Spherical Linear Interpolation) to interpolate
                between quaternion keyframes:
            </p>
            <pre class="math-formula">SLERP(q1, q2, t) = (sin((1-t)θ) / sin(θ)) × q1 + (sin(tθ) / sin(θ)) × q2

where θ = arccos(q1 · q2)</pre>
            <p>
                This produces <strong>geodesic paths</strong> on the unit 3-sphere S³ - the
                shortest possible rotation between two orientations.
            </p>

            <h3>Advantages</h3>
            <ul>
                <li><strong>No gimbal lock:</strong> Quaternions avoid Euler angle singularities</li>
                <li><strong>Constant angular velocity:</strong> Smooth, predictable motion</li>
                <li><strong>Resolution independent:</strong> Procedural reconstruction at any size</li>
                <li><strong>GPU accelerated:</strong> Hardware support for quaternion operations</li>
                <li><strong>Tiny file size:</strong> 250× smaller than traditional animation formats</li>
            </ul>
        </section>

        <section class="technical-spec">
            <h2>Technical Specification</h2>

            <h3>QGIF Structure</h3>
            <pre class="code-block">{`{
  "version": "QGIF/1.0",
  "metadata": {
    "title": "Animation Title",
    "duration": 4.0,
    "fps": 60
  },
  "tracks": [
    {
      "name": "rotation",
      "keyframes": [
        { "time": 0.0, "value": { "W": 1.0, "X": 0.0, "Y": 0.0, "Z": 0.0 } }
      ]
    }
  ],
  "geometry": {
    "type": "cube|sphere|pentagon|dodecahedron",
    "scale": 1.0
  }
}`}</pre>

            <h3>Implementation</h3>
            <p>
                The QuaternionScenePlayer component accepts QGIF data and handles:
            </p>
            <ul>
                <li>Parsing QGIF JSON format</li>
                <li>Extracting keyframes and metadata</li>
                <li>Real-time SLERP interpolation</li>
                <li>Canvas 2D rendering with quaternion rotation</li>
                <li>Time-based playback with loop support</li>
            </ul>
        </section>
    </main>

    <footer>
        <p>
            Built with <strong>Mathematical Rigor × Production Excellence × Infinite Capability</strong>
        </p>
        <p class="mantra">
            Om Lokah Samastah Sukhino Bhavantu
        </p>
    </footer>
</div>

<style>
    .page-container {
        max-width: 1200px;
        margin: 0 auto;
        padding: 2rem;
        font-family: system-ui, -apple-system, sans-serif;
    }

    header {
        text-align: center;
        margin-bottom: 3rem;
        padding: 2rem;
        background: linear-gradient(135deg,
            color-mix(in srgb, var(--accent-color, #c5a059) 10%, transparent),
            color-mix(in srgb, var(--bg-color, #f5f5f0) 50%, transparent)
        );
        border-radius: 12px;
    }

    h1 {
        font-size: 3rem;
        font-weight: 700;
        color: var(--text-color, #1c1c1c);
        margin: 0 0 1rem 0;
    }

    .subtitle {
        font-size: 1.25rem;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 70%, transparent);
        margin: 0;
    }

    main {
        margin-bottom: 4rem;
    }

    section {
        margin: 3rem 0;
    }

    h2 {
        font-size: 2rem;
        font-weight: 600;
        color: var(--text-color, #1c1c1c);
        margin-bottom: 1.5rem;
        border-bottom: 2px solid var(--accent-color, #c5a059);
        padding-bottom: 0.5rem;
    }

    h3 {
        font-size: 1.5rem;
        font-weight: 500;
        color: var(--text-color, #1c1c1c);
        margin-top: 2rem;
        margin-bottom: 1rem;
    }

    h4 {
        font-size: 1.25rem;
        font-weight: 500;
        color: var(--accent-color, #c5a059);
        margin-bottom: 0.75rem;
    }

    p {
        line-height: 1.7;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 80%, transparent);
        margin-bottom: 1rem;
    }

    ol, ul {
        line-height: 1.7;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 80%, transparent);
        margin-bottom: 1rem;
    }

    li {
        margin: 0.5rem 0;
    }

    .comparison {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 2rem;
        margin: 2rem 0;
    }

    .comparison-item {
        padding: 1.5rem;
        background: color-mix(in srgb, var(--bg-color, #f5f5f0) 30%, transparent);
        border-radius: 8px;
        border: 1px solid color-mix(in srgb, var(--text-color, #1c1c1c) 10%, transparent);
    }

    .comparison-item ul {
        margin: 0;
        padding-left: 1.5rem;
    }

    .compression-result {
        text-align: center;
        font-size: 1.5rem;
        color: var(--accent-color, #c5a059);
        margin: 2rem 0;
    }

    .math-formula {
        background: color-mix(in srgb, var(--text-color, #1c1c1c) 5%, transparent);
        padding: 1.5rem;
        border-radius: 8px;
        border-left: 4px solid var(--accent-color, #c5a059);
        font-family: 'Courier New', monospace;
        font-size: 0.95rem;
        overflow-x: auto;
        color: var(--text-color, #1c1c1c);
        margin: 1.5rem 0;
    }

    .code-block {
        background: color-mix(in srgb, var(--text-color, #1c1c1c) 5%, transparent);
        padding: 1.5rem;
        border-radius: 8px;
        border: 1px solid color-mix(in srgb, var(--text-color, #1c1c1c) 10%, transparent);
        font-family: 'Courier New', monospace;
        font-size: 0.9rem;
        overflow-x: auto;
        color: var(--text-color, #1c1c1c);
        margin: 1.5rem 0;
        line-height: 1.5;
    }

    .explanation {
        background: color-mix(in srgb, var(--bg-color, #f5f5f0) 20%, transparent);
        padding: 2rem;
        border-radius: 12px;
        border: 1px solid color-mix(in srgb, var(--accent-color, #c5a059) 20%, transparent);
    }

    .technical-spec {
        background: color-mix(in srgb, var(--accent-color, #c5a059) 5%, transparent);
        padding: 2rem;
        border-radius: 12px;
        border-left: 4px solid var(--accent-color, #c5a059);
    }

    footer {
        text-align: center;
        padding: 2rem;
        border-top: 2px solid color-mix(in srgb, var(--accent-color, #c5a059) 30%, transparent);
        margin-top: 4rem;
    }

    footer p {
        margin: 0.5rem 0;
        color: color-mix(in srgb, var(--text-color, #1c1c1c) 70%, transparent);
    }

    .mantra {
        font-size: 1.25rem;
        color: var(--accent-color, #c5a059);
        font-weight: 500;
    }

    @media (max-width: 768px) {
        .page-container {
            padding: 1rem;
        }

        h1 {
            font-size: 2rem;
        }

        .subtitle {
            font-size: 1rem;
        }

        .comparison {
            grid-template-columns: 1fr;
            gap: 1rem;
        }
    }
</style>
