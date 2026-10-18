import js from '@eslint/js'
import { tsPlugin, tsParser } from '@zhengshi/vue-typescript/eslint'
import { defineConfig } from 'eslint/config'
import prettierRecommended from 'eslint-plugin-prettier/recommended'
import vue from 'eslint-plugin-vue'
import globals from 'globals'

// Vue SFC scripts use TypeScript, so apply its core-rule compatibility preset
// there too (for example, TypeScript checks names such as DOM's BlobPart).
const typescriptRecommended = tsPlugin.configs['flat/recommended'].map((config) =>
    config.name === 'typescript-eslint/eslint-recommended'
        ? { ...config, files: [...config.files, '**/*.vue'] }
        : config,
)

export default defineConfig([
    { ignores: ['**/dist/**', '**/node_modules/**'] },
    {
        files: ['**/*.{js,cjs,mjs,jsx,ts,tsx,vue}'],
        extends: [
            js.configs.recommended,
            typescriptRecommended,
            vue.configs['flat/essential'],
            prettierRecommended,
        ],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
                ...globals.node,
                ...globals.jest,
            },
            parserOptions: {
                parser: tsParser,
                // These SFCs use <Type> assertions, which JSX would misparse.
                ecmaFeatures: { jsx: false },
            },
        },
        rules: {
            'no-var': 'error',
            'no-multiple-empty-lines': ['warn', { max: 1 }],
            'no-console': process.env.NODE_ENV === 'production' ? 'error' : 'off',
            'no-debugger': process.env.NODE_ENV === 'production' ? 'error' : 'off',
            'no-unexpected-multiline': 'error',
            'no-useless-escape': 'off',

            '@typescript-eslint/no-unused-vars': 'error',
            // Replaces prefer-ts-expect-error and keeps @ts-ignore prohibited.
            '@typescript-eslint/ban-ts-comment': 'error',
            '@typescript-eslint/no-explicit-any': 'off',
            '@typescript-eslint/no-non-null-assertion': 'off',
            '@typescript-eslint/no-namespace': 'off',

            'vue/multi-word-component-names': 'off',
            // vue-eslint-parser now tracks template uses of script-setup bindings.
            'vue/no-mutating-props': 'off',
            'vue/attribute-hyphenation': 'off',
        },
    },
    {
        files: ['**/*.{jsx,tsx}'],
        languageOptions: {
            parserOptions: { ecmaFeatures: { jsx: true } },
        },
    },
])
