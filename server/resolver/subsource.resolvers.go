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
	return &obj.DeletedAt.Time, nil
}

func (r *subSourceResolver) Source(ctx context.Context, obj *model.SubSource) (*model.Source, error) {
	var source model.Source
	r.DB.Where("id = ?", obj.SourceID).First(&source)

	return &source, nil
}

// SubSource returns generated.SubSourceResolver implementation.
func (r *Resolver) SubSource() generated.SubSourceResolver { return &subSourceResolver{r} }

type subSourceResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *subSourceResolver) IconURL(ctx context.Context, obj *model.SubSource) (*string, error) {
	panic(fmt.Errorf("not implemented"))
}
