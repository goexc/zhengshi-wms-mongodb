package delivery

import (
	"context"
	"fmt"
	"sort"

	"api/internal/svc"
	quoteCode "api/pkg/code"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type deliveryProjectionVersion struct {
	ID             primitive.ObjectID `bson:"_id"`
	Revision       int64              `bson:"delivery_projection_revision"`
	FirstTime      int64              `bson:"first_delivery_time"`
	FirstOrderCode string             `bson:"first_delivery_order_code"`
}

// Read versions before scanning source orders. Each material is published in a
// short transaction only if no departure/removal has changed its version since.
func readDeliveryProjectionVersions(ctx context.Context, svcCtx *svc.ServiceContext) (map[string]deliveryProjectionVersion, error) {
	cur, err := svcCtx.MaterialModel.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{
		"_id": 1, "delivery_projection_revision": 1, "first_delivery_time": 1, "first_delivery_order_code": 1,
	}))
	if err != nil {
		return nil, fmt.Errorf("读取交付投影版本失败:%w", err)
	}
	defer cur.Close(ctx)
	versions := make(map[string]deliveryProjectionVersion)
	for cur.Next(ctx) {
		var version deliveryProjectionVersion
		if err = cur.Decode(&version); err != nil {
			return nil, err
		}
		versions[version.ID.Hex()] = version
	}
	return versions, cur.Err()
}

func deliveryProjectionVersionFilter(id primitive.ObjectID, revision int64) bson.M {
	filter := bson.M{"_id": id}
	if revision == 0 {
		filter["$or"] = []bson.M{{"delivery_projection_revision": int64(0)}, {"delivery_projection_revision": bson.M{"$exists": false}}}
	} else {
		filter["delivery_projection_revision"] = revision
	}
	return filter
}

func publishRebuildSummaries(ctx context.Context, svcCtx *svc.ServiceContext, versions map[string]deliveryProjectionVersion,
	summaries map[string]rebuildSummary, globals map[string]materialFirstDelivery, now int64) error {
	byMaterial := make(map[string]map[string]rebuildSummary)
	for key, summary := range summaries {
		id := summary.record.MaterialId
		if byMaterial[id] == nil {
			byMaterial[id] = make(map[string]rebuildSummary)
		}
		byMaterial[id][key] = summary
	}
	// Include historical projections and global fields whose last source vanished.
	for id, version := range versions {
		if version.FirstTime != 0 || version.FirstOrderCode != "" {
			if _, exists := byMaterial[id]; !exists {
				byMaterial[id] = nil
			}
		}
	}
	cur, err := svcCtx.CustomerMaterialDeliveryModel.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"material_id": 1}))
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row struct {
			MaterialID string `bson:"material_id"`
		}
		if err = cur.Decode(&row); err != nil {
			return err
		}
		if _, exists := byMaterial[row.MaterialID]; !exists {
			byMaterial[row.MaterialID] = nil
		}
	}
	if err = cur.Err(); err != nil {
		return err
	}
	ids := make([]string, 0, len(byMaterial))
	for id := range byMaterial {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	session, err := svcCtx.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	for _, id := range ids {
		version, exists := versions[id]
		if !exists {
			return fmt.Errorf("物料[%s]在重建扫描期间已变化或不存在，请重新执行重建", id)
		}
		_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (interface{}, error) {
			result, e := svcCtx.MaterialModel.UpdateOne(tx, deliveryProjectionVersionFilter(version.ID, version.Revision),
				bson.M{"$inc": bson.M{"delivery_projection_revision": int64(1)}})
			if e != nil {
				return nil, e
			}
			if result.MatchedCount != 1 {
				return nil, fmt.Errorf("物料[%s]的交付数据在扫描期间已变化，请重新执行重建", id)
			}
			for _, summary := range byMaterial[id] {
				if e = upsertDeliverySummary(tx, svcCtx, summary.record, now); e != nil {
					return nil, e
				}
			}
			if e = cleanupStaleCustomerMaterialDeliveries(tx, svcCtx, byMaterial[id], now, true, bson.M{"material_id": id}); e != nil {
				return nil, e
			}
			_, e = svcCtx.CustomerMaterialDeliveryModel.UpdateMany(tx, bson.M{
				"material_id": id, "$or": []bson.M{{"quote_status": bson.M{"$exists": false}}, {"quote_status": ""}},
			}, bson.M{"$set": bson.M{"quote_status": quoteCode.QuoteStatusUnquoted, "updated_at": now}})
			if e != nil {
				return nil, e
			}
			return nil, updateGlobalFirstDelivery(tx, svcCtx, version.ID, globals[id], now)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
