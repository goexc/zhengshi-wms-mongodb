package warehouse

import (
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TreeLogic {
	return &TreeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TreeLogic) Tree() (resp *types.WarehouseTreeResponse, err error) {
	resp = new(types.WarehouseTreeResponse)

	//1.查询仓库列表
	var warehouses []model.Warehouse
	//2.查询库区列表
	var zones []model.WarehouseZone
	//3.查询货架列表
	var racks []model.WarehouseRack
	//4.查询货位列表
	var bins []model.WarehouseBin

	opts := options.Find().SetSort(bson.M{"created_at": 1})
	//1.查询仓库列表
	cur, err := l.svcCtx.WarehouseModel.Find(l.ctx, bson.M{"status": bson.M{"$ne": code.WarehouseStatusCode("删除")}}, opts)
	if err != nil {
		fmt.Printf("[Error]查询仓库树warehouse列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &warehouses); err != nil {
		fmt.Println("[Error]解析仓库树warehouse列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if len(warehouses) == 0 {
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	//2.查询库区列表
	cur, err = l.svcCtx.WarehouseZoneModel.Find(l.ctx, bson.M{"status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")}}, opts)
	if err != nil {
		fmt.Printf("[Error]查询仓库树zone列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &zones); err != nil {
		fmt.Println("[Error]解析仓库树zone列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.查询货架列表
	cur, err = l.svcCtx.WarehouseRackModel.Find(l.ctx, bson.M{"status": bson.M{"$ne": code.WarehouseRackStatusCode("删除")}}, opts)
	if err != nil {
		fmt.Printf("[Error]查询仓库树rack列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &racks); err != nil {
		fmt.Println("[Error]解析仓库树rack列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.查询货位列表
	cur, err = l.svcCtx.WarehouseBinModel.Find(l.ctx, bson.M{"status": bson.M{"$ne": code.WarehouseBinStatusCode("删除")}}, opts)
	if err != nil {
		fmt.Printf("[Error]查询仓库树bin列表:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	if err = cur.All(l.ctx, &bins); err != nil {
		fmt.Println("[Error]解析仓库树bin列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.整理数据
	//var warehouseMap = make(map[string][]types.WarehouseTree, 0)
	var zoneMap = make(map[string][]types.WarehouseTree, 0)
	var rackMap = make(map[string][]types.WarehouseTree, 0)
	var binMap = make(map[string][]types.WarehouseTree, 0)

	//var warehouseTree = make([]types.WarehouseTree, 0)
	//var zoneTree = make([]types.WarehouseTree, 0)
	//var rackTree = make([]types.WarehouseTree, 0)
	//var binTree = make([]types.WarehouseTree, 0)

	for _, one := range bins {
		//binTree = append(binTree, types.WarehouseTree{
		//	Id:       one.Id.Hex(),
		//	Name:     one.Name,
		//})

		binMap[one.WarehouseRackId.Hex()] = append(binMap[one.WarehouseRackId.Hex()], types.WarehouseTree{
			Id:   one.Id.Hex(),
			Name: one.Name,
		})
	}

	for _, one := range racks {
		tree := types.WarehouseTree{
			Id:       one.Id.Hex(),
			Name:     one.Name,
			Children: binMap[one.Id.Hex()],
		}

		rackMap[one.WarehouseZoneId.Hex()] = append(rackMap[one.WarehouseZoneId.Hex()], tree)
	}

	for _, one := range zones {
		tree := types.WarehouseTree{
			Id:       one.Id.Hex(),
			Name:     one.Name,
			Children: rackMap[one.Id.Hex()],
		}

		zoneMap[one.WarehouseId.Hex()] = append(zoneMap[one.WarehouseId.Hex()], tree)
	}

	for _, one := range warehouses {
		tree := types.WarehouseTree{
			Id:       one.Id.Hex(),
			Name:     one.Name,
			Children: zoneMap[one.Id.Hex()],
		}

		//warehouseMap[one.Id.Hex()] = append(warehouseMap[one.Id.Hex()], tree)
		resp.Data = append(resp.Data, tree)
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
