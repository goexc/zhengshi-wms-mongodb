package images

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

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

func (l *RemoveLogic) Remove(req *types.ImageRemoveRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	key := strings.TrimSpace(req.Key)

	var filter = bson.M{"object_key": key}

	//1.数据库中是否存在
	var one model.Image
	singleRes := l.svcCtx.ImageModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //图片存在
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析图片信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //图片不存在
		resp.Code = http.StatusBadRequest
		resp.Msg = "图片不存在"
		return resp, nil

	default:
		fmt.Printf("[Error]查询图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.出库单单据是否占用
	//i 表示不区分大小写
	regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + key + ".*", Options: "i"}}
	filter = bson.M{"annex": regex}
	cur, err := l.svcCtx.OutboundOrderModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询出库单：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var orders []model.OutboundOrder
	if err = cur.All(l.ctx, &orders); err != nil {
		fmt.Println("[Error]解析出库单：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	for _, order := range orders {
		for _, annex := range strings.Split(order.Annex, ",") {
			if annex == key {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("出库单[%s]占用该图片，无法删除", order.Code)
				return resp, nil
			}
		}
	}

	//3.物料是否占用图片
	filter = bson.M{"image": key}
	singleRes = l.svcCtx.MaterialModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //图片被物料占用
		var material model.Material
		if err = singleRes.Decode(&material); err != nil {
			fmt.Printf("[Error]解析物料信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("物料[%s]占用该图片，无法删除", material.Model)
		return resp, nil

	case mongo.ErrNoDocuments: //图片未被物料占用
	default:
		fmt.Printf("[Error]查询物料图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.用户是否占用图片
	filter = bson.M{"avatar": key}
	singleRes = l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //图片被用户占用
		var user model.User
		if err = singleRes.Decode(&user); err != nil {
			fmt.Printf("[Error]解析用户信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("用户[%s]占用该图片，无法删除", user.Name)
		return resp, nil

	case mongo.ErrNoDocuments: //图片未被用户占用
	default:
		fmt.Printf("[Error]查询用户图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.供应商是否占用图片
	filter = bson.M{"image": key}
	singleRes = l.svcCtx.SupplierModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //图片被供应商占用
		var supplier model.Supplier
		if err = singleRes.Decode(&supplier); err != nil {
			fmt.Printf("[Error]解析供应商信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("供应商[%s]占用该图片，无法删除", supplier.Name)
		return resp, nil

	case mongo.ErrNoDocuments: //图片未被供应商占用
	default:
		fmt.Printf("[Error]查询供应商图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//6.客户是否占用图片
	filter = bson.M{"image": key}
	singleRes = l.svcCtx.CustomerModel.FindOne(l.ctx, filter)
	switch err = singleRes.Err(); {
	case err == nil: //图片被客户占用
		var customer model.Customer
		if err = singleRes.Decode(&customer); err != nil {
			fmt.Printf("[Error]解析客户信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("客户[%s]占用该图片，无法删除", customer.Name)
		return resp, nil
	case errors.Is(err, mongo.ErrNoDocuments): //图片未被客户占用
	default:
		fmt.Printf("[Error]查询客户图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//7.承运商是否占用图片
	filter = bson.M{"image": key}
	singleRes = l.svcCtx.CarrierModel.FindOne(l.ctx, filter)
	switch err = singleRes.Err(); {
	case err == nil: //图片被承运商占用
		var carrier model.Carrier
		if err = singleRes.Decode(&carrier); err != nil {
			fmt.Printf("[Error]解析承运商信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("承运商[%s]占用该图片，无法删除", carrier.Name)
		return resp, nil
	case errors.Is(err, mongo.ErrNoDocuments): //图片未被承运商占用
	default:
		fmt.Printf("[Error]查询承运商图片[%s]:%s\n", key, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//8.从OSS中删除
	objectPath := "images/" + key
	_, err = l.svcCtx.Cos.Object.Delete(l.ctx, objectPath)
	if err != nil {
		fmt.Printf("[Error]删除图片[%s]：%s\n", objectPath, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "失败"
		return resp, nil
	}

	//9.从数据库删除
	filter = bson.M{"object_key": key}
	_, err = l.svcCtx.ImageModel.DeleteOne(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]删除数据库图片[%s]:%s\n", key, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
