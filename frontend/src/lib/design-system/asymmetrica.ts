// ============================================================
// ASYMMETRICA DESIGN SYSTEM - Mathematical UI/UX Specification
// ============================================================
// "Better Math for Everyone" - Every component derived from equations
// ============================================================

// ============================================================
// 1. SACRED CONSTANTS
// ============================================================

/** Golden Ratio - The divine proportion */
export const PHI = 1.618033988749895;

/** Inverse Golden Ratio */
export const PHI_INV = 0.618033988749895;

/** Square root of PHI - Musical interval (perfect fifth) */
export const PHI_SQRT = Math.sqrt(PHI); // ≈ 1.272

/** Wobble frequency for organic circles */
export const WOBBLE_K = 13.7;

/** Default imperfection coefficient */
export const EPSILON = 0.35;

/** Breath cycle periods (ms) */
export const BREATH = {
  MEDITATIVE: 5000,  // Slow, zen
  CALM: 2000,        // Relaxed
  ALERT: 700,        // Attention needed
} as const;

// ============================================================
// 2. GEOMETRY FUNCTIONS - G(x,y)
// ============================================================

/**
 * Golden ratio spacing scale
 * space(n) = 8 × φⁿ
 */
export function space(n: number): number {
  return Math.round(8 * Math.pow(PHI, n));
}

/** Pre-computed spacing scale */
export const SPACE = {
  0: space(0),   // 8px  - micro
  1: space(1),   // 13px - small
  2: space(2),   // 21px - medium
  3: space(3),   // 34px - large
  4: space(4),   // 55px - xlarge
  5: space(5),   // 89px - section
  6: space(6),   // 144px - hero
} as const;

/**
 * Typography scale based on musical intervals
 * size(n) = 16 × φ^(n/2)
 */
export function fontSize(n: number): number {
  return Math.round(16 * Math.pow(PHI_SQRT, n));
}

/** Pre-computed type scale */
export const TYPE_SCALE = {
  '-2': fontSize(-2),  // 10px - caption
  '-1': fontSize(-1),  // 12px - small
  '0': fontSize(0),    // 16px - body
  '1': fontSize(1),    // 20px - lead
  '2': fontSize(2),    // 26px - h3
  '3': fontSize(3),    // 33px - h2
  '4': fontSize(4),    // 42px - h1
  '5': fontSize(5),    // 53px - display
} as const;

// ============================================================
// 3. IMPERFECTION FUNCTIONS - I(ε)
// ============================================================

/**
 * Wabi-sabi noise function
 * Returns pseudo-random value in [0, 1] based on position
 */
export function noise(x: number, y: number): number {
  return Math.abs(Math.sin(x * 12.9898 + y * 78.233) * 43758.5453 % 1);
}

/**
 * Apply imperfection to a value
 * I(v, ε) = v + ε × noise × amplitude
 */
export function imperfect(value: number, epsilon: number = EPSILON, amplitude: number = 10): number {
  const n = noise(value, value * 0.7);
  return value + (n * 2 - 1) * epsilon * amplitude;
}

/**
 * Generate imperfect circle radius at angle θ
 * r(θ) = R × (1 + ω × sin(θ × k))
 */
export function wobbleRadius(baseRadius: number, theta: number, wobble: number = 0.02): number {
  return baseRadius * (1 + wobble * Math.sin(theta * WOBBLE_K));
}

/**
 * Stroke fade function
 * opacity(i) = α₀ × λⁱ
 */
export function strokeFade(index: number, initialAlpha: number = 0.8, fadeRate: number = 0.97): number {
  return initialAlpha * Math.pow(fadeRate, index);
}

// ============================================================
// 4. BREATH FUNCTIONS - B(t)
// ============================================================

/**
 * Breathing animation value
 * B(t) = A × sin(ω × t + φ₀)
 */
export function breath(
  time: number,
  amplitude: number = 12,
  period: number = BREATH.MEDITATIVE,
  phase: number = 0
): number {
  const omega = (2 * Math.PI) / period;
  return amplitude * Math.sin(omega * time + phase);
}

/**
 * Easing: Wabi (decelerate - natural stop)
 * ease_wabi(t) = t × (2 - t)
 */
