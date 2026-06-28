package delivery

import (
	"api/internal/svc"
	"api/model"
	quoteCode "api/pkg/code"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SyncCustomerMaterialDeliveries 维护客户物料首次交付记录和物料全局首次交付时间。
func SyncCustomerMaterialDeliveries(ctx context.Context, svcCtx *svc.ServiceContext, order model.OutboundOrder, materials []model.OutboundOrderMaterial) error {
	if !quoteCode.IsCustomerDeliveryOutbound(order.TypeCode, order.Type) {
		return nil
	}
	if order.CustomerId == "" || order.DepartureTime <= 0 || len(materials) == 0 {
		return nil
	}

	now := time.Now().Unix()
	for _, material := range materials {
		if material.MaterialId == "" {
			return fmt.Errorf("出库单[%s]存在空物料id", order.Code)
		}
		if err := syncGlobalMaterialFirstDelivery(ctx, svcCtx, order, material, now); err != nil {
			return err
		}
		if err := syncCustomerMaterialDelivery(ctx, svcCtx, order, material, now); err != nil {
			return err
		}
	}
	return nil
}

func syncGlobalMaterialFirstDelivery(ctx context.Context, svcCtx *svc.ServiceContext, order model.OutboundOrder, material model.OutboundOrderMaterial, now int64) error {
	materialId, err := primitive.ObjectIDFromHex(material.MaterialId)
	if err != nil {
		return fmt.Errorf("出库单[%s]物料id[%s]格式错误:%w", order.Code, material.MaterialId, err)
	}

	filter := bson.M{
		"_id": materialId,
		"$or": []bson.M{
			{"first_delivery_time": bson.M{"$exists": false}},
			{"first_delivery_time": int64(0)},
			{"first_delivery_time": bson.M{"$gt": order.DepartureTime}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"first_delivery_time":       order.DepartureTime,
			"first_delivery_order_code": order.Code,
			"updated_at":                now,
		},
	}
	if _, err = svcCtx.MaterialModel.UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("更新物料[%s]全局首次交付时间失败:%w", material.MaterialId, err)
	}
	return nil
}

func syncCustomerMaterialDelivery(ctx context.Context, svcCtx *svc.ServiceContext, order model.OutboundOrder, material model.OutboundOrderMaterial, now int64) error {
	filter := bson.M{
		"customer_id": order.CustomerId,
		"material_id": material.MaterialId,
	}

	var existing model.CustomerMaterialDelivery
	err := svcCtx.CustomerMaterialDeliveryModel.FindOne(ctx, filter).Decode(&existing)
	switch err {
	case nil:
		updateSet := bson.M{
			"material_name":          material.Name,
			"material_model":         material.Model,
			"material_specification": material.Specification,
			"material_unit":          material.Unit,
			"updated_at":             now,
		}
		if existing.QuoteStatus == "" {
			updateSet["quote_status"] = quoteCode.QuoteStatusUnquoted
		}
		if existing.FirstDeliveryTime == 0 || existing.FirstDeliveryTime > order.DepartureTime {
			updateSet["first_delivery_time"] = order.DepartureTime
			updateSet["first_delivery_order_code"] = order.Code
			updateSet["first_delivery_quantity"] = material.Quantity
			updateSet["first_delivery_price"] = material.Price
		}

		update := bson.M{
			"$set": updateSet,
			"$max": bson.M{
				"last_delivery_time": order.DepartureTime,
			},
			"$inc": bson.M{
				"delivery_count": int64(1),
			},
		}
		if _, err = svcCtx.CustomerMaterialDeliveryModel.UpdateOne(ctx, filter, update); err != nil {
			return fmt.Errorf("更新客户[%s]物料[%s]首次交付记录失败:%w", order.CustomerId, material.MaterialId, err)
		}
		return nil
	case mongo.ErrNoDocuments:
		record := model.CustomerMaterialDelivery{
			Id:                     primitive.NewObjectID(),
			CustomerId:             order.CustomerId,
			CustomerName:           order.CustomerName,
			MaterialId:             material.MaterialId,
			MaterialName:           material.Name,
			MaterialModel:          material.Model,
			MaterialSpecification:  material.Specification,
			MaterialUnit:           material.Unit,
			FirstDeliveryTime:      order.DepartureTime,
			FirstDeliveryOrderCode: order.Code,
			FirstDeliveryQuantity:  material.Quantity,
			FirstDeliveryPrice:     material.Price,
			LastDeliveryTime:       order.DepartureTime,
			DeliveryCount:          1,
			QuoteStatus:            quoteCode.QuoteStatusUnquoted,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if _, err = svcCtx.CustomerMaterialDeliveryModel.InsertOne(ctx, &record); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return syncCustomerMaterialDelivery(ctx, svcCtx, order, material, now)
			}
			return fmt.Errorf("创建客户[%s]物料[%s]首次交付记录失败:%w", order.CustomerId, material.MaterialId, err)
		}
		return nil
	default:
		return fmt.Errorf("查询客户[%s]物料[%s]首次交付记录失败:%w", order.CustomerId, material.MaterialId, err)
	}
}
