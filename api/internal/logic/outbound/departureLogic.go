package outbound

import (
	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type DepartureLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDepartureLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DepartureLogic {
	return &DepartureLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DepartureLogic) Departure(req *types.OutboundOrderDepartureRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.查询出库单
	//1.1 查询出库单
	singleRes := l.svcCtx.OutboundOrderModel.FindOne(l.ctx, bson.M{"code": req.Code})
	switch singleRes.Err() {
	case nil: //出库单存在
	case mongo.ErrNoDocuments: //出库单不存在
		fmt.Printf("[Error]出库单[%s]不存在\n", req.Code)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询出库单[%s]:%s\n", req.Code, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var order model.OutboundOrder
	if err = singleRes.Decode(&order); err != nil {
		fmt.Printf("[Error]解析出库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.2 出库单状态是“已拣货”、“已打包”、“已称重”
	if order.Status != "已拣货" {
		switch order.Status {
		case "预发货", "待拣货":
			resp.Msg = fmt.Sprintf("%s出库单无法出库", order.Status)
			resp.Code = http.StatusBadRequest
			return resp, nil
		case "已拣货", "已打包", "已称重":
		default:
			resp.Msg = "不能重复出库"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
	}

	//2.承运商是否存在
	var carrier model.Carrier
	if len(req.CarrierId) > 0 {
		carrierId, _ := primitive.ObjectIDFromHex(req.CarrierId)
		singleRes = l.svcCtx.CarrierModel.FindOne(l.ctx, bson.M{"_id": carrierId})
		switch singleRes.Err() {
		case nil:
			if err = singleRes.Decode(&carrier); err != nil {
				fmt.Printf("[Error]承运商[%s]解析:%s\n", req.CarrierId, err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments:
			fmt.Printf("[Error]承运商[%s]不存在\n", req.CarrierId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "入库单不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]承运商查询：%s\n", singleRes.Err().Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	fmt.Println("承运商：", carrier.Name)

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]出库单[%s]创建出库事务：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]出库单[%s]开启出库事务：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.查询发货单物料，后续用于维护客户首次交付记录
	cur, err := l.svcCtx.OutboundMaterialModel.Find(dbCtx, bson.M{"order_code": strings.TrimSpace(req.Code)})
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]查询出库单[%s]物料列表：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	var materials []model.OutboundOrderMaterial
	if err = cur.All(dbCtx, &materials); err != nil {
		_ = cur.Close(dbCtx)
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]解析出库单[%s]物料列表：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	_ = cur.Close(dbCtx)

	//3.修改发货单状态：“已拣货”、“已打包”、“已称重”->“已出库”
	var update = bson.M{
		"$set": bson.M{
			"status":         "已出库",
			"carrier_id":     req.CarrierId,
			"carrier_name":   carrier.Name,
			"carrier_cost":   req.CarrierCost,
			"other_cost":     req.OtherCost,
			"departure_time": req.DepartureTime,
		},
	}
	updateRes, err := l.svcCtx.OutboundOrderModel.UpdateOne(dbCtx, bson.M{"_id": order.Id, "status": order.Status}, update)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]更新出库单[%s]状态(已出库)：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if updateRes.MatchedCount != 1 {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单状态已变化，请刷新后重试"
		return resp, nil
	}

	deliveredOrder := order
	deliveredOrder.Status = "已出库"
	deliveredOrder.CarrierId = req.CarrierId
	deliveredOrder.CarrierName = carrier.Name
	deliveredOrder.CarrierCost = req.CarrierCost
	deliveredOrder.OtherCost = req.OtherCost
	deliveredOrder.DepartureTime = req.DepartureTime

	if err = materialdelivery.SyncCustomerMaterialDeliveries(dbCtx, l.svcCtx, deliveredOrder, materials); err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]维护出库单[%s]客户物料首次交付记录：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "维护客户物料首次交付记录失败"
		return resp, nil
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		fmt.Printf("[Error]出库单[%s]提交出库事务：%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
