package inbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProcurementLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProcurementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProcurementLogic {
	return &ProcurementLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 采购订单编号格式
const codeFormat = "PC-%s-%d"

func (l *ProcurementLogic) Procurement(req *types.ProcurementRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.采购物料索引校验
	var index = make(map[string]int, len(req.Materials))
	var quantity = make(map[string]float64, len(req.Materials))
	for _, m := range req.Materials {
		if m.Index >= len(req.Materials) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "物料顺序错误"
			return resp, nil
		}
		index[m.Id] = m.Index
		quantity[m.Id] = m.Quantity
	}

	//1.入库单号为空时，自动生成
	var code = strings.TrimSpace(req.Code)
	if code == "" {
		timestamp := time.Now().Format("20060102150405")
		randomNum := rand.Intn(10000)
		code = fmt.Sprintf(codeFormat, timestamp, randomNum)
	}

	//2.订单编号是否重复
	var order = strings.TrimSpace(req.Order)
	var filter = bson.M{}
	if order != "" {
		filter = bson.M{"order": order}
		count, err := l.svcCtx.InboundModel.CountDocuments(l.ctx, filter)
		if err != nil {
			fmt.Printf("[Error]查询订单编号[%s]是否占用：%s\n", order, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if count > 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "订单编号重复"
			return resp, nil
		}
	}

	//TODO:4.仓库是否存在

	//3.确认采购清单中的物料
	if len(req.Materials) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请填写需要采购的物料清单"
		return resp, nil
	}

	var materialsId bson.A
	for _, m := range req.Materials {
		mId, e := primitive.ObjectIDFromHex(m.Id)
		if e != nil {
			fmt.Printf("[Error]物料[%s]格式错误:%s\n", m.Id, e.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "物料不存在"
			return resp, nil
		}

		materialsId = append(materialsId, mId)
	}

	filter = bson.M{"_id": bson.M{"$in": materialsId}}

	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询物料清单：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Material
	if err = cur.All(l.ctx, &ms); err != nil {
		fmt.Println("[Error]解析物料清单：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.判断缺失的物料
	if len(ms) < len(req.Materials) {
		var provide = make(map[string]types.InboundMaterial)
		for _, m := range req.Materials {
			provide[m.Id] = m
		}

		for _, m := range ms {
			delete(provide, m.Id.Hex())
		}

		var materialsName []string
		for _, m := range provide {
			materialsName = append(materialsName, m.Name)
		}

		resp.Code = http.StatusNoContent
		resp.Msg = fmt.Sprintf("没有查询到以下物料：%s", strings.Join(materialsName, ","))
		return resp, nil
	}

	//5.整理数据
	var materials = make([]model.InboundMaterial, len(ms))
	for _, m := range ms {
		materials[index[m.Id.Hex()]] = model.InboundMaterial{
			Index:        index[m.Id.Hex()],
			MaterialId:   m.Id,
			MaterialName: m.Name,
			Quantity:     quantity[m.Id.Hex()],
			Unit:         m.Unit,
		}
	}

	var inbound = model.Inbound{
		Code:       code,
		Order:      order,
		SupplierId: primitive.ObjectID{},
		Status:     "待审核",
		Materials:  materials,
		Remark:     strings.TrimSpace(req.Remark),
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	_, err = l.svcCtx.InboundModel.InsertOne(l.ctx, &inbound)
	if err != nil {
		fmt.Printf("[Error]创建采购入库单:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
