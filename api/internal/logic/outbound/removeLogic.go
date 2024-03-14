package outbound

import (
	"api/model"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 可删除的出库单状态
//var canDeleteStatus = map[string]string{"预发货": ""}

func (l *RemoveLogic) Remove(req *types.OutboundOrderIdRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//预发货的出库单可以删除

	//1.出库单是否存在
	//1.1 出库单信息
	id, _ := primitive.ObjectIDFromHex(req.Id)
	filter := bson.M{"_id": id}
	var receipt model.OutboundOrder
	singleRes := l.svcCtx.OutboundOrderModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析重复个人:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		//if _, ok := canDeleteStatus[receipt.Status]; !ok {
		//	resp.Code = http.StatusBadRequest
		//	resp.Msg = fmt.Sprintf("无法删除[%s]状态的出库单", receipt.Status)
		//	return resp, nil
		//}

	case mongo.ErrNoDocuments: //出库单不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询出库单[%s]是否存在:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.2 出库单物料信息
	var materials []model.OutboundOrderMaterial
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": receipt.Code})
	if err != nil {
		fmt.Printf("[Error]查询出库单[%s]物料列表:%s\n", receipt.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单[%s]物料列表:%s\n", receipt.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//TODO:2.创建事务
	//2.1 创建会话
	sess, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]删除出库单：创建会话：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer sess.EndSession(l.ctx)

	//2.2 根据会话获取数据库操作的上下文
	dbCtx := mongo.NewSessionContext(l.ctx, sess)

	//2.3 开始事务
	err = sess.StartTransaction()
	if err != nil {
		fmt.Printf("[Error]签收出库单：开始事务：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.删除出库单
	_, err = l.svcCtx.OutboundOrderModel.DeleteOne(dbCtx, filter)
	if err != nil {
		fmt.Printf("[Error]删除出库单[%s]:%s\n", receipt.Code, err.Error())
		// 回滚事务
		sess.AbortTransaction(dbCtx)

		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//3.删除出库单物料
	filter = bson.M{"order_code": strings.TrimSpace(receipt.Code)}
	_, err = l.svcCtx.OutboundMaterialModel.DeleteMany(dbCtx, filter)
	if err != nil {
		fmt.Printf("[Error]删除出库单[%s]物料:%s\n", receipt.Code, err.Error())
		// 回滚事务
		sess.AbortTransaction(dbCtx)

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.库存恢复
	// 4.1 构建批量更新的过滤条件
	var bulkWrites = make([]mongo.WriteModel, 0)

	//遍历物料
	for _, material := range materials {
		//遍历物料取用库存
		for _, inventory := range material.Inventorys {
			inventoryId, _ := primitive.ObjectIDFromHex(inventory.InventoryId)
			ft := bson.D{{"_id", inventoryId}}

			switch receipt.Status {
			case "预发货":
			case "待拣货": //扣减locked_quantity
				update := bson.M{
					"$inc": bson.M{"locked_quantity": -inventory.ShipmentQuantity},
				}

				bulkWrite := mongo.NewUpdateOneModel()
				bulkWrite.SetFilter(ft)
				bulkWrite.SetUpdate(update)
				bulkWrites = append(bulkWrites, bulkWrite)

			case "已拣货", "已打包", "已称重", "已出库", "已签收": //恢复available_quantity
				update := bson.M{
					"$inc": bson.M{"available_quantity": inventory.ShipmentQuantity},
				}

				bulkWrite := mongo.NewUpdateOneModel()
				bulkWrite.SetFilter(ft)
				bulkWrite.SetUpdate(update)
				bulkWrites = append(bulkWrites, bulkWrite)

			default:

			}
		}
	}

	//4.2 执行批量更新操作
	if len(bulkWrites) > 0 {
		bulkOptions := options.BulkWriteOptions{}
		_, err = l.svcCtx.InventoryModel.BulkWrite(l.ctx, bulkWrites, &bulkOptions)
		if err != nil {
			fmt.Printf("[Error]批量更新客户交易流水：%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	//5.删除客户交易流水(订单为签收状态时)
	if receipt.Status == "已签收" {
		singleRes = l.svcCtx.CustomerTransactionModel.FindOneAndDelete(dbCtx, bson.M{"order_code": receipt.Code})
		switch err = singleRes.Err(); {
		case err == nil:
		case errors.Is(err, mongo.ErrNoDocuments):
			fmt.Printf("[Error]客户出库单[%s]交易流水不存在:%s\n", receipt.Code, err.Error())
			// 回滚事务
			sess.AbortTransaction(dbCtx)

			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		default:
			fmt.Printf("[Error]删除客户出库单[%s]交易流水:%s\n", receipt.Code, err.Error())
			// 回滚事务
			sess.AbortTransaction(dbCtx)

			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}
	}

	//6.扣减应收账款(订单为签收状态时)
	if receipt.Status == "已签收" {
		customerId, _ := primitive.ObjectIDFromHex(receipt.CustomerId)
		update := bson.M{
			"$inc": bson.M{
				"receivable_balance": -receipt.TotalAmount,
			},
		}

		_, err = l.svcCtx.CustomerModel.UpdateByID(dbCtx, customerId, &update)
		if err != nil {
			fmt.Printf("[Error]删除出库单[%s],扣减客户[%s]应收账款：%s\n", receipt.Code, receipt.CustomerName, err.Error())
			// 回滚事务
			sess.AbortTransaction(dbCtx)

			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}
	}

	//6.提交事务
	err = sess.CommitTransaction(dbCtx)
	if err != nil {
		fmt.Printf("[Error]删除出库单[%s]，事务提交失败: %s\n", receipt.Code, err.Error())

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
