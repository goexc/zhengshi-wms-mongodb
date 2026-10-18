module.exports = {
    extends: [
        'stylelint-config-standard-scss', // 包含标准 CSS 和 SCSS 校验
        'stylelint-config-standard-vue/scss', // Vue SFC 使用 postcss-html 解析
        'stylelint-config-recess-order', // 配置stylelint css属性书写顺序插件,
    ],
    // Stylelint 16 移除的大括号间距规则由 Stylistic 接续。
    plugins: ['@stylistic/stylelint-plugin'],
    overrides: [
        {
            files: ['**/*.scss'],
            customSyntax: 'postcss-scss',
        },
        {
            files: ['**/*.{html,vue}'],
            customSyntax: 'postcss-html',
        },
    ],
    ignoreFiles: [
        '**/*.js',
        '**/*.jsx',
        '**/*.tsx',
        '**/*.ts',
        '**/*.json',
        '**/*.md',
        '**/*.yaml',
    ],
    /**
     * null  => 关闭该规则
     * always => 必须
     */
    rules: {
        'value-keyword-case': null, // 在 css 中使用 v-bind，不报错
        'no-descending-specificity': null, // 禁止在具有较高优先级的选择器后出现被其覆盖的较低优先级的选择器
        'function-url-quotes': 'always', // URL 必须加引号
        'no-empty-source': null, // 关闭禁止空源码
        'selector-class-pattern': null, // 关闭强制选择器类名的格式
        'property-no-unknown': null, // 禁止未知的属性(true 为不允许)
        '@stylistic/block-opening-brace-space-before': 'always', // 大括号之前必须有一个空格
        'value-no-vendor-prefix': null, // 关闭 属性值前缀 --webkit-box
        'property-no-vendor-prefix': null, // 关闭 属性前缀 -webkit-mask
        'selector-pseudo-class-no-unknown': [
            // 不允许未知的选择器
            true,
            {
                ignorePseudoClasses: ['global', 'v-deep', 'deep'], // 忽略属性，修改element默认样式的时候能使用到
            },
        ],
    },
}
