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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type rebuildSummary struct {
	record model.CustomerMaterialDelivery
}

type materialFirstDelivery struct {
	materialId primitive.ObjectID
	time       int64
	orderCode  string
}

// RebuildProgressFunc reports rebuild progress after each order batch.
type RebuildProgressFunc func(orderCount, deliveryCount int)

const rebuildOrderBatchSize = 500

// RebuildCustomerMaterialDeliveries 从已有出库单重建客户物料首次交付记录。
func RebuildCustomerMaterialDeliveries(ctx context.Context, svcCtx *svc.ServiceContext, progress ...RebuildProgressFunc) (int, int, error) {
	orderFilter := bson.M{
		"customer_id":    bson.M{"$ne": ""},
		"departure_time": bson.M{"$gt": int64(0)},
		"status":         bson.M{"$in": []string{"已出库", "已签收"}},
		"$or": []bson.M{
			{"type_code": bson.M{"$in": []string{quoteCode.OutboundTypeCodeSales, quoteCode.OutboundTypeCodeSample, quoteCode.OutboundTypeCodeGift}}},
			{"type": bson.M{"$in": []string{"销售出库", "样品出库", "赠品出库"}}},
		},
	}
	orderOpts := options.Find().SetSort(bson.D{{"departure_time", 1}, {"code", 1}})
	orderCur, err := svcCtx.OutboundOrderModel.Find(ctx, orderFilter, orderOpts)
	if err != nil {
		return 0, 0, fmt.Errorf("查询客户交付出库单失败:%w", err)
	}
	defer orderCur.Close(ctx)

	now := time.Now().Unix()
	summaries := make(map[string]rebuildSummary)
	globalFirstDeliveries := make(map[string]materialFirstDelivery)
	orderBatch := make([]model.OutboundOrder, 0, rebuildOrderBatchSize)
	orderCount := 0

	for orderCur.Next(ctx) {
		var order model.OutboundOrder
		if err = orderCur.Decode(&order); err != nil {
			return 0, 0, fmt.Errorf("解析客户交付出库单失败:%w", err)
		}

		orderBatch = append(orderBatch, order)
		if len(orderBatch) >= rebuildOrderBatchSize {
			if err = processRebuildOrderBatch(ctx, svcCtx, orderBatch, summaries, globalFirstDeliveries, now); err != nil {
				return orderCount, len(summaries), err
			}
			orderCount += len(orderBatch)
			reportRebuildProgress(progress, orderCount, len(summaries))
			orderBatch = orderBatch[:0]
		}
	}
	if err = orderCur.Err(); err != nil {
		return 0, 0, fmt.Errorf("遍历客户交付出库单失败:%w", err)
	}
	if len(orderBatch) > 0 {
		if err = processRebuildOrderBatch(ctx, svcCtx, orderBatch, summaries, globalFirstDeliveries, now); err != nil {
			return orderCount, len(summaries), err
		}
		orderCount += len(orderBatch)
		reportRebuildProgress(progress, orderCount, len(summaries))
	}

	for _, summary := range summaries {
		record := summary.record
		filter := bson.M{"customer_id": record.CustomerId, "material_id": record.MaterialId}
		update := bson.M{
			"$set": bson.M{
				"customer_name":             record.CustomerName,
				"material_name":             record.MaterialName,
				"material_model":            record.MaterialModel,
				"material_specification":    record.MaterialSpecification,
				"material_unit":             record.MaterialUnit,
				"first_delivery_time":       record.FirstDeliveryTime,
				"first_delivery_order_code": record.FirstDeliveryOrderCode,
				"first_delivery_quantity":   record.FirstDeliveryQuantity,
				"first_delivery_price":      record.FirstDeliveryPrice,
				"last_delivery_time":        record.LastDeliveryTime,
				"delivery_count":            record.DeliveryCount,
				"updated_at":                now,
			},
			"$setOnInsert": bson.M{
				"quote_status": quoteCode.QuoteStatusUnquoted,
				"created_at":   now,
			},
		}
		if _, err = svcCtx.CustomerMaterialDeliveryModel.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true)); err != nil {
			return orderCount, 0, fmt.Errorf("回填客户[%s]物料[%s]首次交付记录失败:%w", record.CustomerId, record.MaterialId, err)
		}
	}

	missingQuoteStatusFilter := bson.M{
		"$or": []bson.M{
			{"quote_status": bson.M{"$exists": false}},
			{"quote_status": ""},
		},
	}
	if _, err = svcCtx.CustomerMaterialDeliveryModel.UpdateMany(ctx, missingQuoteStatusFilter, bson.M{"$set": bson.M{"quote_status": quoteCode.QuoteStatusUnquoted, "updated_at": now}}); err != nil {
		return orderCount, len(summaries), fmt.Errorf("修复空报价状态失败:%w", err)
	}

	for _, first := range globalFirstDeliveries {
		update := bson.M{
			"$set": bson.M{
				"first_delivery_time":       first.time,
				"first_delivery_order_code": first.orderCode,
				"updated_at":                now,
			},
		}
		if _, err = svcCtx.MaterialModel.UpdateByID(ctx, first.materialId, update); err != nil {
			return orderCount, len(summaries), fmt.Errorf("回填物料[%s]全局首次交付时间失败:%w", first.materialId.Hex(), err)
		}
	}

	return orderCount, len(summaries), nil
}

