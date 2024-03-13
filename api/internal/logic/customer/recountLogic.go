package customer

import (
	"api/model"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecountLogic {
	return &RecountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecountLogic) Recount() (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.初始化客户应收账款
	var balances = make(map[string]decimal.Decimal)

	//TODO: START
	//1.查询客户列表
	cur, err := l.svcCtx.CustomerModel.Find(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询客户列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var customers []model.Customer
	if err = cur.All(l.ctx, &customers); err != nil {
		fmt.Printf("[Error]解析客户列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	//2.检索出库单列表
	cur, err = l.svcCtx.OutboundOrderModel.Find(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询订单列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var orders []model.OutboundOrder
	if err = cur.All(l.ctx, &orders); err != nil {
		fmt.Printf("[Error]解析订单列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	//3.将出库单金额信息，更新到客户交易流水表
	// 3.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)

	for _, order := range orders {
		filter := bson.D{{"order_code", order.Code}}
		update := bson.D{
			{"$set", bson.D{
				{"code", fmt.Sprintf("CT-%s-%d", time.Now().Format("2006-01-02-15-04-05"), time.Now().UnixMilli()%1000)}, //交易编号:YYYY-MM-DD-HH-mm-ss-SSS
				{"order_code", order.Code},
				{"type", "应收账款"},
				{"customer_id", order.CustomerId},
				{"customer_name", order.CustomerName},
				{"amount", order.TotalAmount},
				{"remark", ""},
				{"annex", order.Annex},
				{"time", order.ReceiptTime},
				{"creator", l.ctx.Value("uid").(string)},
				{"creator_name", l.ctx.Value("name").(string)},
				{"editor", ""},
				{"editor_name", ""},
				{"created_at", time.Now().Unix()},
				{"updated_at", 0},
			}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpsert(true)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//3.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	_, err = l.svcCtx.CustomerTransactionModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新客户交易流水：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	//4.统计所有客户交易流水，更新到客户表
	//4.1查询所有客户交易流水
	cur, err = l.svcCtx.CustomerTransactionModel.Find(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]查询客户交易流水:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var records []model.CustomerTransaction
	if err = cur.All(l.ctx, &records); err != nil {
		fmt.Printf("[Error]解析客户交易流水:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//for _, customer := range customers {
	//	balances[customer.Id.Hex()] = decimal.NewFromFloat(0)
	//}
	//4.2 累计客户应收账款
	for _, record := range records {
		switch record.Type {
		case "应收账款":
			balances[record.CustomerId] = balances[record.CustomerId].Add(decimal.NewFromFloat(record.Amount))
		case "回款", "退货":
			balances[record.CustomerId] = balances[record.CustomerId].Sub(decimal.NewFromFloat(record.Amount))
		default:
			fmt.Printf("[Error]未知类型[%s]交易流水[%s]\n", record.Type, record.Id.Hex())
			resp.Code = http.StatusBadRequest
			resp.Msg = "存在未知类型的交易流水"
			return resp, nil
		}
	}

	//4.3 批量更新客户应收账款
	// 4.3.1 构建批量更新的过滤条件
	bulkWrites = make([]mongo.WriteModel, 0)

	for key, balance := range balances {
		customerId, _ := primitive.ObjectIDFromHex(key)
		filter := bson.D{{"_id", customerId}}
		update := bson.D{
			{"$set", bson.D{
				{"receivable_balance", balance.InexactFloat64()},
			}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(filter)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//4.3.2 执行批量更新操作
	bulkOptions = options.BulkWriteOptions{}
	_, err = l.svcCtx.CustomerModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新客户应收账款：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	//TODO: END

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
