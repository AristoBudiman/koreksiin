/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'system-ui', '-apple-system', 'sans-serif']
      },
      colors: {
        brand: {
          50: '#FEECEB',
          100: '#FED5D2',
          200: '#FDB0AA',
          300: '#FC8179',
          400: '#FB6156',
          500: '#FF5347',
          600: '#ED2A1D',
          700: '#CB1D10',
          800: '#A0170D',
          900: '#76110A'
        },
        ink: {
          DEFAULT: '#1E1E1E',
          muted: '#6B6B6B',
          line: '#E5E5E5',
          bg: '#FAFAFA',
          card: '#FFFFFF',
          darkBg: '#141414',
          darkCard: '#1E1E1E',
          darkText: '#F5F5F5',
          darkLine: '#2E2E2E'
        },
        criteria: {
          terpenuhi: '#16A34A',
          sebagian: '#D97706',
          tidakAda: '#64748B'
        }
      },
      boxShadow: {
        paper: '0 2px 10px -2px rgba(30, 30, 30, 0.06), 0 1px 3px 0 rgba(30, 30, 30, 0.04)',
        paperHover: '0 8px 20px -4px rgba(30, 30, 30, 0.1), 0 2px 6px -1px rgba(30, 30, 30, 0.05)'
      }
    }
  },
  plugins: []
}

