package outbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReviseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviseLogic {
	return &ReviseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReviseLogic) Revise(req *types.OutboundOrderReviseRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.出库单是否存在
	var filter = bson.M{"code": req.Code}
	var order model.OutboundOrder
	singleRes := l.svcCtx.OutboundOrderModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&order); err != nil {
			fmt.Printf("[Error]解析出库单:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //出库单不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询出库单[%s]:%s\n", req.Code, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	fmt.Println("出库单存在")

	//2.客户id是否一致
	if strings.TrimSpace(req.CustomerId) != order.CustomerId {
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单与客户不匹配"
		return resp, nil
	}

	//3.出库单物料是否匹配
	filter = bson.M{"order_code": req.Code}
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]图片分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var materials = make([]model.OutboundOrderMaterial, 0)
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单物料列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var materialsMap = make(map[string]model.OutboundOrderMaterial)
	for _, one := range materials {
		materialsMap[one.MaterialId] = one
	}

	//3.1 出库单物料是否匹配
	for _, one := range req.MaterialsPrice {
		if _, ok := materialsMap[one.MaterialId]; !ok {
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("缺少部分物料的价格")
			return resp, nil
		}
	}

	//3.2 是否存在不相关物料
	if len(materials) < len(req.MaterialsPrice) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请勿提供多余物料"
		return resp, nil
	}

	//4.更新出库单物料单价
	// 4.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)

	for _, one := range req.MaterialsPrice {
		ft := bson.D{
			{"order_code", req.Code},
			{"material_id", one.MaterialId},
		}
		update := bson.D{
			{"$set", bson.D{
				{"price", one.Price},
			}},
		}

		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(ft)
		bulkWrite.SetUpdate(update)

		bulkWrites = append(bulkWrites, bulkWrite)
	}

	//4.2 执行批量更新操作
	bulkOptions := options.BulkWriteOptions{}
	_, err = l.svcCtx.OutboundMaterialModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
	if err != nil {
		fmt.Printf("[Error]批量更新出库单[%s]物料单价：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.添加/更新物料的单价(忽略<=0的数据)
	for _, one := range req.MaterialsPrice {
		if one.Price <= 0 { //忽略无效的物料单价
			continue
		}

		update := bson.M{
			"$set": bson.M{
				"material":      one.MaterialId,
				"customer_id":   req.CustomerId,
				"customer_name": order.CustomerName,
				"price":         one.Price,
				"creator":       l.ctx.Value("uid").(string),
				"creator_name":  l.ctx.Value("name").(string),
				"created_at":    time.Now().Unix(),
			},
		}
		var opts = options.Update().SetUpsert(true)

		_, err = l.svcCtx.MaterialPriceModel.UpdateOne(l.ctx,
			bson.M{"material": one.MaterialId, "customer_id": req.CustomerId, "price": one.Price},
			update,
			opts)
		if err != nil {
			fmt.Printf("[Error]存储客户[%s]物料[%s][%s]单价:%s\n", req.CustomerId, one.MaterialId, materialsMap[one.MaterialId].Model, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	//6.重新计算出库单总金额
	var amount decimal.Decimal
	amount = decimal.NewFromFloat(order.CarrierCost).Add(decimal.NewFromFloat(order.OtherCost))

	for _, one := range req.MaterialsPrice {
		amount = decimal.NewFromFloat(materialsMap[one.MaterialId].Quantity).Mul(decimal.NewFromFloat(one.Price)).Add(amount)
	}

	fmt.Println("总金额：", amount.InexactFloat64())
	//6.1 记录出库单原先总金额
	prevAmount := order.TotalAmount

	//6.2 更新
	update := bson.M{
		"$set": bson.M{
			"total_amount": amount.InexactFloat64(),
			"updated_at":   time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.OutboundOrderModel.UpdateByID(l.ctx, order.Id, update)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]总金额:%s\n", order.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//7.修改交易流水金额
	filter = bson.M{"order_code": order.Code}
	update = bson.M{
		"$set": bson.M{
			"amount": amount.InexactFloat64(),
		},
	}
	_, err = l.svcCtx.CustomerTransactionModel.UpdateOne(l.ctx, filter, &update)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]交易流水总金额[%f]：%s\n", req.Code, amount.InexactFloat64(), err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//8.修改应收账款金额
	customerId, _ := primitive.ObjectIDFromHex(order.CustomerId)
	update = bson.M{
		"$inc": bson.M{
			"receivable_balance": decimal.NewFromFloat(-prevAmount).Add(amount).InexactFloat64(),
		},
	}
	_, err = l.svcCtx.CustomerModel.UpdateByID(l.ctx, customerId, &update)
	if err != nil {
		fmt.Printf("[Error]更新客户[%s]应收账款：%s\n", order.CustomerName, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
