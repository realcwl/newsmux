package publisher

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Luismorlan/newsmux/model"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

const (
	// develop based on this json structure, if there is any changes in json structure
	// please also change this in order to keep unit test still alive
	jsonStringForTest = `
	{
		"dataExpression":{
		   "id":"1",
		   "expr":{
			  "allOf":[
				 {
					"id":"1.1",
					"expr":{
					   "anyOf":[
						  {
							 "id":"1.1.1",
							 "expr":{
								"pred":{
								   "type":"LITERAL",
								   "param":{
									  "text":"bitcoin"
								   }
								}
							 }
						  },
						  {
							 "id":"1.1.2",
							 "expr":{
								"pred":{
								   "type":"LITERAL",
								   "param":{
									  "text":"以太坊"
								   }
								}
							 }
						  }
					   ]
					}
				 },
				 {
					"id":"1.2",
					"expr":{
					   "notTrue":{
						  "id":"1.2.1",
						  "expr":{
							 "pred":{
								"type":"LITERAL",
								"param":{
								   "text":"马斯克"
								}
							 }
						  }
					   }
					}
				 }
			  ]
		   }
		}
	 }
	`
)

func TestDataExpressionUnmarshal(t *testing.T) {
	t.Run("Test unmarshal 1", func(t *testing.T) {
		jsonStr := jsonStringForTest
		// Check  marshal - unmarshal are consistent
		var root model.DataExpressionRoot
		json.Unmarshal([]byte(jsonStr), &root)
		fmt.Printf("%+v\n", root)

		bytes, _ := json.Marshal(root)
		var newRoot model.DataExpressionRoot

		json.Unmarshal(bytes, &newRoot)
		fmt.Printf("%+v\n", newRoot)

		newBytes, _ := json.Marshal(root)

		require.True(t, cmp.Equal(root, newRoot))
		require.Equal(t, bytes, newBytes)
	})
}

func TestDataExpressionMatch(t *testing.T) {
	t.Run("Test matching function", func(t *testing.T) {
		var root = model.DataExpressionRoot{
			Root: model.DataExpressionWrap{
				ID: "1",
				Expr: model.AllOf{
					AllOf: []model.DataExpressionWrap{
						{
							ID: "1.1",
							Expr: model.AnyOf{
								AnyOf: []model.DataExpressionWrap{
									{
										ID: "1.1.1",
										Expr: model.PredicateWrap{
											Predicate: model.Predicate{
												Type:  "LITERAL",
												Param: model.Literal{"bitcoin"},
											},
										},
									},
									{
										ID: "1.1.2",
										Expr: model.PredicateWrap{
											Predicate: model.Predicate{
												Type:  "LITERAL",
												Param: model.Literal{"以太坊"},
											},
										},
									},
								},
							},
						},
						{
							ID: "1.2",
							Expr: model.NotTrue{
								NotTrue: model.DataExpressionWrap{
									ID: "1.2.1",
									Expr: model.PredicateWrap{
										Predicate: model.Predicate{
											Type:  "LITERAL",
											Param: model.Literal{"马斯克"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		bytes, _ := json.Marshal(root)

		var res model.DataExpressionRoot
		json.Unmarshal(bytes, &res)

		fmt.Printf("%+v \n", res.Root.Expr)

		bytes2, _ := json.Marshal(res)
		fmt.Println(string(bytes2))

		matched, err := DataExpressionMatch(res.Root, model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res.Root, model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatch(res.Root, model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res.Root, model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})

	t.Run("Test matching from json string", func(t *testing.T) {

		matched, err := DataExpressionMatchPost(jsonStringForTest, model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPost(jsonStringForTest, model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPost(jsonStringForTest, model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPost(jsonStringForTest, model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})
}
