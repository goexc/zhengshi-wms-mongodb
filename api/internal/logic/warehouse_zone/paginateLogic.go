package warehouse_zone

import (
	"api/model"
	"api/pkg/code"
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

type PaginateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPaginateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PaginateLogic {
	return &PaginateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PaginateLogic) Paginate(req *types.WarehouseZonesRequest) (resp *types.WarehouseZonesResponse, err error) {
	resp = new(types.WarehouseZonesResponse)

	//1.库区分页
	//1.1 过滤已删除库区
	var filter = bson.M{"status": bson.M{"$ne": code.WarehouseZoneStatusCode("删除")}} //过滤已删除库区
	var matchStage = bson.D{}
	//1.2 查询指定名称的库区
	name := strings.TrimSpace(req.Name)
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter["name"] = regex
	}
	//1.3 查询指定编号的库区
	if strings.TrimSpace(req.Code) != "" {
		filter["code"] = strings.TrimSpace(req.Code)
	}

	//1.4 查询指定状态的库区
	status := code.WarehouseZoneStatusCode(strings.TrimSpace(req.Status))
	if status > 0 {
		filter["status"] = status
	}
	//1.5 从指定的仓库中查询
	if strings.TrimSpace(req.WarehouseId) != "" {
		warehouseId, e := primitive.ObjectIDFromHex(req.WarehouseId)
		if e != nil {
			fmt.Printf("[Error]仓库错误[%s]:%s\n", req.WarehouseId, e.Error())
			resp.Code = http.StatusBadRequest
			resp.Msg = "请选择可用仓库"
			return resp, nil
		}
		filter["warehouse_id"] = warehouseId
	}

	//注入过滤器
	matchStage = bson.D{
		{"$match", filter},
	}
	//$lookup 阶段进行关联查询，
	lookupUserStage := bson.D{
		//将 warehouse_zone 集合中的 create_by 字段与 user 集合中的 _id 字段进行关联
		{"$lookup", bson.D{
			{"from", "user"},
			{"localField", "creator"},
			{"foreignField", "_id"},
			{"as", "user"},
		}},
	}
	lookupWarehouseStage := bson.D{
		//将 warehouse_zone 集合中的 warehouse_id 字段与 warehouse 集合中的 _id 字段进行关联
		{"$lookup", bson.D{
			{"from", "warehouse"},
			{"localField", "warehouse_id"},
			{"foreignField", "_id"},
			{"as", "warehouse"},
		}},
	}

	//使用 $unwind 阶段展开 user、warehouse 数组
	//使用 $unwind 阶段展开关联结果，并设置 preserveNullAndEmptyArrays 选项为 true，以便在没有关联数据时仍然保留主集合中的记录。
	unwindUserStage := bson.D{
		{"$unwind", bson.D{{"path", "$user"}, {"preserveNullAndEmptyArrays", true}}},
	}
	unwindWarehouseStage := bson.D{
		{"$unwind", bson.D{{"path", "$warehouse"}, {"preserveNullAndEmptyArrays", true}}},
	}

	//使用 $project 阶段对结果进行投影，将 user.name 的值赋给 create_by 字段，将 warehouse.name 的值赋给 warehouse_name 字段。
	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 1},
			{"warehouse_id", 1},
			{"name", 1},
			{"code", 1},
			{"capacity", 1},
			{"capacity_unit", 1},
			{"status", 1},
			{"manager", 1},
			{"contact", 1},
			{"remark", 1},
			{"creator_name", bson.D{
				//在投影中，使用 $ifNull 操作符检查 user.name 字段，
				//如果为空（即关联集合没有数据），则将 create_by 字段置为空字符串。
				//这样就可以将关联字段置为空，以处理关联集合没有查找到数据的情况。
				{"$ifNull", bson.A{"$user.name", ""}},
			}},
			{"warehouse_name", bson.D{
				//在投影中，使用 $ifNull 操作符检查 warehouse.name 字段，
				//如果为空（即关联集合没有数据），则将 warehouse_name 字段置为空字符串。
				//这样就可以将关联字段置为空，以处理关联集合没有查找到数据的情况。
				{"$ifNull", bson.A{"$warehouse.name", ""}},
			}},
			{"created_at", 1},
			{"updated_at", 1},
			//{"creator", 0},//$project 阶段只能用于指定要包含的字段，而不能同时指定要包含和排除的字段。
		}},
	}
	sortStage := bson.D{
		{"$sort", bson.D{
			{"created_at", -1}, // 降序排列
		},
		},
	}
	skipStage := bson.D{{"$skip", (req.Page - 1) * req.Size}}
	limitStage := bson.D{{"$limit", req.Size}}

	pipeline := mongo.Pipeline{
		matchStage,
		lookupUserStage,
		lookupWarehouseStage,
		unwindUserStage,
		unwindWarehouseStage,
		projectStage,
		sortStage,
		skipStage,
		limitStage,
	}

	cur, err := l.svcCtx.WarehouseZoneModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询库区列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var zones []model.WarehouseZone
	if err = cur.All(l.ctx, &zones); err != nil {
		fmt.Println("[Error]解析库区列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.库区总数量
	total, err := l.svcCtx.WarehouseZoneModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]库区总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.Total = total
	resp.Data.List = make([]types.WarehouseZone, 0)
	for _, zone := range zones {
		resp.Data.List = append(resp.Data.List, types.WarehouseZone{
			Id:            zone.Id.Hex(),
			WarehouseId:   zone.WarehouseId.Hex(),
			WarehouseName: zone.WarehouseName,
			Name:          zone.Name,
			Code:          strings.TrimSpace(zone.Code),
			Capacity:      zone.Capacity,
			CapacityUnit:  zone.CapacityUnit,
			Manager:       zone.Manager,
			Contact:       zone.Contact,
			Status:        code.WarehouseZoneStatusText(zone.Status),
			Remark:        zone.Remark,
			CreateBy:      zone.CreatorName,
			CreatedAt:     zone.CreatedAt,
			UpdatedAt:     zone.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
