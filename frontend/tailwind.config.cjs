const config = {
  content: ["./index.html", "./src/**/*.{html,js,svelte,ts}"],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#000000', // Black
          dark: '#1a1a1a',
        },
        secondary: {
          DEFAULT: '#FFFFFF', // White
          light: '#F5F5F5',
        },
        paper: '#FFFFFF', // White
        surface: '#FFFFFF',
        ink: {
          DEFAULT: '#000000', // Black
          light: '#404040',
          faint: '#A0A0A0',
        },
        gold: '#000000', // Replaced gold with black
        safe: '#000000', // Monochrome success
        danger: '#CC0000', // Minimal red
        stone: '#E5E5E5',
      },
      fontFamily: {
        sans: ['Gravitica', 'system-ui', 'sans-serif'],
        display: ['Gravitica', 'sans-serif'],
        serif: ['Georgia', 'serif'], // Fallback
        mono: ['Menlo', 'Monaco', 'Courier New', 'monospace'],
      },
      backgroundImage: {
        'gradient-brand': 'linear-gradient(135deg, #000000 0%, #333333 100%)',
      }
    },
  },
  plugins: [],
};

module.exports = config;
