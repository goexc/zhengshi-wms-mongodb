# ReadMe

## 1.生成template模板
```bash
goctl template init
```
将生成的模板复制到项目根目录，重命名为`template`

## 1.生成api代码
```bash
goctl api go -api main.api -style goZero -home ./template -dir .
```


## 2.生成MongoDB代码
```bash
goctl model mongo --type User -home ./template --dir ./model --cache
```

## 3.系统初始化

- 1.添加角色：`超级系统管理员`
- 2.添加部门：`某某公司`
- 3.添加用户：`超级管理员`