package images

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.ImagesRequest) (resp *types.ImagesResponse, err error) {
	resp = new(types.ImagesResponse)

	var filter = bson.M{}
	var option = options.Find().SetSort(bson.M{"created_at": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)
	if strings.TrimSpace(req.Name) != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + strings.TrimSpace(req.Name) + ".*", Options: "i"}}
		filter["alt"] = regex
	}

	//2.查询分页
	cur, err := l.svcCtx.ImageModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Printf("[Error]图片分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var images = make([]model.Image, 0)
	if err = cur.All(l.ctx, &images); err != nil {
		fmt.Printf("[Error]解析图片分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	for _, one := range images {
		resp.Data.List = append(resp.Data.List, types.ImageItem{
			Url: one.ObjectKey,
			Alt: one.Alt,
		})
	}

	//3.统计总数
	total, err := l.svcCtx.ImageModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]查询图片总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.Total = total
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
