package svc

import (
	"api/internal/config"
	"api/model"
	"context"
	"fmt"
	"github.com/casbin/casbin/v2"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type ServiceContext struct {
	Config          config.Config
	Cache           cache.Cache
	Enforcer        *casbin.SyncedEnforcer
	SystemInitModel *mongo.Collection
	CompanyModel    *mongo.Collection
	UserModel       *mongo.Collection
	ApiModel        *mongo.Collection
	MenuModel       *mongo.Collection
	DepartmentModel *mongo.Collection
	RoleModel       *mongo.Collection
	RoleMenuModel   *mongo.Collection
	SupplierModel   *mongo.Collection
}

func NewServiceContext(c config.Config) *ServiceContext {
	//1.MongoDB初始化
	db := InitMongoDB(c)

	ctx := &ServiceContext{
		Config:          c,
		Cache:           cache.New(c.CacheRedis, syncx.NewSingleFlight(), cache.NewStat(""), mongo.ErrNoDocuments),
		Enforcer:        InitCasbin(c),
		SystemInitModel: db.Collection("system_init"),
		CompanyModel:    db.Collection("company"),
		UserModel:       db.Collection("user"),
		ApiModel:        db.Collection("api"),
		MenuModel:       db.Collection("menu"),
		DepartmentModel: db.Collection("department"),
		RoleModel:       db.Collection("role"),
		RoleMenuModel:   db.Collection("role_menu"),
		SupplierModel:   db.Collection("supplier"),
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

	//3.系统初始化步骤
	SystemInit(ctx)

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

func SystemInit(ctx *ServiceContext) {
	id, err := primitive.ObjectIDFromHex(ctx.Config.Ids.SystemInit)
	if err != nil {
		panic(fmt.Sprintf("[Error]解析系统初始化步骤id:%s\n", err.Error()))
	}

	var sysi model.SystemInit
	res := ctx.SystemInitModel.FindOne(context.Background(), bson.M{"_id": id})
	switch res.Err() {
	case nil:
		if e := res.Decode(&sysi); e != nil {
			panic(fmt.Sprintf("[Error]解析系统初始化数据:%s\n", res.Err().Error()))
		}
	case mongo.ErrNoDocuments: //没有数据
		_, e := ctx.SystemInitModel.InsertOne(context.Background(), bson.D{
			{"_id", id},
			{"step", 0},
		})
		if e != nil {
			panic(fmt.Sprintf("[Error]系统初始化步骤写入:%s\n", e.Error()))
		}

		sysi.Id = id
		sysi.Step = 0
	default:
		panic(fmt.Sprintf("[Error]查询系统初始化步骤:%s\n", res.Err().Error()))
	}

	if sysi.Step == 0 {
		//todo:写入api列表[先清空，后写入]
		//todo:写入菜单列表[先清空，后写入]
		//修改系统初始化步骤
		//singleRes := ctx.SystemInitModel.FindOneAndUpdate(context.Background(), bson.M{"_id": id}, bson.M{"step": 1})
		//if singleRes.Err()!=nil{
		//	panic(fmt.Sprintf("[Error]系统初始化步骤gengxin[0->1]:%s", singleRes.Err().Error()))
		//}
	}
}
