package plan

import (
	"api/model"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
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

func (l *AddLogic) Add(req *types.PlanAddRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.物料是否存在
	materialId, _ := primitive.ObjectIDFromHex(req.MaterialId)
	singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, bson.M{"_id": materialId})
	var material model.Material
	switch err = singleRes.Err(); {
	case err == nil: //物料存在
		if err = singleRes.Decode(&material); err != nil {
			fmt.Printf("[Error]解析物料信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	case errors.Is(err, mongo.ErrNoDocuments): //物料不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "物料不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询物料[%s]:%s\n", req.MaterialId, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.采购计划类型中，供应商是否存在
	var supplier model.Supplier
	if req.Type == "采购计划" {
		supplierId, _ := primitive.ObjectIDFromHex(req.SupplierId)
		singleRes = l.svcCtx.SupplierModel.FindOne(l.ctx, bson.M{"_id": supplierId})

		switch err = singleRes.Err(); {
		case err == nil: //供应商存在
			if err = singleRes.Decode(&supplier); err != nil {
				fmt.Printf("[Error]解析供应商信息:%s\n", err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		case errors.Is(err, mongo.ErrNoDocuments): //供应商不存在
			resp.Code = http.StatusBadRequest
			resp.Msg = "供应商不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]查询供应商[%s]:%s\n", req.SupplierId, singleRes.Err().Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}
	//3.生产计划类型中，客户是否存在
	var customer model.Customer
	if req.Type == "生产计划" {
		customerId, _ := primitive.ObjectIDFromHex(req.CustomerId)
		singleRes = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId})

		switch err = singleRes.Err(); {
		case err == nil: //客户存在
			if err = singleRes.Decode(&customer); err != nil {
				fmt.Printf("[Error]解析客户信息:%s\n", err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		case errors.Is(err, mongo.ErrNoDocuments): //客户不存在
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户不存在"
			return resp, nil
		default:
			fmt.Printf("[Error]查询客户[%s]:%s\n", req.SupplierId, singleRes.Err().Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	//4.添加计划
	var plan = model.Plan{
		Id:               primitive.ObjectID{},
		Type:             req.Type,
		Status:           "执行中",
		SupplierId:       req.SupplierId,
		SupplierName:     supplier.Name,
		CustomerId:       req.CustomerId,
		CustomerName:     customer.Name,
		MaterialId:       material.Id.Hex(),
		MaterialName:     material.Name,
		MaterialImage:    material.Image,
		MaterialModel:    material.Model,
		MaterialUnit:     material.Unit,
		MaterialQuantity: req.MaterialQuantity,
		Deadline:         req.Deadline,
		CreatorId:        l.ctx.Value("uid").(string),
		CreatorName:      l.ctx.Value("name").(string),
		CreatedAt:        time.Now().Unix(),
	}
	_, err = l.svcCtx.PlanModel.InsertOne(l.ctx, &plan)
	if err != nil {
		fmt.Printf("[Error]添加计划:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
