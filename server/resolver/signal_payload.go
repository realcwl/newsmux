package resolver

import (
	"fmt"
	"strings"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
)

type SignalPayload interface {
	Unmarshal(sigPayload string) error
	Marshal() (string, error)
}

type ReadStatusPayload struct {
	delimiter   string
	read        bool
	itemNodeIds []string
	itemType    model.ItemType
}

var _ SignalPayload = &ReadStatusPayload{}

// The unmarshal function is not used in backend but we implement it anyway
func (r *ReadStatusPayload) Unmarshal(sigPayload string) error {
	splits := strings.Split(sigPayload, r.delimiter)
	if len(splits) < 3 {
		return fmt.Errorf("invalid sigPayload: %s", sigPayload)
	}

	if splits[0] != model.ItemTypePost.String() && splits[0] != model.ItemTypeDuplication.String() {
		return fmt.Errorf("invalid sigPayload: %s", sigPayload)
	}
	if splits[1] != utils.RedisTrue && splits[1] != utils.RedisFalse {
		return fmt.Errorf("invalid sigPayload: %s", sigPayload)
	}

	r.itemType = model.ItemType(splits[0])
	r.read = splits[1] == utils.RedisTrue
	r.itemNodeIds = splits[1:]
	return nil
}

func (r *ReadStatusPayload) Marshal() (string, error) {
	res := ""
	res += string(r.itemType) + r.delimiter
	if r.read {
		res += utils.RedisTrue
	} else {
		res += utils.RedisFalse
	}
	for _, pid := range r.itemNodeIds {
		if strings.Contains(pid, r.delimiter) {
			return "", fmt.Errorf("postId conflicts with delimiter: %s, %s", pid, r.delimiter)
		}
		res += r.delimiter + pid
	}
	return res, nil
}
