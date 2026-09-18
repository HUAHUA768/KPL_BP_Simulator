/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      gridTemplateColumns: {
        // 英雄池固定每行 10 个
        '10': 'repeat(10, minmax(0, 1fr))',
      },
    },
  },
  plugins: [],
}