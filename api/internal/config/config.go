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
}
