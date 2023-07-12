package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	CacheRedis cache.CacheConf
	Auth       struct {
		AccessSecret string
		AccessExpire int64
	}
	Salt   string
	Avatar string
	Casbin struct {
		Redis string
		Pass  string
		Conf  string
	}
	MongoDB struct {
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
	OSS struct {
		EndPoint        string
		AccessKeyID     string
		AccessKeySecret string
		Bucket          string
		Domain          string
	}
}
