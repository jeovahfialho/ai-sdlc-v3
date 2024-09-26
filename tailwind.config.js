/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      backgroundColor: {
        background: "hsl(var(--background))",
      },
      textColor: {
        foreground: "hsl(var(--foreground))",
      },
      colors: {
        muted: "hsl(var(--muted))",
        "muted-foreground": "hsl(var(--muted-foreground))",
      },
    },
  },
  plugins: [],
}