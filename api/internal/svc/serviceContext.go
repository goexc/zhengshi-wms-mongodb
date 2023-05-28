package svc

import (
	"api/internal/config"
	"context"
	"fmt"
	"github.com/casbin/casbin/v2"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type ServiceContext struct {
	Config          config.Config
	Cache           cache.Cache
	Enforcer        *casbin.SyncedEnforcer
	UserModel       *mongo.Collection
	ApiModel        *mongo.Collection
	MenuModel       *mongo.Collection
	DepartmentModel *mongo.Collection
	RoleModel       *mongo.Collection
	RoleMenuModel   *mongo.Collection
}

func NewServiceContext(c config.Config) *ServiceContext {
	//1.MongoDB初始化
	db := InitMongoDB(c)

	ctx := &ServiceContext{
		Config:          c,
		Cache:           cache.New(c.CacheRedis, syncx.NewSingleFlight(), cache.NewStat(""), mongo.ErrNoDocuments),
		Enforcer:        InitCasbin(c),
		UserModel:       db.Collection("user"),
		ApiModel:        db.Collection("api"),
		MenuModel:       db.Collection("menu"),
		DepartmentModel: db.Collection("department"),
		RoleModel:       db.Collection("role"),
		RoleMenuModel:   db.Collection("role_menu"),
	}

	//2.角色表添加索引
	var indexModel = mongo.IndexModel{
		Keys:    bson.M{"name": 1}, // 指定要索引的字段和排序顺序
		Options: nil,
	}
	var indexOpt = options.CreateIndexes().SetMaxTime(5 * time.Second)
	_, err := ctx.RoleModel.Indexes().CreateOne(context.Background(), indexModel, indexOpt)
	if err != nil {
		panic(fmt.Sprintf("[Error]创建角色索引:%s", err.Error()))
	}

	return ctx
}

func InitMongoDB(c config.Config) *mongo.Database {
	//option
	option := options.Client().ApplyURI(c.MongoDB.URI)
	option.SetAuth(options.Credential{
		AuthMechanism:           "",
		AuthMechanismProperties: nil,
		AuthSource:              "",
		Username:                c.MongoDB.UserName,
		Password:                c.MongoDB.Password,
		PasswordSet:             true,
	})

	client, err := mongo.Connect(context.Background(), option)
	if err != nil {
		fmt.Println("连接失败：", err.Error())
		panic(err)
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		fmt.Println("Ping失败：", err.Error())
		panic(err)
	}

	fmt.Println("MonogoDB连接成功")

	return client.Database(c.MongoDB.Database)
}

func InitCasbin(c config.Config) *casbin.SyncedEnforcer {

	//2.casbin初始化
	//2.1 casbin 数据库连接
	//adapter, err := gormadapter.NewAdapter("mysql", c.Casbin.MySQL)
	adapter, err := redisadapter.NewAdapterWithPassword("tcp", c.Casbin.Redis, c.Casbin.Pass)
	if err != nil {
		panic(fmt.Sprintf("casbin数据库连接：%s", err.Error()))
	}
	//2.2 casbin model
	enforcer, err := casbin.NewSyncedEnforcer(c.Casbin.Conf, adapter)
	if err != nil {
		panic("enforcer初始化失败:" + err.Error())
	}

	enforcer.EnableLog(true)

	//2.3 casbin auto load
	enforcer.StartAutoLoadPolicy(time.Minute)

	return enforcer
}
