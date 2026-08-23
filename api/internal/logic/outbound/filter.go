package outbound

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type distinctCollection interface {
	Distinct(context.Context, string, interface{}, ...*options.DistinctOptions) ([]interface{}, error)
}

// applyMaterialModelFilter resolves the outbound order codes containing the
// selected model. The model predicate intentionally uses an exact match; the
// remote dropdown is responsible for fuzzy model discovery.
func applyMaterialModelFilter(
	ctx context.Context,
	collection distinctCollection,
	filter bson.M,
	model string,
) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}

	values, err := collection.Distinct(ctx, "order_code", bson.M{"model": model})
	if err != nil {
		return err
	}

	orderCodes := make([]string, 0, len(values))
	for _, value := range values {
		orderCode, ok := value.(string)
		if !ok || strings.TrimSpace(orderCode) == "" {
			continue
		}
		orderCodes = append(orderCodes, orderCode)
	}

	conditions, _ := filter["$and"].(bson.A)
	filter["$and"] = append(conditions, bson.M{
		"code": bson.M{"$in": orderCodes},
	})
	return nil
}
