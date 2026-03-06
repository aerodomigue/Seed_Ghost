/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        ghost: {
          50: '#ecfdf5',
          100: '#d1fae5',
          200: '#a7f3d0',
          300: '#6ee7b7',
          400: '#2DD4A8',
          500: '#10b981',
          600: '#059669',
          700: '#047857',
          800: '#065f46',
          900: '#064e3b',
          950: '#022c22',
        },
        dark: {
          50:  '#f2f8f5',
          100: '#e6efe9',
          200: '#d1dfd8',
          300: '#afc2b8',
          400: '#829a90',
          500: '#5a756a',
          600: '#3a4f46',
          700: '#25342d',
          800: '#1a2620',
          900: '#111916',
          950: '#0a0f0d',
        },
      },
    },
  },
  plugins: [],
}
