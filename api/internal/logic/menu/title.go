package menu

import (
	"api/internal/types"
	"api/model"
	"strings"
	"unicode"

	"go.mongodb.org/mongo-driver/bson"
)

func newMenuRecord(req *types.Menu, now int64) model.Menu {
	return model.Menu{
		Type: req.Type, Name: req.Name, Title: newMenuTitle(req.Meta.Title, req.Name),
		Path: req.Path, ParentId: req.ParentId, SortId: req.SortId, Component: req.Component,
		Icon: req.Meta.Icon, Transition: req.Meta.Transition, Hidden: req.Meta.Hidden,
		Fixed: req.Meta.Fixed, IsFull: req.Meta.IsFull, Perms: req.Meta.Perms,
		Remark: req.Remark, CreatedAt: now, UpdatedAt: now,
	}
}

func newMenuTitle(title, name string) string {
	if title = strings.TrimSpace(title); title != "" {
		return title
	}
	// Older Chinese menus used name for both identity and display. Never turn
	// an English route identifier into a new explicit display title.
	for _, r := range name {
		if unicode.Is(unicode.Han, r) {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

func menuUpdateFields(req *types.Menu, now int64) bson.M {
	fields := bson.M{
		"type": req.Type, "name": req.Name, "path": req.Path,
		"parent_id": req.ParentId, "sort_id": req.SortId, "component": req.Component,
		"icon": req.Meta.Icon, "transition": req.Meta.Transition,
		"hidden": req.Meta.Hidden, "fixed": req.Meta.Fixed, "is_full": req.Meta.IsFull,
		"perms": req.Meta.Perms, "remark": req.Remark, "updated_at": now,
	}
	// A legacy client omits meta.title. Omitting this $set field preserves the
	// current title atomically, including a concurrent title edit by another client.
	if title := strings.TrimSpace(req.Meta.Title); title != "" {
		fields["title"] = title
	}
	return fields
}
