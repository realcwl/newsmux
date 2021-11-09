package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
)

func (r *subSourceResolver) DeletedAt(ctx context.Context, obj *model.SubSource) (*time.Time, error) {
	return &obj.DeletedAt.Time, nil
}

func (r *subSourceResolver) Source(ctx context.Context, obj *model.SubSource) (*model.Source, error) {
	var source model.Source
	// all calls into this resolver only need id, which is already in SubSource
	// this is a perf optimization, current fix is a hot fix
	// TODO: In UI, if we request subsource -> source -> id, alernatively we can request subsource -> source_id
	//       to save query into DB (which is a lot and making DB out of connections)
	source.Id = obj.SourceID
	// r.DB.Where("id = ?", obj.SourceID).First(&source)
	return &source, nil
}

// SubSource returns generated.SubSourceResolver implementation.
func (r *Resolver) SubSource() generated.SubSourceResolver { return &subSourceResolver{r} }

type subSourceResolver struct{ *Resolver }
