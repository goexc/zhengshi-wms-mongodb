package receipt

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type MaterialLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMaterialLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MaterialLogic {
	return &MaterialLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MaterialLogic) Material(req *types.InboundReceiptMaterialRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, _ := primitive.ObjectIDFromHex(req.Id)
	if id.IsZero() {
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数id错误"
		return resp, nil
	}

	var receipt model.InboundReceipt

	//1.入库单号是否存在
	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.InboundReceiptModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&receipt); err != nil {
			fmt.Printf("[Error]解析入库单:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //入库单未占用
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询入库单:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//待审核、审核不通过：不能执行发货/入库操作
	if receipt.Status == code.InboundReceiptStatusCode("待审核") || receipt.Status == code.InboundReceiptStatusCode("审核不通过") {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请审核后再操作"
		return resp, nil
	}

	//入库完成：不能继续执行发货/入库操作
	if receipt.Status == code.InboundReceiptStatusCode("入库完成") {
		resp.Code = http.StatusBadRequest
		resp.Msg = "入库单已完成，不能操作发货/入库"
		return resp, nil
	}

	//2.入库单状态
	var statuses = make(map[int]int, 0)

	//3.更改物料状态、实际入库数量
	if len(receipt.Materials) != len(req.Materials) {
		resp.Code = http.StatusOK
		resp.Msg = "物料数量不一致"
		return resp, nil
	}

	var materialsMap = make(map[string]types.InboundMaterialStatus)
	for idx := range req.Materials {
		materialsMap[req.Materials[idx].Id] = req.Materials[idx]
	}

	for idx, one := range receipt.Materials {
		if _, ok := materialsMap[one.Id]; !ok {
			fmt.Printf("[Error]物料[%s]缺少状态值\n", one.Name)
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("物料[%s]未设置入库状态", one.Name)
			return resp, nil
		}

		receipt.Materials[idx].Status = code.InboundReceiptStatusCode(materialsMap[one.Id].Status)
		statuses[receipt.Materials[idx].Status]++
	}

	//4.更新物料状态和入库单状态
	update := bson.M{
		"$set": bson.M{
			"status":    getReceiptStatus(statuses),
			"materials": receipt.Materials,
		},
	}
	_, err = l.svcCtx.InboundReceiptModel.UpdateByID(l.ctx, receipt.Id, &update)
	if err != nil {
		fmt.Printf("[Error]更新入库单[%s]物料状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}

// 根据物料状态确定入库单状态
// statuses: map[物料状态值]对应的物料数量
func getReceiptStatus(statuses map[int]int) (status int) {
	//0 只有一个状态值时，直接返回该状态值
	//0.1 物料全部作废，入库单状态才作废，否则忽略
	//0.2 物料全部完成，入库单状态修改为入库完成
	//1 存在部分入库的物料，入库单状态统一设置为部分入库
	//2 先选择最大的物料状态值
	for key := range statuses {
		if len(statuses) == 1 {
			fmt.Println("只有状态：", key)
			return key
		}

		if key == code.InboundReceiptStatusCode("部分入库") {
			fmt.Println("部分入库")
			return key
		}

		if status < key {
			status = key
		}
	}

	fmt.Println("状态：", status)
	return status
}
