package outbound

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type distinctCollectionStub struct {
	called    bool
	fieldName string
	filter    interface{}
	values    []interface{}
	err       error
}

func (s *distinctCollectionStub) Distinct(
	_ context.Context,
	fieldName string,
	filter interface{},
	_ ...*options.DistinctOptions,
) ([]interface{}, error) {
	s.called = true
	s.fieldName = fieldName
	s.filter = filter
	return s.values, s.err
}

func TestApplyMaterialModelFilterUsesExactModelAndOrderCodes(t *testing.T) {
	collection := &distinctCollectionStub{
		values: []interface{}{"OUT-001", "OUT-002", 3, ""},
	}
	filter := bson.M{
		"code": bson.M{"$regex": "OUT"},
	}

	err := applyMaterialModelFilter(context.Background(), collection, filter, "  RGV4102030035  ")
	if err != nil {
		t.Fatalf("applyMaterialModelFilter() error = %v", err)
	}
	if !collection.called {
		t.Fatal("expected Distinct to be called")
	}
	if collection.fieldName != "order_code" {
		t.Fatalf("Distinct field = %q, want order_code", collection.fieldName)
	}

	wantMaterialFilter := bson.M{"model": "RGV4102030035"}
	if !reflect.DeepEqual(collection.filter, wantMaterialFilter) {
		t.Fatalf("Distinct filter = %#v, want %#v", collection.filter, wantMaterialFilter)
	}

	wantConditions := bson.A{
		bson.M{"code": bson.M{"$in": []string{"OUT-001", "OUT-002"}}},
	}
	if !reflect.DeepEqual(filter["$and"], wantConditions) {
		t.Fatalf("order conditions = %#v, want %#v", filter["$and"], wantConditions)
	}
	if _, ok := filter["code"]; !ok {
		t.Fatal("existing order-code filter should be preserved")
	}
}

func TestApplyMaterialModelFilterIgnoresBlankModel(t *testing.T) {
	collection := &distinctCollectionStub{}
	filter := bson.M{}

	err := applyMaterialModelFilter(context.Background(), collection, filter, "   ")
	if err != nil {
		t.Fatalf("applyMaterialModelFilter() error = %v", err)
	}
	if collection.called {
		t.Fatal("blank model should not query outbound materials")
	}
	if len(filter) != 0 {
		t.Fatalf("blank model changed filter: %#v", filter)
	}
}

func TestApplyMaterialModelFilterReturnsDistinctError(t *testing.T) {
	wantErr := errors.New("distinct failed")
	collection := &distinctCollectionStub{err: wantErr}

	err := applyMaterialModelFilter(context.Background(), collection, bson.M{}, "RGV")
	if !errors.Is(err, wantErr) {
		t.Fatalf("applyMaterialModelFilter() error = %v, want %v", err, wantErr)
	}
}
