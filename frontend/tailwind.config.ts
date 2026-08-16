import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "class",
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        base: {
          950: "#0a0d14",
          900: "#0f1420",
          800: "#161d2e",
          700: "#212a3f",
        },
        accent: {
          blue: "#4f7cff",
          purple: "#9b5de5",
          teal: "#22d3c5",
        },
        up: "#5ebd8b",
        down: "#e0707a",
      },
      fontFamily: {
        body: ["var(--font-inter)", "sans-serif"],
        mono: ["var(--font-mono)", "monospace"],
      },
      backgroundImage: {
        "accent-gradient": "linear-gradient(135deg, #4f7cff 0%, #9b5de5 55%, #22d3c5 100%)",
      },
    },
  },
  plugins: [],
};

export default config;
