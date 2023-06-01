package supplier

import (
	"api/model"
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

func (l *AddLogic) Add(req *types.SupplierRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}
	//1.供应商是否存在
	var name = strings.TrimSpace(req.Name)
	//i 表示不区分大小写
	filter := bson.M{"name": name, "status": bson.M{"$ne": 100}}
	count, err := l.svcCtx.SupplierModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询供应商[%s]是否存在:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "供应商已存在"
		return resp, nil
	}

	//2.添加供应商
	var supplier = model.Supplier{
		Name:      strings.TrimSpace(req.Name),
		Address:   strings.TrimSpace(req.Address),
		Contact:   strings.TrimSpace(req.Contact),
		Manager:   strings.TrimSpace(req.Manager),
		Level:     req.Level,
		Status:    10, //默认启用
		Remark:    strings.TrimSpace(req.Remark),
		Creator:   uObjectID,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_, err = l.svcCtx.SupplierModel.InsertOne(l.ctx, &supplier)
	if err != nil {
		fmt.Printf("[Error]供应商[%s]入库:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
