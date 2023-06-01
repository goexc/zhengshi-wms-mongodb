package supplier

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (l *UpdateLogic) Update(req *types.SupplierRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//todo:用于操作记录
	//uid := l.ctx.Value("uid").(string)
	//uObjectID, err := primitive.ObjectIDFromHex(uid)
	//if err != nil {
	//	fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
	//	resp.Code = http.StatusBadRequest
	//	resp.Msg = "参数错误"
	//	return resp, nil
	//}
	//1.供应商是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]供应商[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "供应商参数错误"
		return resp, nil
	}
	//排除已删除的供应商
	filter := bson.M{"_id": id, "status": bson.M{"$ne": 100}}
	count, err := l.svcCtx.SupplierModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询供应商[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "供应商不存在"
		return resp, nil
	}

	//2.供应商名称是否重复
	filter = bson.M{
		"_id":  bson.M{"$ne": id},
		"name": strings.TrimSpace(req.Name),
	}
	count, err = l.svcCtx.SupplierModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询供应商名称[%s]是否占用:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "供应商已存在"
		return resp, nil
	}

	//3.更新供应商信息
	var update = bson.M{
		"$set": bson.M{
			"name":       strings.TrimSpace(req.Name),
			"address":    req.Address,
			"contact":    req.Contact,
			"manager":    req.Manager,
			"level":      req.Level,
			"remark":     req.Remark,
			"updated_at": time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.SupplierModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新供应商[%s]信息：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
