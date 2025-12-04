package biz

import (
	"context"
	"strconv"
)

// Segment represents a basic segment structure
type Segment struct {
	ID       int
	ParentID *int
}

// Placeholder resolvers for GraphQL
type segmentResolver struct{}

func (r *segmentResolver) ID(ctx context.Context, obj *Segment) (string, error) {
	return strconv.Itoa(obj.ID), nil
}

func (r *segmentResolver) ParentID(ctx context.Context, obj *Segment) (*string, error) {
	if obj.ParentID != nil {
		str := strconv.Itoa(*obj.ParentID)
		return &str, nil
	}
	return nil, nil
}