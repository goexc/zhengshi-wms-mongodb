package material

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

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

func (l *RemoveLogic) Remove(req *types.MaterialIdRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]物料[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	//TODO:1.物料是否使用

	//2.创建事务
	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]删除物料：创建事务：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	// 根据会话获取数据库操作的上下文
	dbCtx := mongo.NewSessionContext(l.ctx, session)

	//2.1 开始事务
	err = session.StartTransaction()
	if err != nil {
		fmt.Printf("[Error]删除物料：开始事务：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.删除物料（mongodb事务）
	filter := bson.M{"_id": id}
	singleRes := l.svcCtx.MaterialModel.FindOneAndDelete(dbCtx, filter)
	switch singleRes.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		// 回滚事务
		session.AbortTransaction(dbCtx)

		resp.Msg = "物料不存在"
		resp.Code = http.StatusBadRequest
		return resp, nil
	default:
		fmt.Printf("[Error]删除物料[%s]：%s\n", req.Id, singleRes.Err().Error())

		// 回滚事务
		session.AbortTransaction(dbCtx)

		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	//4.删除物料价格（mongodb事务）
	_, err = l.svcCtx.MaterialPriceModel.DeleteMany(dbCtx, bson.M{"material": req.Id})
	if err != nil {
		fmt.Printf("[Error]删除物料[%s]价格：%s\n", req.Id, err.Error())
		// 回滚事务
		session.AbortTransaction(dbCtx)

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	// 提交事务
	err = session.CommitTransaction(dbCtx)
	if err != nil {
		fmt.Printf("[Error]删除物料[%s]事务提交失败: %s\n", req.Id, err.Error())

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
