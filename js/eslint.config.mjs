import eslint from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
    eslint.configs.recommended,
    ...tseslint.configs.recommended,
    {
        ignores: ["dist/**", "node_modules/**", "bindata/**"],
    },
    {
        rules: {
            // Allow declare var for Go backend globals
            "no-var": "off",
            // Prefer const over let
            "prefer-const": "error",
            // Enforce strict equality
            "eqeqeq": ["error", "always"],
            // No unused variables (warn for _ prefixed)
            "@typescript-eslint/no-unused-vars": [
                "error",
                { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
            ],
            // No explicit any unless necessary
            "@typescript-eslint/no-explicit-any": "warn",
            // No unnecessary semicolons on class members
            "no-extra-semi": "error",
        },
    },
);
