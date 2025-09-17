import colors from "tailwindcss/colors";

/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class", // enables class-based dark mode
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}", // include all source files
  ],
  theme: {
    extend: {
      colors: {
        primary: { DEFAULT: colors.violet[600], foreground: colors.white },
        secondary: { DEFAULT: colors.zinc[700], foreground: colors.white },
        accent: { DEFAULT: colors.violet[500], foreground: colors.white },
      },
      fontFamily: {
        sans: ["JetBrains Mono", "monospace"],
        mono: ["JetBrains Mono", "monospace"],
      },
    },
  },
  plugins: [],
};
