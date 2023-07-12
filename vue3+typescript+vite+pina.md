1.创建vue3项目（基于vite）

```bash
# npm init vite@latest
# 或
# npm init vue@latest
# yarn config set registry http://registry.npm.taobao.org/
```

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

