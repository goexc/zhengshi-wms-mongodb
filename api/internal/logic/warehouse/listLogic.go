package warehouse

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

func (l *ListLogic) List(req *types.WarehouseListRequest) (resp *types.WarehouseListResponse, err error) {
	resp = new(types.WarehouseListResponse)

	//1.仓库列表
	//1.1 过滤已删除仓库
	var filter = bson.M{"status": bson.M{"$ne": code.WarehouseStatusCode("删除")}} //过滤已删除仓库
	var matchStage = bson.D{}
	//1.2 查询指定名称的仓库
	name := strings.TrimSpace(req.Name)
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		//filter = bson.M{"name": regex, "status": bson.M{"$ne": 100}}
		filter["name"] = regex
	}
	//1.3 查询指定类型的仓库
	t := code.WarehouseTypeCode(strings.TrimSpace(req.Type))
	fmt.Println("仓库类型：", req.Type, t)
	if t > 0 {
		filter["type"] = t
	}

	//1.4 查询指定编号的仓库
	if strings.TrimSpace(req.Code) != "" {
		filter["code"] = strings.TrimSpace(req.Code)
	}

	//1.5 查询指定状态的仓库
	status := code.WarehouseStatusCode(strings.TrimSpace(req.Status))
	if status > 0 {
		filter["status"] = status
	}

	//注入过滤器
	matchStage = bson.D{
		{"$match", filter},
	}
	//$lookup 阶段进行关联查询，
	//将 warehouse 集合中的 create_by 字段与 user 集合中的 _id 字段进行关联
	lookupStage := bson.D{
		{"$lookup", bson.D{
			{"from", "user"},
			{"localField", "creator"},
			{"foreignField", "_id"},
			{"as", "user"},
		}},
	}

	//使用 $unwind 阶段展开 user 数组
	//使用 $unwind 阶段展开关联结果，并设置 preserveNullAndEmptyArrays 选项为 true，以便在没有关联数据时仍然保留主集合中的记录。
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$user"}, {"preserveNullAndEmptyArrays", true}}}}

	//使用 $project 阶段对结果进行投影，将 user.name 的值赋给 create_by 字段。
	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 1},
			{"type", 1},
			{"code", 1},
			{"legal_representative", 1},
			{"unified_social_credit_identifier", 1},
			{"name", 1},
			{"address", 1},
			{"contact", 1},
			{"manager", 1},
			{"level", 1},
			{"status", 1},
			{"image", 1},
			{"remark", 1},
			{"creator_name", bson.D{
				//在投影中，使用 $ifNull 操作符检查 user.name 字段，
				//如果为空（即关联集合没有数据），则将 create_by 字段置为空字符串。
				//这样就可以将关联字段置为空，以处理关联集合没有查找到数据的情况。
				{"$ifNull", bson.A{"$user.name", ""}},
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

	pipeline := mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		projectStage,
		sortStage,
	}

	cur, err := l.svcCtx.WarehouseModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询仓库列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var warehouses []model.Warehouse
	if err = cur.All(l.ctx, &warehouses); err != nil {
		fmt.Println("[Error]解析仓库列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.List = make([]types.Warehouse, 0)
	for _, warehouse := range warehouses {
		resp.Data.List = append(resp.Data.List, types.Warehouse{
			Id:        warehouse.Id.Hex(),
			Type:      code.WarehouseTypeText(warehouse.Type),
			Code:      strings.TrimSpace(warehouse.Code),
			Name:      warehouse.Name,
			Address:   warehouse.Address,
			Contact:   warehouse.Contact,
			Manager:   warehouse.Manager,
			Status:    code.WarehouseStatusText(warehouse.Status),
			Image:     warehouse.Image,
			Remark:    warehouse.Remark,
			CreateBy:  warehouse.CreatorName,
			CreatedAt: warehouse.CreatedAt,
			UpdatedAt: warehouse.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
