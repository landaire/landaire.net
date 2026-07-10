/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html",
  ],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        mono: ["BerkeleyMono", "ui-monospace", "SFMono-Regular", "monospace"],
        sans: [
          "system-ui",
          "-apple-system",
          "Segoe UI",
          "Roboto",
          "Helvetica Neue",
          "Arial",
          "sans-serif",
        ],
      },
      colors: {
        accent: {
          50: "#eff6ff",
          100: "#dbeafe",
          200: "#bfdbfe",
          300: "#93c5fd",
          400: "#60a5fa",
          500: "#3b82f6",
          600: "#2563eb",
          700: "#1d4ed8",
          800: "#1e40af",
          900: "#1e3a8a",
        },
      },
      typography: ({ theme }) => ({
        DEFAULT: {
          css: {
            "--tw-prose-links": theme("colors.accent.600"),
            "--tw-prose-code": theme("colors.gray.800"),
            maxWidth: "none",
            a: {
              textDecoration: "none",
              backgroundImage:
                "linear-gradient(currentColor, currentColor)",
              backgroundSize: "0% 1px",
              backgroundRepeat: "no-repeat",
              backgroundPosition: "center bottom",
              "&:hover": {
                backgroundSize: "100% 1px",
              },
            },
            code: {
              fontFamily: theme("fontFamily.mono").join(", "),
              fontWeight: "400",
              fontVariantLigatures: "none",
            },
            "code::before": { content: "none" },
            "code::after": { content: "none" },
            pre: {
              fontFamily: theme("fontFamily.mono").join(", "),
              fontVariantLigatures: "none",
              borderRadius: theme("borderRadius.lg"),
              maxHeight: "80vh",
              overflowY: "auto",
              fontSize: "0.8125rem",
              lineHeight: "1.7",
              letterSpacing: "0",
            },
            "pre code": {
              fontFamily: "inherit",
              fontVariantLigatures: "none",
              fontSize: "inherit",
              letterSpacing: "0",
            },
            blockquote: {
              fontStyle: "italic",
              borderLeftWidth: "2px",
            },
            img: {
              borderRadius: theme("borderRadius.lg"),
            },
            table: {
              fontSize: theme("fontSize.sm")[0],
            },
          },
        },
        invert: {
          css: {
            "--tw-prose-links": theme("colors.accent.400"),
            "--tw-prose-code": theme("colors.gray.200"),
          },
        },
      }),
    },
  },
  plugins: [require("@tailwindcss/typography")],
};
