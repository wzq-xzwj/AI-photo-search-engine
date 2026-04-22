/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f3eeff',
          100: '#e4dbff',
          200: '#c9b7ff',
          300: '#ae93ff',
          400: '#9570ff',
          500: '#8155ff',
          600: '#6a3de6',
          700: '#5330b3',
          800: '#3c2380',
          900: '#25164d',
        },
        accent: {
          50: '#fce8f0',
          100: '#f9d1e1',
          200: '#f3a3c3',
          300: '#ed75a5',
          400: '#e74787',
          500: '#e1266c',
          600: '#b41e56',
          700: '#871741',
          800: '#5a0f2b',
          900: '#2d0816',
        },
        secondary: {
          50: '#e0f5f7',
          100: '#c1ebef',
          200: '#83d7df',
          300: '#45c3cf',
          400: '#22aab8',
          500: '#008393',
          600: '#006c79',
          700: '#00555f',
          800: '#003e45',
          900: '#00272b',
        },
        dark: {
          bg: '#0F111A',
          surface: '#1A1D2E',
          card: '#232640',
          border: '#2E3148',
          text: '#E4E6F0',
          muted: '#8B8FA8',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        display: ['Plus Jakarta Sans', 'Inter', 'sans-serif'],
      },
      boxShadow: {
        'glow': '0 0 20px rgba(129, 85, 255, 0.3)',
        'glow-sm': '0 0 10px rgba(129, 85, 255, 0.2)',
        'card': '0 4px 24px rgba(0, 0, 0, 0.12)',
        'card-dark': '0 4px 24px rgba(0, 0, 0, 0.4)',
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.4s ease-out',
        'pulse-soft': 'pulseSoft 2s infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(16px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        pulseSoft: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.7' },
        },
      },
    },
  },
  plugins: [],
}
