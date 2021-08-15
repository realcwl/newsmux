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

func (r *postResolver) DeletedAt(ctx context.Context, obj *model.Post) (*time.Time, error) {
	return &obj.DeletedAt.Time, nil
}

func (r *postResolver) ImageUrls(ctx context.Context, obj *model.Post) ([]string, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *postResolver) FileUrls(ctx context.Context, obj *model.Post) ([]string, error) {
	panic(fmt.Errorf("not implemented"))
}

// Post returns generated.PostResolver implementation.
func (r *Resolver) Post() generated.PostResolver { return &postResolver{r} }

type postResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *postResolver) CrawledAt(ctx context.Context, obj *model.Post) (*time.Time, error) {
	panic(fmt.Errorf("not implemented"))
}
func (r *postResolver) ContentGeneratedAt(ctx context.Context, obj *model.Post) (*time.Time, error) {
	panic(fmt.Errorf("not implemented"))
}
func (r *postResolver) OriginURL(ctx context.Context, obj *model.Post) (*string, error) {
	panic(fmt.Errorf("not implemented"))
}
