package svc

import (
	"backend/config"
	"backend/model_mysql"
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)
import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServiceContext struct {
	Config                     config.Config
	Cache                      cache.Cache
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
	//旧系统物料
	ProductModel model_mysql.ProductModel
	//旧系统物料单价
	ProductPriceModel model_mysql.ProductPriceModel
	//旧系统供应商
	OldSupplierModel model_mysql.SupplierModel
	//旧系统客户
	OldCustomerModel model_mysql.ClientModel
	//旧系统出库单
	OldStockOutModel model_mysql.StockOutModel
	//旧系统出库单详情
	OldStockOutDetailModel model_mysql.StockOutDetailModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	//1.MongoDB初始化
	db := InitMongoDB(c)

	ctx := &ServiceContext{
		Config:                     c,
		Cache:                      cache.New([]cache.NodeConf{c.CacheRedis}, syncx.NewSingleFlight(), cache.NewStat(""), mongo.ErrNoDocuments),
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
		//旧系统物料
		ProductModel: model_mysql.NewProductModel(sqlx.NewMysql(c.MySQL), c.RedisCache),

		//旧系统物料单价
		ProductPriceModel: model_mysql.NewProductPriceModel(sqlx.NewMysql(c.MySQL), c.RedisCache),

		//旧系统供应商
		OldSupplierModel: model_mysql.NewSupplierModel(sqlx.NewMysql(c.MySQL), c.RedisCache),
		//旧系统客户
		OldCustomerModel: model_mysql.NewClientModel(sqlx.NewMysql(c.MySQL), c.RedisCache),
		//旧系统出库单
		OldStockOutModel: model_mysql.NewStockOutModel(sqlx.NewMysql(c.MySQL), c.RedisCache),
		//旧系统出库单详情
		OldStockOutDetailModel: model_mysql.NewStockOutDetailModel(sqlx.NewMysql(c.MySQL), c.RedisCache),
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

	// 设置连接超时时间（默认为 30 秒）
	option.SetConnectTimeout(10 * time.Second)
	// 设置 Socket 超时时间（默认为 30 秒）
	option.SetSocketTimeout(10 * time.Second)

	// 设置操作超时时间（默认为 30 秒）
	option.SetMaxConnIdleTime(1 * time.Minute)

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
