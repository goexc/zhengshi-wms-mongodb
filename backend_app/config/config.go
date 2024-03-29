package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	CacheRedis cache.NodeConf
	//旧系统数据库
	MySQL string
	//旧系统缓存
	RedisCache cache.CacheConf
	Auth       struct {
		AccessSecret string
		AccessExpire int64
	}
	Uid      string //操作员id
	UserName string //操作员
	Salt     string
	Avatar   string
	MongoDB  struct {
		URI      string
		UserName string
		Password string
		Database string
	}
	Ids struct {
		SystemInit string //系统初始化步骤id
		Company    string //企业id
		Department string //顶级部门：企业id
		Role       string //系统管理角色id
		User       string //系统管理员
	}
	Collections struct {
		SystemInit string
		Company    string
		User       string
		Api        string
		Menu       string
		Department string
		Role       string
		RoleMenu   string
		Supplier   string
	}
}
