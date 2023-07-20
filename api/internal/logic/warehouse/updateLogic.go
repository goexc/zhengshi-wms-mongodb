package warehouse

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.WarehouseRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.仓库是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]仓库[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "仓库参数错误"
		return resp, nil
	}
	//排除已删除的仓库
	filter := bson.M{
		"_id":    id,
		"status": bson.M{"$ne": code.WarehouseStatusCode("删除")},
	}
	warehouseRes := l.svcCtx.WarehouseModel.FindOne(l.ctx, filter)
	switch warehouseRes.Err() {
	case nil:
		var warehouse model.Warehouse
		if err = warehouseRes.Decode(&warehouse); err != nil {
			fmt.Printf("[Error]解析仓库[%s]:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		//仓库是否在激活状态
		switch warehouse.Status {
		case 10: //激活
		default: //非激活状态不能执行库存管理和操作
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("仓库%s，无法执行操作", code.WarehouseStatusText(warehouse.Status))
			return resp, nil
		}
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "仓库不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询仓库[%s]：%s\n", req.Id, warehouseRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.仓库名称是否重复
	filter = bson.M{
		"_id":    bson.M{"$ne": id},
		"status": bson.M{"$ne": code.WarehouseStatusCode("删除")},
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
	}
	singleRes := l.svcCtx.WarehouseModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Warehouse
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复仓库:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "仓库名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "仓库编号已占用"
		default:
			resp.Msg = "仓库未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //仓库未占用
	default:
		fmt.Printf("[Error]查询重复仓库:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.更新仓库信息:不更新仓库状态
	var update = bson.M{
		"$set": bson.M{
			"type":          code.WarehouseTypeCode(req.Type),
			"name":          strings.TrimSpace(req.Name),
			"code":          strings.TrimSpace(req.Code),
			"image":         strings.TrimSpace(req.Image),
			"address":       strings.TrimSpace(req.Address),
			"capacity":      req.Capacity,
			"capacity_unit": strings.TrimSpace(req.CapacityUnit),
			"contact":       strings.TrimSpace(req.Contact),
			"manager":       strings.TrimSpace(req.Manager),
			"remark":        strings.TrimSpace(req.Remark),
			"updated_at":    time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.WarehouseModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新仓库[%s]信息：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
