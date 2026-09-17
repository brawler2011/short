/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        bg: '#0f1117',
        surface: '#1a1d27',
        surfaceHover: '#232736',
        accent: '#5b7bff',
        accentHover: '#4b6be5',
      },
    },
  },
  plugins: [],
}
