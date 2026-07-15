package delivery

import (
	"api/internal/svc"
	"api/model"
	quoteCode "api/pkg/code"
	"context"
	"fmt"
	"strings"
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

const (
	rebuildOrderBatchSize         = 500
	staleDeliveryQuoteLookupBatch = 1000
)

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
		if err = resetRevalidatedDeliveryQuoteState(ctx, svcCtx, filter, now); err != nil {
			return orderCount, len(summaries), fmt.Errorf("重置客户[%s]物料[%s]失效首次交付记录失败:%w", record.CustomerId, record.MaterialId, err)
		}
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
				"source_valid":              true,
				"source_invalid_reason":     "",
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

	if err = cleanupStaleCustomerMaterialDeliveries(ctx, svcCtx, summaries, now); err != nil {
		return orderCount, len(summaries), err
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

func resetRevalidatedDeliveryQuoteState(ctx context.Context, svcCtx *svc.ServiceContext, filter bson.M, now int64) error {
	resetFilter := bson.M{
		"$and": []bson.M{
			filter,
			{"source_valid": false},
			{"source_invalid_reason": bson.M{"$ne": ""}},
		},
	}
	_, err := svcCtx.CustomerMaterialDeliveryModel.UpdateOne(ctx, resetFilter, bson.M{
		"$set": bson.M{
			"quote_status":          quoteCode.QuoteStatusUnquoted,
			"latest_quote_id":       "",
			"latest_quote_no":       "",
			"latest_price":          float64(0),
			"source_invalid_reason": "",
			"updated_at":            now,
		},
	})
	return err
}

func cleanupStaleCustomerMaterialDeliveries(ctx context.Context, svcCtx *svc.ServiceContext, summaries map[string]rebuildSummary, now int64) error {
	cur, err := svcCtx.CustomerMaterialDeliveryModel.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("query customer material delivery history failed:%w", err)
	}
	defer cur.Close(ctx)

	staleRecords := make([]model.CustomerMaterialDelivery, 0)
	staleDeliveryIds := make([]string, 0)
	for cur.Next(ctx) {
		var record model.CustomerMaterialDelivery
		if err = cur.Decode(&record); err != nil {
			return fmt.Errorf("decode customer material delivery history failed:%w", err)
		}
		if _, ok := summaries[customerMaterialDeliveryKey(record.CustomerId, record.MaterialId)]; ok {
			continue
		}
		staleRecords = append(staleRecords, record)
		if !record.Id.IsZero() {
			staleDeliveryIds = append(staleDeliveryIds, record.Id.Hex())
		}
	}
	if err = cur.Err(); err != nil {
		return fmt.Errorf("iterate customer material delivery history failed:%w", err)
	}

	quotedDeliveries, err := findQuotedDeliveryIds(ctx, svcCtx, staleDeliveryIds)
	if err != nil {
		return err
	}
	for _, record := range staleRecords {
		hasQuotes := quotedDeliveries[record.Id.Hex()]
		if shouldDeleteStaleCustomerMaterialDelivery(record) && !hasQuotes {
			if _, err = svcCtx.CustomerMaterialDeliveryModel.DeleteOne(ctx, bson.M{"_id": record.Id}); err != nil {
				return fmt.Errorf("delete stale customer[%s] material[%s] delivery record failed:%w", record.CustomerId, record.MaterialId, err)
			}
			continue
		}
		update := bson.M{"$set": bson.M{
			"source_valid":          false,
			"source_invalid_reason": quoteCode.SourceInvalidReasonRebuildChanged,
			"updated_at":            now,
		}}
		if _, err = svcCtx.CustomerMaterialDeliveryModel.UpdateByID(ctx, record.Id, update); err != nil {
			return fmt.Errorf("mark stale customer[%s] material[%s] delivery record invalid failed:%w", record.CustomerId, record.MaterialId, err)
		}
		if err = invalidateDeliveryQuotesAndPrices(ctx, svcCtx, record.Id.Hex(), now); err != nil {
			return err
		}
	}
	return nil
}

func findQuotedDeliveryIds(ctx context.Context, svcCtx *svc.ServiceContext, deliveryIds []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for start := 0; start < len(deliveryIds); start += staleDeliveryQuoteLookupBatch {
		end := start + staleDeliveryQuoteLookupBatch
		if end > len(deliveryIds) {
			end = len(deliveryIds)
		}
		cur, err := svcCtx.MaterialQuoteModel.Find(
			ctx,
			bson.M{"delivery_id": bson.M{"$in": deliveryIds[start:end]}},
			options.Find().SetProjection(bson.M{"delivery_id": 1}),
		)
		if err != nil {
			return nil, fmt.Errorf("query stale delivery related quotes failed:%w", err)
		}
		for cur.Next(ctx) {
			var quote model.MaterialQuote
			if err = cur.Decode(&quote); err != nil {
				_ = cur.Close(ctx)
				return nil, fmt.Errorf("decode stale delivery related quote failed:%w", err)
			}
			if strings.TrimSpace(quote.DeliveryId) != "" {
				result[quote.DeliveryId] = true
			}
		}
		if err = cur.Err(); err != nil {
			_ = cur.Close(ctx)
			return nil, fmt.Errorf("iterate stale delivery related quotes failed:%w", err)
		}
		if err = cur.Close(ctx); err != nil {
			return nil, fmt.Errorf("close stale delivery related quotes cursor failed:%w", err)
		}
	}
	return result, nil
}

func invalidateDeliveryQuotesAndPrices(ctx context.Context, svcCtx *svc.ServiceContext, deliveryId string, now int64) error {
	cur, err := svcCtx.MaterialQuoteModel.Find(ctx, bson.M{"delivery_id": deliveryId})
	if err != nil {
		return fmt.Errorf("查询首次交付记录[%s]关联报价单失败:%w", deliveryId, err)
	}
	defer cur.Close(ctx)

	quoteIds := make([]string, 0)
	for cur.Next(ctx) {
		var quote model.MaterialQuote
		if err = cur.Decode(&quote); err != nil {
			return fmt.Errorf("解析首次交付记录[%s]关联报价单失败:%w", deliveryId, err)
		}
		if !quote.Id.IsZero() {
			quoteIds = append(quoteIds, quote.Id.Hex())
		}
	}
	if err = cur.Err(); err != nil {
		return fmt.Errorf("遍历首次交付记录[%s]关联报价单失败:%w", deliveryId, err)
	}

	quoteUpdate := bson.M{"$set": bson.M{
		"source_valid":          false,
		"source_invalid_reason": quoteCode.SourceInvalidReasonRebuildChanged,
		"updated_at":            now,
	}}
	if _, err = svcCtx.MaterialQuoteModel.UpdateMany(ctx, bson.M{"delivery_id": deliveryId}, quoteUpdate); err != nil {
		return fmt.Errorf("标记首次交付记录[%s]关联报价单失效失败:%w", deliveryId, err)
	}

	priceFilters := []bson.M{{"source_delivery_id": deliveryId}}
	if len(quoteIds) > 0 {
		priceFilters = append(priceFilters, bson.M{"source_quote_id": bson.M{"$in": quoteIds}})
	}
	priceUpdate := bson.M{"$set": bson.M{
		"source_valid":          false,
		"source_invalid_reason": quoteCode.SourceInvalidReasonRebuildChanged,
	}}
	if _, err = svcCtx.MaterialPriceModel.UpdateMany(ctx, bson.M{"$or": priceFilters}, priceUpdate); err != nil {
		return fmt.Errorf("标记首次交付记录[%s]关联物料价格失效失败:%w", deliveryId, err)
	}
	return nil
}

func shouldDeleteStaleCustomerMaterialDelivery(record model.CustomerMaterialDelivery) bool {
	quoteStatus := strings.TrimSpace(record.QuoteStatus)
	return (quoteStatus == "" || quoteStatus == quoteCode.QuoteStatusUnquoted) &&
		strings.TrimSpace(record.LatestQuoteId) == "" &&
		strings.TrimSpace(record.LatestQuoteNo) == "" &&
		record.LatestPrice <= 0
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

			key := customerMaterialDeliveryKey(order.CustomerId, material.MaterialId)
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

func customerMaterialDeliveryKey(customerId, materialId string) string {
	return customerId + "\x00" + materialId
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
