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

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.WarehouseRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//1.仓库是否存在：仓库名称、仓库编号
	//i 表示不区分大小写
	filter := bson.M{
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
		},
		"status": bson.M{"$ne": 100},
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

	//2.添加仓库
	var warehouse = model.Warehouse{
		Type:         code.WarehouseTypeCode(req.Type),
		Name:         strings.TrimSpace(req.Name),
		Code:         strings.TrimSpace(req.Code),
		Address:      strings.TrimSpace(req.Address),
		Capacity:     req.Capacity,
		CapacityUnit: strings.TrimSpace(req.CapacityUnit),
		Status:       code.WarehouseStatusCode("激活"),
		Manager:      strings.TrimSpace(req.Manager),
		Contact:      strings.TrimSpace(req.Contact),
		Remark:       strings.TrimSpace(req.Remark),
		Creator:      uObjectID,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	_, err = l.svcCtx.WarehouseModel.InsertOne(l.ctx, &warehouse)
	if err != nil {
		fmt.Printf("[Error]新增仓库[%s]:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
