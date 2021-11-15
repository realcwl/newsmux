package utils

import (
	"encoding/json"
	"strings"

	"github.com/Luismorlan/newsmux/model"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/pkg/errors"
)

// TODO(jamie): optimize by first parsing json and match later
// TODO(jamie): should probably create a in-memory cache to avoid constant
// parsing the jsonStr into data expression because such kind of parsing is
// expensive.
func DataExpressionMatchPostChain(jsonStr string, rootPost *model.Post) (bool, error) {
	if len(jsonStr) == 0 {
		return true, nil
	}

	var dataExpressionWrap model.DataExpressionWrap
	if err := json.Unmarshal([]byte(jsonStr), &dataExpressionWrap); err != nil {
		Log.Error("data expression can't be unmarshaled to dataExpressionWrap, error :", err)
		return false, err
	}

	matched, err := DataExpressionMatch(dataExpressionWrap, rootPost)
	if err != nil {
		return false, errors.Wrap(err, "data expression match failed")
	}
	if matched {
		return true, nil
	}
	if rootPost.SharedFromPost != nil {
		return DataExpressionMatchPostChain(jsonStr, rootPost.SharedFromPost)
	}
	return false, nil
}

func DataExpressionMatch(dataExpressionWrap model.DataExpressionWrap, post *model.Post) (bool, error) {
	// Empty data expression should match all post.
	if dataExpressionWrap.IsEmpty() {
		return true, nil
	}
	switch expr := dataExpressionWrap.Expr.(type) {
	case model.AllOf:
		if len(expr.AllOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AllOf {
			match, err := DataExpressionMatch(child, post)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case model.AnyOf:
		if len(expr.AnyOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AnyOf {
			match, err := DataExpressionMatch(child, post)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
		return false, nil
	case model.NotTrue:
		match, err := DataExpressionMatch(expr.NotTrue, post)
		if err != nil {
			return false, err
		}
		return !match, nil
	case model.PredicateWrap:
		if expr.Predicate.Type == "LITERAL" {
			return strings.Contains(strings.ToLower(post.Content), strings.ToLower(expr.Predicate.Param.Text)), nil
		}
	default:
		return false, errors.New("unknown node type when matching data expression")
	}
	return false, nil
}
