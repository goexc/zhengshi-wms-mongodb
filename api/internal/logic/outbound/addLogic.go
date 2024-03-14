package outbound

import (
	"api/model"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// 针对客户的出库类型
var customerTypes = map[string]string{"销售出库": "", "样品出库": "", "赠品出库": ""}

// 针对供应商的出库类型
// var supplierTypes = map[string]string{"退货出库": "", "退料出库": ""}
var supplierTypes = map[string]string{"退货出库": ""}

//TODO:针对生产线的出库类型

func (l *AddLogic) Add(req *types.OutboundOrderAddRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	var receipt model.OutboundOrder
	receipt.Status = "预发货"
	receipt.IsPack = 0
	receipt.IsWeigh = 0
	receipt.Type = strings.TrimSpace(req.Type)
	receipt.Code = strings.TrimSpace(req.Code)
	receipt.Remark = strings.TrimSpace(req.Remark)
	receipt.Annex = strings.Join(req.Annex, ",")

	//1.出库单号是否冲突
	var filter = bson.M{"code": req.Code, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.OutboundOrderModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询出库单[%s]是否冲突:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单号已占用"
		return resp, nil
	}

	//2.供应商是否存在
	if _, ok := supplierTypes[req.Type]; ok {
		var supplier model.Supplier
		supplierId, _ := primitive.ObjectIDFromHex(req.SupplierId)
		singleRes := l.svcCtx.SupplierModel.FindOne(l.ctx, bson.M{"_id": supplierId})
		switch singleRes.Err() {
		case nil: //供应商存在
			if e := singleRes.Decode(&supplier); e != nil {
				fmt.Printf("[Error]解析供应商[%s]:%s\n", req.SupplierId, e.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //供应商不存在
			fmt.Printf("[Error]供应商[%s]不存在\n", req.SupplierId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "供应商不存在"
			return resp, nil
		default: //其他错误
			fmt.Printf("[Error]查询供应商[%s]是否存在:%s\n", req.SupplierId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		receipt.SupplierId = supplier.Id.Hex()
		receipt.SupplierName = supplier.Name
	}

	//3.客户是否存在
	var customer model.Customer
	if _, ok := customerTypes[req.Type]; ok {
		customerId, _ := primitive.ObjectIDFromHex(req.CustomerId)
		singleRes := l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId})
		switch singleRes.Err() {
		case nil: //客户存在
			if e := singleRes.Decode(&customer); e != nil {
				fmt.Printf("[Error]解析客户[%s]:%s\n", req.CustomerId, e.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //客户不存在
			fmt.Printf("[Error]客户[%s]不存在\n", req.CustomerId)
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户不存在"
			return resp, nil
		default: //其他错误
			fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.CustomerId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}

		receipt.CustomerId = customer.Id.Hex()
		receipt.CustomerName = customer.Name
	}

	//TODO:4.生产线是否存在

	//4.收集仓库、库区、货架、货位id
	var materialsId = make([]primitive.ObjectID, 0)

	//4.1 收集id
	for _, one := range req.Materials {
		materialId, _ := primitive.ObjectIDFromHex(strings.TrimSpace(one.MaterialId))
		materialsId = append(materialsId, materialId)
	}

	//4.2 查询物料
	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": materialsId}})
	if err != nil {
		fmt.Printf("[Error]查询物料列表失败：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Material
	if err = cur.All(l.ctx, &ms); err != nil {
		fmt.Printf("[Error]解析物料分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if len(materialsId) != len(ms) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分物料不存在"
		return resp, nil
	}

	var materialsMap = make(map[string]model.Material)
	for _, one := range ms {
		materialsMap[one.Id.Hex()] = one
	}

	//总金额
	var amount decimal.Decimal
	for _, one := range req.Materials {
		amount = decimal.NewFromFloat(one.Quantity).Mul(decimal.NewFromFloat(one.Price)).Add(amount)
	}
	receipt.TotalAmount = amount.InexactFloat64()

	receipt.CreatorId = l.ctx.Value("uid").(string)
	receipt.CreatorName = l.ctx.Value("name").(string)
	receipt.CreatedAt = time.Now().Unix()

	_, err = l.svcCtx.OutboundOrderModel.InsertOne(l.ctx, &receipt)
	if err != nil {
		fmt.Printf("[Error]新增出库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.仓库、库区、货架、货位是否存在
	var materials = make([]interface{}, 0) //出库单的物料列表
	for _, one := range req.Materials {
		//收集物料列表
		im := model.OutboundOrderMaterial{
			OrderCode:  req.Code,
			MaterialId: one.MaterialId,
			Index:      one.Index,
			Price:      one.Price,
			Name:       materialsMap[one.MaterialId].Name,
			Model:      materialsMap[one.MaterialId].Model,
			Quantity:   one.Quantity,
			Unit:       materialsMap[one.MaterialId].Unit,
		}

		//校验仓库、库区、货架、货位
		materials = append(materials, im)

		//收集物料价格
		if one.Price > 0 {
			var update = bson.M{
				"$set": bson.M{
					"material":      one.MaterialId,
					"customer_id":   customer.Id.Hex(),
					"customer_name": customer.Name,
					"price":         one.Price,
					"creator":       l.ctx.Value("uid").(string),
					"creator_name":  l.ctx.Value("name").(string),
					"created_at":    time.Now().Unix(),
				},
			}

			//记录物料单价
			opts := options.Update().SetUpsert(true) //更新时，不存在就插入
			_, err = l.svcCtx.MaterialPriceModel.UpdateOne(l.ctx, bson.M{"material": one.MaterialId, "customer_id": customer.Id.Hex(), "price": one.Price}, update, opts)
			if err != nil {
				fmt.Printf("[Error]记录物料价格:%s\n", err.Error())
				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		}
	}

	//6.存储出库单物料
	_, err = l.svcCtx.OutboundMaterialModel.InsertMany(l.ctx, materials)
	if err != nil {
		fmt.Printf("[Error]保存出库单[%s]物料:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
