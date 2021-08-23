package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
)

const (
	DefaultSubSourceName = "default"
)

func (r *sourceResolver) DeletedAt(ctx context.Context, obj *model.Source) (*time.Time, error) {
	return &obj.DeletedAt.Time, nil
}

// Source returns generated.SourceResolver implementation.
func (r *Resolver) Source() generated.SourceResolver { return &sourceResolver{r} }

type sourceResolver struct{ *Resolver }
