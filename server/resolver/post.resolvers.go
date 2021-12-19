package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
)

func (r *postResolver) DeletedAt(ctx context.Context, obj *model.Post) (*time.Time, error) {
	return &obj.DeletedAt.Time, nil
}

func (r *postResolver) ImageUrls(ctx context.Context, obj *model.Post) ([]string, error) {
	return obj.ImageUrls, nil
}

func (r *postResolver) FileUrls(ctx context.Context, obj *model.Post) ([]string, error) {
	return obj.FileUrls, nil
}

func (r *postResolver) Tags(ctx context.Context, obj *model.Post) ([]string, error) {
	return strings.Split(obj.Tag, ","), nil
}

// Post returns generated.PostResolver implementation.
func (r *Resolver) Post() generated.PostResolver { return &postResolver{r} }

type postResolver struct{ *Resolver }
