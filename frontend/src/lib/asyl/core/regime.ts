
export enum Regime {
    Discovery = 'discovery',
    Refinement = 'refinement',
    Completion = 'completion'
}

export const FIBONACCI_SPACING = {
    base: 1.618, // rem
    unit: 8, // px
    sequence: [1, 2, 3, 5, 8, 13, 21, 34, 55, 89]
};

export const ANIMATION_DURATION = {
    short: 0.236, // 1/phi^3
    medium: 0.382, // 1/phi^2
    long: 0.618 // 1/phi
};
