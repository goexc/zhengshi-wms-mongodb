package role

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

type StatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatusLogic {
	return &StatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StatusLogic) Status(req *types.RoleStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	if len(req.Id) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择角色"
		return resp, nil
	}

	// 1.角色是否存在
	var ids = make([]primitive.ObjectID, 0)
	for _, id := range req.Id {
		userId, e := primitive.ObjectIDFromHex(strings.TrimSpace(id))
		if e != nil {
			fmt.Printf("[Error]角色[%s]id转换：%s\n", req.Id, e.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "参数错误"
			return resp, nil
		}
		ids = append(ids, userId)
	}

	fmt.Println("角色列表：", strings.Join(req.Id, ","))
	fmt.Println("角色列表：", ids)

	var filter = bson.M{"_id": bson.M{"$in": ids}, "status": bson.M{"$ne": "删除"}}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色[%s]是否存在:%s\n", strings.Join(req.Id, ","), err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	fmt.Println("角色数量：", count, len(req.Id))
	if count < int64(len(req.Id)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分角色不存在"
		return resp, nil
	}

	//2.修改角色状态
	var update = bson.M{"$set": bson.M{
		"status":     req.Status,
		"updated_at": time.Now().Unix(),
	}}
	_, err = l.svcCtx.RoleModel.UpdateMany(l.ctx, filter, &update)
	if err != nil {
		fmt.Printf("[Error]更新角色[%s]状态：%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