export function easeWabi(t: number): number {
  return t * (2 - t);
}

/**
 * Easing: Sabi (smooth S-curve - organic)
 * ease_sabi(t) = t² × (3 - 2t)
 */
export function easeSabi(t: number): number {
  return t * t * (3 - 2 * t);
}

// ============================================================
// 5. COLOR SYSTEM - C(context)
// ============================================================

export const COLOR = {
  // Core palette (from SSOT)
  paper: '#fdfbf7',
  ink: '#1c1c1c',
  inkLight: '#57534e',
  
  // Semantic colors
  safe: '#15803d',
  danger: '#ef4444',
  warning: '#fbbf24',
  gold: '#c5a059',      // Kintsugi accent
  
  // Competition/heavy
  stone: '#475569',
  
  // Functional
  info: '#3b82f6',
} as const;

/** Opacity levels */
export const ALPHA = {
  solid: 1.0,
  emphasis: 0.87,
  medium: 0.60,
  subtle: 0.38,
  hint: 0.12,
  ghost: 0.05,
} as const;

/**
 * Apply alpha to a hex color
 */
export function withAlpha(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

/**
 * Semantic color mapping
 */
export type SemanticState = 'neutral' | 'safe' | 'danger' | 'warning' | 'focus' | 'info';

export function semanticColor(state: SemanticState): { bg: string; fg: string } {
  const map: Record<SemanticState, { bg: string; fg: string }> = {
    neutral: { bg: COLOR.paper, fg: COLOR.ink },
    safe: { bg: COLOR.paper, fg: COLOR.safe },
    danger: { bg: COLOR.paper, fg: COLOR.danger },
    warning: { bg: COLOR.paper, fg: COLOR.warning },
    focus: { bg: COLOR.ink, fg: COLOR.paper },
    info: { bg: COLOR.paper, fg: COLOR.info },
  };
  return map[state];
}

// ============================================================
// 6. TYPOGRAPHY - T(role)
// ============================================================

export const FONT = {
  prose: "'Georgia', 'Times New Roman', serif",
  data: "'Courier Prime', 'Courier New', monospace",
  ui: "system-ui, -apple-system, sans-serif",
} as const;

export const LINE_HEIGHT = {
  tight: 1.1,
  heading: 1.2,
  body: 1.5,
  relaxed: 1.7,
} as const;

// ============================================================
// 7. COMPONENT FORMULAS
// ============================================================

/** Card styling formula */
export const CARD = {
  background: withAlpha('#ffffff', 0.4),
  border: `1px solid ${withAlpha('#000000', 0.05)}`,
  padding: `${SPACE[3]}px`,
  borderRadius: `${SPACE[1]}px`,
  shadow: 'none',
  shadowHover: `0 ${SPACE[2]}px ${SPACE[4]}px ${withAlpha('#000000', 0.05)}`,
  transition: `all 0.3s cubic-bezier(0.4, 0, 0.2, 1)`,
} as const;

/** Button styling formula */
export function buttonStyle(variant: 'primary' | 'secondary' | 'ghost' = 'primary') {
  const base = {
    padding: `${SPACE[1]}px ${SPACE[3]}px`,
    borderRadius: `${SPACE[0]}px`,
    fontFamily: FONT.prose,
    fontSize: `${TYPE_SCALE['0']}px`,
    transition: 'all 0.2s cubic-bezier(0, 0, 0.2, 1)',
    cursor: 'pointer',
  };
  
  const variants = {
    primary: {
      ...base,
      background: COLOR.ink,
      color: COLOR.paper,
      border: `1px solid ${COLOR.ink}`,
    },
    secondary: {
      ...base,
      background: 'transparent',
      color: COLOR.ink,
      border: `1px solid ${COLOR.ink}`,
    },
    ghost: {
      ...base,
      background: 'transparent',
      color: COLOR.ink,
      border: '1px solid transparent',
    },
  };
  
  return variants[variant];
}

/** Input styling formula */
export const INPUT = {
  padding: `${SPACE[1]}px ${SPACE[2]}px`,
  background: withAlpha('#ffffff', 0.6),
  border: `1px solid ${withAlpha('#000000', 0.1)}`,
  borderRadius: `${SPACE[0]}px`,
  fontFamily: FONT.prose,
  fontSize: `${TYPE_SCALE['0']}px`,
  color: COLOR.ink,
  transition: 'border-color 0.2s ease',
  focusBorder: COLOR.ink,
} as const;

// ============================================================
// 8. ANIMATION PRESETS
// ============================================================

export const ANIMATION = {
  // Durations (based on φ)
  instant: 100,
  fast: Math.round(100 * PHI),      // 162ms
  normal: Math.round(100 * PHI * PHI), // 262ms
  slow: Math.round(100 * PHI * PHI * PHI), // 424ms
  
  // Easing curves
  easeWabi: 'cubic-bezier(0, 0, 0.2, 1)',      // Decelerate
  easeSabi: 'cubic-bezier(0.4, 0, 0.2, 1)',    // Standard
  easeEnter: 'cubic-bezier(0, 0, 0.2, 1)',     // Enter
  easeExit: 'cubic-bezier(0.4, 0, 1, 1)',      // Exit
  
  // Breath keyframes
  breathe: `
    @keyframes breathe {
      0%, 100% { transform: scale(1); opacity: 0.8; }
      50% { transform: scale(1.05); opacity: 1; }
    }
  `,
  
  // Pulse keyframes
  pulse: `
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  `,
  
  // Fade in
  fadeIn: `
    @keyframes fadeIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }
  `,
  
  // Slide up
  slideUp: `
    @keyframes slideUp {
      from { transform: translateY(20px); opacity: 0; }
      to { transform: translateY(0); opacity: 1; }
    }
  `,
} as const;

// ============================================================
// 9. CSS VARIABLE GENERATOR
// ============================================================

/**
 * Generate CSS custom properties from the design system
 */
export function generateCSSVariables(): string {
  return `
:root {
  /* Geometry (φ-based spacing) */
  --space-0: ${SPACE[0]}px;
  --space-1: ${SPACE[1]}px;
  --space-2: ${SPACE[2]}px;
  --space-3: ${SPACE[3]}px;
  --space-4: ${SPACE[4]}px;
  --space-5: ${SPACE[5]}px;
  --space-6: ${SPACE[6]}px;
  
  /* Colors (semantic) */
  --color-paper: ${COLOR.paper};
  --color-ink: ${COLOR.ink};
  --color-ink-light: ${COLOR.inkLight};
  --color-safe: ${COLOR.safe};
  --color-danger: ${COLOR.danger};
  --color-warning: ${COLOR.warning};
  --color-gold: ${COLOR.gold};
  --color-stone: ${COLOR.stone};
  --color-info: ${COLOR.info};
  
  /* Typography (musical scale) */
  --text-xs: ${TYPE_SCALE['-2']}px;
  --text-sm: ${TYPE_SCALE['-1']}px;
  --text-base: ${TYPE_SCALE['0']}px;
  --text-lg: ${TYPE_SCALE['1']}px;
  --text-xl: ${TYPE_SCALE['2']}px;
  --text-2xl: ${TYPE_SCALE['3']}px;
  --text-3xl: ${TYPE_SCALE['4']}px;
  --text-4xl: ${TYPE_SCALE['5']}px;
  
  /* Fonts */
  --font-prose: ${FONT.prose};
  --font-data: ${FONT.data};
  --font-ui: ${FONT.ui};
  
  /* Line heights */
  --leading-tight: ${LINE_HEIGHT.tight};
  --leading-heading: ${LINE_HEIGHT.heading};
  --leading-body: ${LINE_HEIGHT.body};
  --leading-relaxed: ${LINE_HEIGHT.relaxed};
  
  /* Opacity */
  --alpha-solid: ${ALPHA.solid};
  --alpha-emphasis: ${ALPHA.emphasis};
  --alpha-medium: ${ALPHA.medium};
  --alpha-subtle: ${ALPHA.subtle};
  --alpha-hint: ${ALPHA.hint};
  --alpha-ghost: ${ALPHA.ghost};
  
  /* Animation */
  --duration-instant: ${ANIMATION.instant}ms;
  --duration-fast: ${ANIMATION.fast}ms;
  --duration-normal: ${ANIMATION.normal}ms;
  --duration-slow: ${ANIMATION.slow}ms;
  --ease-wabi: ${ANIMATION.easeWabi};
  --ease-sabi: ${ANIMATION.easeSabi};
  
  /* Imperfection */
  --epsilon: ${EPSILON};
  --wobble-k: ${WOBBLE_K};
  
  /* Breath */
  --breath-meditative: ${BREATH.MEDITATIVE}ms;
  --breath-calm: ${BREATH.CALM}ms;
  --breath-alert: ${BREATH.ALERT}ms;
  
  /* Sacred constants */
  --phi: ${PHI};
  --phi-inv: ${PHI_INV};
}
`;
}

// ============================================================
// 10. CANVAS DRAWING UTILITIES
// ============================================================

/**
 * Draw a sumi-e brush stroke on canvas
 */
export function drawSumiEStroke(
  ctx: CanvasRenderingContext2D,
  x1: number, y1: number,
  x2: number, y2: number,
  options: {
    variability?: number;
    fadeRate?: number;
    width?: number;
    segments?: number;
    color?: string;
  } = {}
): void {
  const {
    variability = EPSILON,
    fadeRate = 0.97,
    width = 4,
    segments = 60,
    color = withAlpha(COLOR.ink, 0.6),
  } = options;

  const points: Array<{ x: number; y: number }> = [];

  for (let i = 0; i <= segments; i++) {
    const t = i / segments;
    let x = x1 + (x2 - x1) * t;
    let y = y1 + (y2 - y1) * t;

    const jx = (noise(x * 0.01, y * 0.01) * 2 - 1) * variability * 10;
    const jy = (noise(y * 0.02, x * 0.02) * 2 - 1) * variability * 10;

    points.push({ x: x + jx, y: y + jy });
  }

  ctx.lineCap = 'round';
  for (let i = 0; i < points.length - 1; i++) {
    const p = points[i];
    const q = points[i + 1];

    const alpha = strokeFade(i, 0.6, fadeRate);
    ctx.strokeStyle = color.replace(/[\d.]+\)$/, `${alpha})`);
    ctx.lineWidth = width * (1 + (Math.random() * variability - variability / 2));

    ctx.beginPath();
    ctx.moveTo(p.x, p.y);
    ctx.lineTo(q.x, q.y);
    ctx.stroke();
  }
}

