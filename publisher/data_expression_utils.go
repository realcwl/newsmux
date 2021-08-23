package publisher

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Luismorlan/newsmux/model"
)

// TODO(jamie): optimize by first parsing json and match later
func DataExpressionJsonMatch(jsonStr string, content string) (bool, error) {
	var res model.DataExpressionRoot
	json.Unmarshal([]byte(jsonStr), &res)
	return DataExpressionMatch(res.Root, content)
}

func DataExpressionMatch(node model.DataExpressionWrap, content string) (bool, error) {
	switch expr := node.Expr.(type) {
	case model.AllOf:
		if len(expr.AllOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AllOf {
			match, err := DataExpressionMatch(child, content)
			if err != nil {
				return false, err
			}
			if match == false {
				return false, nil
			}
		}
		return true, nil
	case model.AnyOf:
		if len(expr.AnyOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AnyOf {
			match, err := DataExpressionMatch(child, content)
			if err != nil {
				return false, err
			}
			if match == true {
				return true, nil
			}
		}
		return false, nil
	case model.NotTrue:
		match, err := DataExpressionMatch(expr.NotTrue, content)
		if err != nil {
			return false, err
		}
		return !match, nil
	case model.PredicateWrap:
		if expr.Predicate.Type == "LITERAL" {
			return strings.Contains(content, expr.Predicate.Param.Text), nil
		}
	default:
		return false, errors.New("unknown node type when matching data expression")
	}
	return false, nil
}
