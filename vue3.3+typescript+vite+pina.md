### 1.创建vue3项目（基于vite）

```bash
npm i -g pnpm

pnpm create vite
```

### 2.eslint配置
**eslint中文官网:http://eslint.cn/**

首先安装eslint

```bash
pnpm i eslint -D
```

生成配置文件:.eslint.cjs

```bash
npx eslint --init
```

**.eslint.cjs配置文件**
```
module.exports = {
   //运行环境
    "env": { 
        "browser": true,//浏览器端
        "es2021": true,//es2021
    },
    //规则继承
    "extends": [ 
       //全部规则默认是关闭的,这个配置项开启推荐规则,推荐规则参照文档
       //比如:函数不能重名、对象不能出现重复key
        "eslint:recommended",
        //vue3语法规则
        "plugin:vue/vue3-essential",
        //ts语法规则
        "plugin:@typescript-eslint/recommended"
    ],
    //要为特定类型的文件指定处理器
    "overrides": [
    ],
    //指定解析器:解析器
    //Esprima 默认解析器
    //Babel-ESLint babel解析器
    //@typescript-eslint/parser ts解析器
    "parser": "@typescript-eslint/parser",
    //指定解析器选项
    "parserOptions": {
        "ecmaVersion": "latest",//校验ECMA最新版本
        "sourceType": "module"//设置为"script"（默认），或者"module"代码在ECMAScript模块中
    },
    //ESLint支持使用第三方插件。在使用插件之前，您必须使用npm安装它
    //该eslint-plugin-前缀可以从插件名称被省略
    "plugins": [
        "vue",
        "@typescript-eslint"
    ],
    //eslint规则
    "rules": {
    }
}
```

3. vue3环境代码校验插件
```
# 让所有与prettier规则存在冲突的Eslint rules失效，并使用prettier进行代码检查
"eslint-config-prettier": "^8.6.0",
"eslint-plugin-import": "^2.27.5",
"eslint-plugin-node": "^11.1.0",
# 运行更漂亮的Eslint，使prettier规则优先级更高，Eslint优先级低
"eslint-plugin-prettier": "^4.2.1",
# vue.js的Eslint插件（查找vue语法错误，发现错误指令，查找违规风格指南
"eslint-plugin-vue": "^9.9.0",
# 该解析器允许使用Eslint校验所有babel code
"@babel/eslint-parser": "^7.19.1",
```

安装指令
```bash
pnpm install -D eslint-plugin-import eslint-plugin-vue eslint-plugin-node eslint-plugin-prettier eslint-config-prettier eslint-plugin-node @babel/eslint-parser
```


4. 修改.eslintrc.cjs配置文件

   ```
   // @see https://eslint.bootcss.com/docs/rules/
   
   module.exports = {
     env: {
       browser: true,
       es2021: true,
       node: true,
       jest: true,
     },
     /* 指定如何解析语法 */
     parser: 'vue-eslint-parser',
     /** 优先级低于 parse 的语法解析配置 */
     parserOptions: {
       ecmaVersion: 'latest',
       sourceType: 'module',
       parser: '@typescript-eslint/parser',
       jsxPragma: 'React',
       ecmaFeatures: {
         jsx: true,
       },
     },
     /* 继承已有的规则 */
     extends: [
       'eslint:recommended',
       'plugin:vue/vue3-essential',
       'plugin:@typescript-eslint/recommended',
       'plugin:prettier/recommended',
     ],
     plugins: ['vue', '@typescript-eslint'],
     /*
      * "off" 或 0    ==>  关闭规则
      * "warn" 或 1   ==>  打开的规则作为警告（不影响代码执行）
      * "error" 或 2  ==>  规则作为一个错误（代码不能执行，界面报错）
      */
     rules: {
       // eslint（https://eslint.bootcss.com/docs/rules/）
       'no-var': 'error', // 要求使用 let 或 const 而不是 var
       'no-multiple-empty-lines': ['warn', { max: 1 }], // 不允许多个空行
       'no-console': process.env.NODE_ENV === 'production' ? 'error' : 'off',
       'no-debugger': process.env.NODE_ENV === 'production' ? 'error' : 'off',
       'no-unexpected-multiline': 'error', // 禁止空余的多行
       'no-useless-escape': 'off', // 禁止不必要的转义字符
   
       // typeScript (https://typescript-eslint.io/rules)
       '@typescript-eslint/no-unused-vars': 'error', // 禁止定义未使用的变量
       '@typescript-eslint/prefer-ts-expect-error': 'error', // 禁止使用 @ts-ignore
       '@typescript-eslint/no-explicit-any': 'off', // 禁止使用 any 类型
       '@typescript-eslint/no-non-null-assertion': 'off',
       '@typescript-eslint/no-namespace': 'off', // 禁止使用自定义 TypeScript 模块和命名空间。
       '@typescript-eslint/semi': 'off',
   
       // eslint-plugin-vue (https://eslint.vuejs.org/rules/)
       'vue/multi-word-component-names': 'off', // 要求组件名称始终为 “-” 链接的单词
       'vue/script-setup-uses-vars': 'error', // 防止<script setup>使用的变量<template>被标记为未使用
       'vue/no-mutating-props': 'off', // 不允许组件 prop的改变
       'vue/attribute-hyphenation': 'off', // 对模板中的自定义组件强制执行属性命名样式
     },
   }
   
   ```

   

5. eslintignore忽略文件

   ```
   dist
   node_modules
   ```

   

6. 运行脚本

   package.json新增两个运行脚本

   ```
   "scripts": {
       "lint": "eslint src",
       "fix": "eslint src --fix",
   }
   ```



### 7. 配置**prettier**

有了eslint，为什么还要有prettier？eslint针对的是javascript，他是一个检测工具，包含js语法以及少部分格式问题，在eslint看来，语法对了就能保证代码正常运行，格式问题属于其次；

而prettier属于格式化工具，它看不惯格式不统一，所以它就把eslint没干好的事接着干，另外，prettier支持

包含js在内的多种语言。

总结起来，**eslint和prettier这俩兄弟一个保证js代码质量，一个保证代码美观。**

8. 安装依赖包

   ```
   pnpm install -D eslint-plugin-prettier prettier eslint-config-prettier
   ```

   

9. prettierrc.json添加规则

   ```json
   {
     "singleQuote": true,
     "semi": false,
     "bracketSpacing": true,
     "htmlWhitespaceSensitivity": "ignore",
     "endOfLine": "auto",
     "trailingComma": "all",
     "tabWidth": 2
   }
   ```

   

10. prettierignore忽略文件

    ```
    /dist/*
    /html/*
    .local
    /node_modules/**
    **/*.svg
    **/*.sh
    /public/*
    ```

    **通过pnpm run lint去检测语法，如果出现不规范格式,通过pnpm run fix 修改**

11. 

12. 

13. 




```bash
# 出现问题时执行下面的安装命令
# npm install vite-plugin-vue2 -D

# npm i @vitejs/plugin-vue-jsx -D


# npm install vue-router@4

# npm install pinia@next

# npm install element-plus --save
```

```bash
# 在vite.config.ts配置
imort {createVuePlugin} from 'vite-plugin-vue2'
export default{
plugins: [createVuePlugin(/* options */)]
}
```



```bash
# 安装less
# npm install less less-loader -D
```



```bash
# 占满整个区域
flex: 1
```



```bash
# css: .content, .content-items 
.content {
	flex: 1;
	margin: 20px;
	&-items{
		padding: 20px;
	}
}

```



```bash
# 启用dist目录
# npm install http-server -g
# 在dist目录执行命令
# http-server -p 9002
```

