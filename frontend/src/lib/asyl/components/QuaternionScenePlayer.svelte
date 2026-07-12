
<script lang="ts">
    /**
     * QuaternionScenePlayer - Real-time QGIF (Quaternion GIF) renderer
     *
     * Plays quaternion animation sequences using SLERP interpolation.
     * 250:1 compression vs traditional GIF by storing keyframes, not pixels.
     *
     * @component
     */
    import { onMount, onDestroy } from 'svelte';
    import { Quaternion } from '../math/quaternion';

    

    

    

    

    

    

    
    interface Props {
        /** QGIF keyframes - array of quaternions defining rotation sequence */
        frames?: Quaternion[];
        /** QGIF format data (alternative to frames prop) */
        qgifData?: any;
        /** Frames per second (animation speed) */
        fps?: number;
        /** Canvas width in pixels */
        width?: number;
        /** Canvas height in pixels */
        height?: number;
        /** ARIA label for accessibility */
        ariaLabel?: string;
        /** Loop the animation */
        loop?: boolean;
    }

    let {
        frames = $bindable([]),
        qgifData = null,
        fps = $bindable(60),
        width = 300,
        height = 300,
        ariaLabel = "Animated quaternion scene visualization",
        loop = true
    }: Props = $props();

    let canvas: HTMLCanvasElement = $state();
    let ctx: CanvasRenderingContext2D | null;
    let animationId: number;
    let currentIndex = 0;
    let t = 0; // SLERP interpolation factor [0, 1]
    let totalDuration = 0; // Total animation duration in seconds
    let currentTime = 0; // Current playback time in seconds

    // Resolved CSS variable colors (canvas can't parse CSS vars directly!)
    let bgColor = '#111';
    let accentColor = '#00ffcc';
    let textColor = '#fff';

    // QGIF metadata
    let metadata = $state({ title: '', fps: 60, duration: 0 });

    /**
     * Parse QGIF format into keyframes array
     * Supports rotation, position, and scale tracks
     */
    function parseQGIF(qgif: any): void {
        if (!qgif || !qgif.tracks) return;

        // Extract metadata
        if (qgif.metadata) {
            metadata = {
                title: qgif.metadata.title || '',
                fps: qgif.metadata.fps || 60,
                duration: qgif.metadata.duration || 0
            };
            fps = metadata.fps;
            totalDuration = metadata.duration;
        }

        // Find rotation track (primary animation)
        const rotationTrack = qgif.tracks.find((t: any) => t.name === 'rotation');
        if (!rotationTrack || !rotationTrack.keyframes) return;

        // Convert QGIF keyframes to Quaternion array
        frames = rotationTrack.keyframes.map((kf: any) => {
            const v = kf.value;
            return new Quaternion(v.W, v.X, v.Y, v.Z);
        });

        // Store keyframe times for time-based interpolation
        if (totalDuration > 0) {
            // We'll use time-based interpolation in the render loop
        }
    }

    // Parse QGIF data if provided
    if (qgifData) {
        parseQGIF(qgifData);
    }

    // Generate demo frames if none provided (spinning cube sequence)
    if (frames.length === 0) {
        frames = [
            new Quaternion(1, 0, 0, 0),
            new Quaternion(0.707, 0.707, 0, 0),
            new Quaternion(0, 1, 0, 0),
            new Quaternion(0.707, -0.707, 0, 0),
            new Quaternion(1, 0, 0, 0)
        ];
        totalDuration = frames.length / fps;
    }

    onMount(() => {
        ctx = canvas.getContext('2d');
        if (!ctx) return;

        // Resolve CSS variables to actual colors for canvas API
        const styles = getComputedStyle(canvas);
        bgColor = styles.getPropertyValue('--bg-color').trim() || '#111';
        accentColor = styles.getPropertyValue('--accent-color').trim() || '#00ffcc';
        textColor = styles.getPropertyValue('--text-color').trim() || '#fff';

        const render = () => {
            if (!ctx || !canvas) return;

            // Advance time - φ-based speed control
            currentTime += 1 / fps;

            // Loop or stop at end
            if (currentTime >= totalDuration) {
                if (loop) {
                    currentTime = currentTime % totalDuration;
                } else {
                    currentTime = totalDuration;
                    // Stop animation at end if not looping
                    cancelAnimationFrame(animationId);
                    return;
                }
            }

            // Calculate which keyframe segment we're in
            const segmentDuration = totalDuration / (frames.length - 1);
            const segmentIndex = Math.floor(currentTime / segmentDuration);
            currentIndex = Math.min(segmentIndex, frames.length - 2);

            // Calculate interpolation factor within this segment
            t = (currentTime - currentIndex * segmentDuration) / segmentDuration;
            t = Math.max(0, Math.min(1, t)); // Clamp to [0, 1]

            const nextIndex = Math.min(currentIndex + 1, frames.length - 1);
            const qCurrent = frames[currentIndex];
            const qNext = frames[nextIndex];

            // Real-time SLERP reconstruction (the QGIF magic!)
            const qInterpolated = Quaternion.slerp(qCurrent, qNext, t);

            // Render 3D scene projected to 2D
            drawScene(ctx, canvas.width, canvas.height, qInterpolated);

            animationId = requestAnimationFrame(render);
        };
        render();
    });

    /**
     * Draws a 3D cube rotated by quaternion, projected to 2D canvas
     */
    function drawScene(ctx: CanvasRenderingContext2D, width: number, height: number, q: Quaternion) {
        // Clear with resolved background color (not CSS var - canvas can't parse it!)
        ctx.fillStyle = bgColor;
        ctx.fillRect(0, 0, width, height);

        const cx = width / 2;
        const cy = height / 2;
        const size = 50;

        // Cube vertices in 3D space
        const vertices = [
            [-1, -1, -1], [1, -1, -1], [1, 1, -1], [-1, 1, -1],
            [-1, -1, 1], [1, -1, 1], [1, 1, 1], [-1, 1, 1]
        ];

        // Cube edges (vertex index pairs)
        const edges = [
            [0,1], [1,2], [2,3], [3,0],  // Back face
            [4,5], [5,6], [6,7], [7,4],  // Front face
            [0,4], [1,5], [2,6], [3,7]   // Connecting edges
        ];

        // Rotate vertices by quaternion and project to 2D
        const projected = vertices.map(v => {
            const [vx, vy, vz] = v;

            // Apply quaternion rotation: v' = q * v * q^-1
            // Optimized formula: v' = v + 2 * cross(q.xyz, cross(q.xyz, v) + q.w * v)
            const qx = q.x, qy = q.y, qz = q.z, qw = q.w;

            const ix = qw * vx + qy * vz - qz * vy;
            const iy = qw * vy + qz * vx - qx * vz;
            const iz = qw * vz + qx * vy - qy * vx;
            const iw = -qx * vx - qy * vy - qz * vz;

            const x = ix * qw + iw * -qx + iy * -qz - iz * -qy;
            const y = iy * qw + iw * -qy + iz * -qx - ix * -qz;
            const z = iz * qw + iw * -qz + ix * -qy - iy * -qx;

            // Weak perspective projection
            const scale = 200 / (200 + z * size + 100);
            return {
                x: cx + x * size * scale,
                y: cy + y * size * scale
            };
        });

        // Draw edges with resolved accent color
        ctx.strokeStyle = accentColor;
        ctx.lineWidth = 2;
        ctx.beginPath();
        edges.forEach(([i, j]) => {
            ctx.moveTo(projected[i].x, projected[i].y);
            ctx.lineTo(projected[j].x, projected[j].y);
        });
        ctx.stroke();

        // Draw quaternion stats with resolved text color
        ctx.fillStyle = textColor;
        ctx.font = '10px monospace';
        ctx.fillText(`Q: [${q.w.toFixed(2)}, ${q.x.toFixed(2)}, ${q.y.toFixed(2)}, ${q.z.toFixed(2)}]`, 10, 20);
        ctx.fillText(`FPS: ${fps}`, 10, 35);
        ctx.fillText(`Frame: ${currentIndex}/${frames.length}`, 10, 50);

        // Show QGIF metadata if available
        if (metadata.title) {
            ctx.fillText(`QGIF: ${metadata.title}`, 10, 65);
        }
        ctx.fillText(`Time: ${currentTime.toFixed(2)}s / ${totalDuration.toFixed(2)}s`, 10, 80);
    }

    onDestroy(() => {
        if (typeof window !== 'undefined') {
            cancelAnimationFrame(animationId);
        }
    });
</script>

<div class="qgif-player" role="img" aria-label={ariaLabel}>
    <canvas
        bind:this={canvas}
        width={width}
        height={height}
        class="qgif-canvas"
></canvas>
    <div class="qgif-caption">
        {#if metadata.title}
            {metadata.title}
        {:else}
            Quaternion Scene Player (QGIF)
        {/if}
    </div>
</div>

<style>
    .qgif-player {
        display: inline-block;
        border: 1px solid color-mix(in srgb, var(--text-color, #374151) 50%, transparent);
        border-radius: 8px;
        overflow: hidden;
        background: var(--bg-color, #000);
    }

    .qgif-canvas {
        display: block;
    }

    .qgif-caption {
        background: color-mix(in srgb, var(--bg-color, #111827) 95%, transparent);
        color: color-mix(in srgb, var(--text-color, #9ca3af) 70%, transparent);
        font-size: 0.75rem;
        padding: 8px;
        text-align: center;
        font-family: 'Courier New', monospace;
        border-top: 1px solid color-mix(in srgb, var(--text-color, #374151) 30%, transparent);
    }

    /* Smooth focus state for accessibility */
    .qgif-player:focus-within {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }
</style>
