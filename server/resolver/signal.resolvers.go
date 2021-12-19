package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
)

func (r *signalResolver) SignalPayload(ctx context.Context, obj *model.Signal) (*string, error) {
	if obj.Payload == "" {
		return nil, nil
	}
	return &obj.Payload, nil
}

// Signal returns generated.SignalResolver implementation.
func (r *Resolver) Signal() generated.SignalResolver { return &signalResolver{r} }

type signalResolver struct{ *Resolver }
