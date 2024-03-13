package price

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *types.MaterialPricesRequest) (resp *types.MaterialPricesResponse, err error) {
	resp = new(types.MaterialPricesResponse)

	id, _ := primitive.ObjectIDFromHex(req.MaterialId)

	//1.物料是否存在
	filter := bson.M{"_id": id}
	var material model.Material
	singleRes := l.svcCtx.MaterialModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&material); err != nil {
			fmt.Printf("[Error]解析物料:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //物料未占用
		resp.Code = http.StatusBadRequest
		resp.Msg = "物料不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询物料:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.查询物料价格
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	filter = bson.M{"material": strings.TrimSpace(req.MaterialId)}
	if req.CustomerId != "" {
		filter["customer_id"] = strings.TrimSpace(req.CustomerId)
	}

	cur, err := l.svcCtx.MaterialPriceModel.Find(l.ctx, filter, opts)
	if err != nil {
		fmt.Printf("[Error]查询物料[%s]价格:%s\n", req.MaterialId, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var prices []model.MaterialPrice
	if err = cur.All(l.ctx, &prices); err != nil {
		fmt.Printf("[Error]解析物料[%s]价格列表：%s\n", req.MaterialId, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	for _, one := range prices {
		resp.Data = append(resp.Data, types.MaterialPrice{
			Price:        one.Price,
			Since:        one.CreatedAt,
			CustomerName: one.CustomerName,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
