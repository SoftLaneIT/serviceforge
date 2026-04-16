import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  "#f0fdfb",
          100: "#ccfbf4",
          200: "#99f5e9",
          300: "#5ee8d8",
          400: "#2dd4c4",
          500: "#14b8aa",
          600: "#0f766e",
          700: "#0d6360",
          800: "#0f4f4e",
          900: "#114240",
        },
      },
      fontFamily: {
        sans: ['"IBM Plex Sans"', '"Segoe UI"', "system-ui", "sans-serif"],
        mono: ['"IBM Plex Mono"', "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;
