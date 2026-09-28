/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./cmd/api/web/**/*.{html,js}'],
  theme: {
    extend: {
      colors: {
        ink: '#17251f',
        forest: '#245c48',
        mint: '#e3f2e9',
        coral: '#e66f4b',
        paper: '#f6f7f2',
      },
      fontFamily: {
        display: ['Georgia', 'serif'],
        sans: ['ui-sans-serif', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
};