func processRebuildOrderBatch(ctx context.Context, svcCtx *svc.ServiceContext, orders []model.OutboundOrder, summaries map[string]rebuildSummary, globalFirstDeliveries map[string]materialFirstDelivery, now int64) error {
	materialByOrder, err := outboundMaterialsByOrders(ctx, svcCtx, orders)
	if err != nil {
		return err
	}

	for _, order := range orders {
		materialMap := materialByOrder[order.Code]
		for _, material := range materialMap {
			materialObjectId, e := primitive.ObjectIDFromHex(material.MaterialId)
			if e != nil {
				return fmt.Errorf("出库单[%s]物料id[%s]格式错误:%w", order.Code, material.MaterialId, e)
			}

			key := order.CustomerId + "\x00" + material.MaterialId
			summary, ok := summaries[key]
			if !ok {
				summary.record = model.CustomerMaterialDelivery{
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
					CreatedAt:              now,
					UpdatedAt:              now,
				}
			} else {
				summary.record.CustomerName = order.CustomerName
				summary.record.MaterialName = material.Name
				summary.record.MaterialModel = material.Model
				summary.record.MaterialSpecification = material.Specification
				summary.record.MaterialUnit = material.Unit
				summary.record.DeliveryCount++
				summary.record.UpdatedAt = now
				if order.DepartureTime < summary.record.FirstDeliveryTime {
					summary.record.FirstDeliveryTime = order.DepartureTime
					summary.record.FirstDeliveryOrderCode = order.Code
					summary.record.FirstDeliveryQuantity = material.Quantity
					summary.record.FirstDeliveryPrice = material.Price
				}
				if order.DepartureTime > summary.record.LastDeliveryTime {
					summary.record.LastDeliveryTime = order.DepartureTime
				}
			}
			summaries[key] = summary

			global, exists := globalFirstDeliveries[material.MaterialId]
			if !exists || order.DepartureTime < global.time {
				globalFirstDeliveries[material.MaterialId] = materialFirstDelivery{
					materialId: materialObjectId,
					time:       order.DepartureTime,
					orderCode:  order.Code,
				}
			}
		}
	}
	return nil
}

func outboundMaterialsByOrders(ctx context.Context, svcCtx *svc.ServiceContext, orders []model.OutboundOrder) (map[string]map[string]model.OutboundOrderMaterial, error) {
	result := make(map[string]map[string]model.OutboundOrderMaterial, len(orders))
	orderCodes := make([]string, 0, len(orders))
	for _, order := range orders {
		orderCodes = append(orderCodes, order.Code)
		result[order.Code] = make(map[string]model.OutboundOrderMaterial)
	}
	if len(orderCodes) == 0 {
		return result, nil
	}

	cur, err := svcCtx.OutboundMaterialModel.Find(ctx, bson.M{"order_code": bson.M{"$in": orderCodes}})
	if err != nil {
		return nil, fmt.Errorf("批量查询出库单物料失败:%w", err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var material model.OutboundOrderMaterial
		if err = cur.Decode(&material); err != nil {
			return nil, fmt.Errorf("解析出库单物料失败:%w", err)
		}
		if _, ok := result[material.OrderCode]; !ok {
			result[material.OrderCode] = make(map[string]model.OutboundOrderMaterial)
		}
		if existing, ok := result[material.OrderCode][material.MaterialId]; ok {
			existing.Quantity += material.Quantity
			if existing.Price <= 0 && material.Price > 0 {
				existing.Price = material.Price
			}
			result[material.OrderCode][material.MaterialId] = existing
			continue
		}
		result[material.OrderCode][material.MaterialId] = material
	}
	if err = cur.Err(); err != nil {
		return nil, fmt.Errorf("遍历出库单物料失败:%w", err)
	}
	return result, nil
}

func reportRebuildProgress(progress []RebuildProgressFunc, orderCount, deliveryCount int) {
	for _, fn := range progress {
		if fn != nil {
			fn(orderCount, deliveryCount)
		}
	}
}
