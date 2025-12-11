/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html",
    "./static/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        // Jellycat-inspired soft pastel palette
        jellycat: {
          cream: '#FFF9F0',
          beige: '#F5EDE0',
          'soft-pink': '#FFE5EC',
          'blush': '#FFB5C5',
          'lavender': '#E8D5F2',
          'lilac': '#D5B8E0',
          'sky': '#E0F2FE',
          'mint': '#D5F5E3',
          'peach': '#FFE4D6',
        },
        // Football-themed pastel accents
        football: {
          'grass': '#C8E6C9',
          'field': '#A5D6A7',
          'line': '#F0F0F0',
          'orange': '#FFCC80',
          'goal': '#FFE082',
        },
        // Enhanced primary palette with Jellycat warmth
        primary: {
          50: '#FFF9F0',
          100: '#FFE5EC',
          200: '#FFD1DC',
          300: '#FFB5C5',
          400: '#FF9AB5',
          500: '#FF7FA5',
          600: '#FF6B9D',
          700: '#E85D8C',
          800: '#D14B7B',
          900: '#B83A6A',
        },
        secondary: {
          50: '#F3E5F5',
          100: '#E8D5F2',
          200: '#D5B8E0',
          300: '#C19BD0',
          400: '#AE7FC0',
          500: '#9B62B0',
          600: '#8854A0',
          700: '#764690',
          800: '#633880',
          900: '#512A70',
        },
      },
      fontFamily: {
        sans: [
          'Nunito',
          'Quicksand',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Oxygen',
          'Ubuntu',
          'Cantarell',
          'Helvetica Neue',
          'sans-serif',
        ],
        display: ['Quicksand', 'Nunito', 'sans-serif'],
      },
      borderRadius: {
        'xl': '1rem',
        '2xl': '1.5rem',
        '3xl': '2rem',
        'jellycat': '1.25rem',
      },
      boxShadow: {
        'soft': '0 2px 15px -3px rgba(255, 182, 193, 0.3), 0 4px 6px -2px rgba(255, 182, 193, 0.15)',
        'soft-lg': '0 10px 30px -5px rgba(255, 182, 193, 0.4), 0 8px 10px -5px rgba(255, 182, 193, 0.2)',
        'football': '0 4px 20px -2px rgba(168, 214, 167, 0.4)',
      },
      animation: {
        'bounce-slow': 'bounce 3s infinite',
        'float': 'float 3s ease-in-out infinite',
        'wiggle': 'wiggle 1s ease-in-out infinite',
        'pulse-soft': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-10px)' },
        },
        wiggle: {
          '0%, 100%': { transform: 'rotate(-3deg)' },
          '50%': { transform: 'rotate(3deg)' },
        },
      },
      backgroundImage: {
        'gradient-jellycat': 'linear-gradient(135deg, #FFE5EC 0%, #E8D5F2 50%, #E0F2FE 100%)',
        'gradient-football': 'linear-gradient(to bottom, #C8E6C9 0%, #A5D6A7 100%)',
        'gradient-warm': 'linear-gradient(135deg, #FFF9F0 0%, #FFE4D6 100%)',
      },
    },
  },
  plugins: [],
}