/**
 * Draw an imperfect circle (enso) on canvas
 */
export function drawWabiCircle(
  ctx: CanvasRenderingContext2D,
  cx: number, cy: number,
  radius: number,
  options: {
    wobble?: number;
    strokes?: number;
    color?: string;
    lineWidth?: number;
    breathOffset?: number;
  } = {}
): void {
  const {
    wobble = 0.02,
    strokes = 120,
    color = withAlpha(COLOR.ink, 0.3),
    lineWidth = 1.5,
    breathOffset = 0,
  } = options;

  ctx.beginPath();

  for (let i = 0; i <= strokes; i++) {
    const theta = (i / strokes) * Math.PI * 2;
    const r = radius + Math.sin(theta * WOBBLE_K + breathOffset) * wobble * radius;

    const x = cx + r * Math.cos(theta);
    const y = cy + r * Math.sin(theta);

    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  }

  ctx.strokeStyle = color;
  ctx.lineWidth = lineWidth;
  ctx.stroke();
}

// ============================================================
// EXPORT ALL
// ============================================================

export default {
  // Constants
  PHI, PHI_INV, PHI_SQRT, WOBBLE_K, EPSILON, BREATH,
  // Scales
  SPACE, TYPE_SCALE, COLOR, ALPHA, FONT, LINE_HEIGHT,
  // Functions
  space, fontSize, noise, imperfect, wobbleRadius, strokeFade,
  breath, easeWabi, easeSabi, withAlpha, semanticColor,
  buttonStyle, generateCSSVariables,
  drawSumiEStroke, drawWabiCircle,
  // Presets
  CARD, INPUT, ANIMATION,
};
