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

func (r *sourceResolver) DeletedAt(ctx context.Context, obj *model.Source) (*time.Time, error) {
	panic(fmt.Errorf("not implemented"))
}

// Source returns generated.SourceResolver implementation.
func (r *Resolver) Source() generated.SourceResolver { return &sourceResolver{r} }

type sourceResolver struct{ *Resolver }
