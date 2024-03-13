package svc

import (
	"api/internal/config"
	"api/model"
	"context"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/casbin/casbin/v2"
	redisadapter "github.com/casbin/redis-adapter/v3"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/cos-go-sdk-v5/debug"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/http"
	"net/url"
	"time"
)

type ServiceContext struct {
	Config                     config.Config
	Cache                      cache.Cache
	Enforcer                   *casbin.SyncedEnforcer
	OSS                        *oss.Client
	Cos                        *cos.Client
	DBClient                   *mongo.Client
	SystemInitModel            *mongo.Collection
	ImageModel                 *mongo.Collection
	CompanyModel               *mongo.Collection
	UserModel                  *mongo.Collection
	ApiModel                   *mongo.Collection
	MenuModel                  *mongo.Collection
	DepartmentModel            *mongo.Collection
	RoleModel                  *mongo.Collection
	RoleMenuModel              *mongo.Collection
	CarrierModel               *mongo.Collection
	SupplierModel              *mongo.Collection
	CustomerModel              *mongo.Collection
	WarehouseModel             *mongo.Collection //仓库
	WarehouseZoneModel         *mongo.Collection //库区
	WarehouseRackModel         *mongo.Collection //货架
	WarehouseBinModel          *mongo.Collection //货位
	MaterialCategoryModel      *mongo.Collection //物料分类表
	MaterialModel              *mongo.Collection //物料表
	MaterialPriceModel         *mongo.Collection //物料价格表
	InboundReceiptModel        *mongo.Collection //入库单
	OutboundOrderModel         *mongo.Collection //出库单
	InboundReceiptReceiveModel *mongo.Collection //入库单批次入库记录
	CustomerTransactionModel   *mongo.Collection //客户流水
	SupplierTransactionModel   *mongo.Collection //供应商流水
	CarrierTransactionModel    *mongo.Collection //承运商流水
	InventoryModel             *mongo.Collection //库存
	OutboundMaterialModel      *mongo.Collection //出库单批次出库记录
}

func NewServiceContext(c config.Config) *ServiceContext {
	//1.MongoDB初始化
	//db := InitMongoDB(c)
	mongodbClient := InitMongoDB(c)
	db := mongodbClient.Database(c.MongoDB.Database)

	ctx := &ServiceContext{
		Config:                     c,
		Cache:                      cache.New([]cache.NodeConf{c.CacheRedis}, syncx.NewSingleFlight(), cache.NewStat(""), mongo.ErrNoDocuments),
		Enforcer:                   InitCasbin(c),
		OSS:                        InitOSS(c),
		Cos:                        InitCos(c),
		DBClient:                   mongodbClient,
		SystemInitModel:            db.Collection("system_init"),
		ImageModel:                 db.Collection("image"),
		CompanyModel:               db.Collection("company"),
		UserModel:                  db.Collection("user"),
		ApiModel:                   db.Collection("api"),
		MenuModel:                  db.Collection("menu"),
		DepartmentModel:            db.Collection("department"),
		RoleModel:                  db.Collection("role"),
		RoleMenuModel:              db.Collection("role_menu"),
		SupplierModel:              db.Collection("supplier"),
		CustomerModel:              db.Collection("customer"),
		CarrierModel:               db.Collection("carrier"),
		WarehouseModel:             db.Collection("warehouse"),               //仓库
		WarehouseZoneModel:         db.Collection("warehouse_zone"),          //库区
		WarehouseRackModel:         db.Collection("warehouse_rack"),          //货架
		WarehouseBinModel:          db.Collection("warehouse_bin"),           //货位
		MaterialCategoryModel:      db.Collection("material_category"),       //物料分类表
		MaterialModel:              db.Collection("material"),                //物料表
		MaterialPriceModel:         db.Collection("material_price"),          //物料价格表
		InboundReceiptModel:        db.Collection("inbound_receipt"),         //入库单
		OutboundOrderModel:         db.Collection("outbound_order"),          //出库单
		InboundReceiptReceiveModel: db.Collection("inbound_receipt_receive"), //入库单批次入库记录
		CustomerTransactionModel:   db.Collection("customer_transaction"),    //客户流水
		SupplierTransactionModel:   db.Collection("supplier_transaction"),    //供应商流水
		CarrierTransactionModel:    db.Collection("carrier_transaction"),     //承运商流水
		InventoryModel:             db.Collection("inventory"),               //库存
		OutboundMaterialModel:      db.Collection("outbound_material"),       //出库单物料
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

// func InitMongoDB(c config.Config) *mongo.Database {
func InitMongoDB(c config.Config) *mongo.Client {
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

	// 设置连接超时时间（默认为 30 秒）
	option.SetConnectTimeout(10 * time.Second)
	// 设置 Socket 超时时间（默认为 30 秒）
	option.SetSocketTimeout(10 * time.Second)

	// 设置操作超时时间（默认为 30 秒）
	option.SetMaxConnIdleTime(1 * time.Minute)

	client, err := mongo.Connect(context.Background(), option)
	if err != nil {
		fmt.Println("MongoDB 连接失败：", err.Error())
		panic(err)
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		fmt.Println("MongoDB Ping失败：", err.Error())
		panic(err)
	}

	fmt.Println("MonogoDB连接成功")

	return client
	//return client.Database(c.MongoDB.Database)
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

	enforcer.EnableLog(false)

	//2.3 casbin auto load
	enforcer.StartAutoLoadPolicy(time.Minute)

	return enforcer
}

func InitOSS(c config.Config) *oss.Client {
	client, err := oss.New(c.OSS.EndPoint, c.OSS.AccessKeyID, c.OSS.AccessKeySecret)
	if err != nil {
		panic("oss初始化失败:" + err.Error())
	}

	return client
}

func InitCos(c config.Config) *cos.Client {
	u, _ := url.Parse(c.COS.EndPoint)
	baseUrl := &cos.BaseURL{
		BucketURL: u,
	}

	return cos.NewClient(baseUrl, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.COS.SecretID,
			SecretKey: c.COS.SecretKey,
			Transport: &debug.DebugRequestTransport{
				RequestHeader:  true,
				RequestBody:    true,
				ResponseHeader: true,
				ResponseBody:   true,
			},
		},
	})
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
