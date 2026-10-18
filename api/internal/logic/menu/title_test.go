package menu

import (
	"api/internal/types"
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func decodeMenuRequest(t *testing.T, source string) *types.Menu {
	t.Helper()
	var request types.Menu
	if err := json.Unmarshal([]byte(source), &request); err != nil {
		t.Fatal(err)
	}
	return &request
}

func TestCreateMenuTitleRequestToStoredBSON(t *testing.T) {
	for _, row := range []struct{ source, title, name string }{
		{`{"name":"Finance","meta":{"title":" 客户财务 "}}`, "客户财务", "Finance"},
		{`{"name":"合作伙伴","meta":{}}`, "合作伙伴", "合作伙伴"},
		{`{"name":"Finance","meta":{}}`, "", "Finance"},
	} {
		request := decodeMenuRequest(t, row.source)
		encoded, err := bson.Marshal(newMenuRecord(request, 123))
		if err != nil {
			t.Fatal(err)
		}
		record := bson.Raw(encoded)
		if record.Lookup("title").StringValue() != row.title || record.Lookup("name").StringValue() != row.name {
			t.Fatalf("title lost or identifier changed: %s", record)
		}
		if record.Lookup("created_at").Int64() != 123 {
			t.Fatal("creation metadata lost")
		}
	}
}

func TestUpdateMenuTitleOmissionsPreserveExistingValue(t *testing.T) {
	for _, title := range []string{``, `,"title":""`, `,"title":"   "`} {
		request := decodeMenuRequest(t, `{"name":"Finance","sort_id":8,"meta":{"icon":"Wallet"`+title+`}}`)
		encoded, err := bson.Marshal(bson.M{"$set": menuUpdateFields(request, 456)})
		if err != nil {
			t.Fatal(err)
		}
		fields := bson.Raw(encoded).Lookup("$set").Document()
		if _, err := fields.LookupErr("title"); err == nil {
			t.Fatalf("legacy request would erase the current title: %s", fields)
		}
		if fields.Lookup("name").StringValue() != "Finance" || fields.Lookup("sort_id").Int64() != 8 {
			t.Fatal("unrelated editable fields changed")
		}
		if _, err := fields.LookupErr("created_at"); err == nil {
			t.Fatal("update overwrites creation metadata")
		}
	}
	request := decodeMenuRequest(t, `{"name":"Finance","meta":{"title":" 客户票据管理 "}}`)
	encoded, err := bson.Marshal(bson.M{"$set": menuUpdateFields(request, 456)})
	if err != nil {
		t.Fatal(err)
	}
	fields := bson.Raw(encoded).Lookup("$set").Document()
	if fields.Lookup("title").StringValue() != "客户票据管理" || fields.Lookup("name").StringValue() != "Finance" {
		t.Fatalf("explicit title update lost: %s", fields)
	}
}
