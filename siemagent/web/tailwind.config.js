/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Geist Sans', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['Geist Mono', 'JetBrains Mono', 'monospace'],
      },
      colors: {
        severity: {
          P1: '#DC2626',
          P2: '#EA580C',
          P3: '#CA8A04',
          P4: '#2563EB',
          P5: '#6B7280',
        },
      },
    },
  },
  plugins: [],
}

