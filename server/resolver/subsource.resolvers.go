package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
)

func (r *subSourceResolver) DeletedAt(ctx context.Context, obj *model.SubSource) (*time.Time, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *subSourceResolver) Source(ctx context.Context, obj *model.SubSource) (*model.Source, error) {
	var source model.Source
	r.DB.Where("id = ?", obj.SourceID).First(&source)

	return &source, nil
}

// SubSource returns generated.SubSourceResolver implementation.
func (r *Resolver) SubSource() generated.SubSourceResolver { return &subSourceResolver{r} }

type subSourceResolver struct{ *Resolver }
