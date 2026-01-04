import js from "@eslint/js";
import pluginVue from "eslint-plugin-vue";

export default [
    js.configs.recommended,
    ...pluginVue.configs["flat/recommended"],
    {
        files: ["**/*.{js,vue}"],
        languageOptions: {
            ecmaVersion: 2022,
            sourceType: "module",
            globals: {
                window: "readonly",
                document: "readonly",
                console: "readonly",
                localStorage: "readonly",
                setTimeout: "readonly",
                setInterval: "readonly",
                clearTimeout: "readonly",
                clearInterval: "readonly",
                fetch: "readonly",
                Date: "readonly",
                JSON: "readonly",
                Promise: "readonly",
                Array: "readonly",
                Object: "readonly",
                Math: "readonly",
                Error: "readonly",
                process: "readonly",
            },
        },
        rules: {
            // Vue 規則
            "vue/multi-word-component-names": "off",
            "vue/no-unused-vars": "warn",
            "vue/require-default-prop": "off",
            "vue/max-attributes-per-line": "off",
            "vue/singleline-html-element-content-newline": "off",
            "vue/html-self-closing": ["warn", {
                html: { void: "always", normal: "never", component: "always" },
                svg: "always",
                math: "always",
            }],

            // JS 規則
            "no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],
            "no-console": "off",
            "prefer-const": "warn",
            "no-var": "error",
            "eqeqeq": ["warn", "smart"],
        },
    },
    {
        ignores: ["node_modules/**", "dist/**", "*.config.js"],
    },
];
