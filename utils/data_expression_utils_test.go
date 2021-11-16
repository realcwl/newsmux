package utils

import (
	"encoding/json"
	"testing"

	"github.com/Luismorlan/newsmux/model"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestDataExpressionUnmarshal(t *testing.T) {
	t.Run("Test unmarshal 1", func(t *testing.T) {
		jsonStr := DataExpressionJsonForTest
		// Check  marshal - unmarshal are consistent
		var dataExpressionWrap model.DataExpressionWrap
		json.Unmarshal([]byte(jsonStr), &dataExpressionWrap)

		bytes, _ := json.Marshal(dataExpressionWrap)
		var newDataExpressionWrap model.DataExpressionWrap

		json.Unmarshal(bytes, &newDataExpressionWrap)

		newBytes, _ := json.Marshal(newDataExpressionWrap)

		require.True(t, cmp.Equal(dataExpressionWrap, newDataExpressionWrap))
		require.Equal(t, bytes, newBytes)
	})
}

func TestDataExpressionMatch(t *testing.T) {
	t.Run("Test matching function", func(t *testing.T) {
		var dataExpressionWrap = model.DataExpressionWrap{
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
		}

		bytes, _ := json.Marshal(dataExpressionWrap)

		var res model.DataExpressionWrap
		json.Unmarshal(bytes, &res)

		matched, err := DataExpressionMatch(res, &model.Post{Content: "马斯克做空以太坊"})

		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res, &model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatch(res, &model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res, &model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})

	t.Run("Test matching from json string", func(t *testing.T) {

		matched, err := DataExpressionMatchPostChain(DataExpressionJsonForTest, &model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPostChain(DataExpressionJsonForTest, &model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPostChain(DataExpressionJsonForTest, &model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPostChain(DataExpressionJsonForTest, &model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPostChain(DataExpressionJsonForTest, &model.Post{Content: "老王做空BITCOIN"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})

	t.Run("Test matching from json string with pure id expression", func(t *testing.T) {
		matched, err := DataExpressionMatchPostChain(PureIdExpressionJson, &model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPostChain(PureIdExpressionJson, &model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPostChain(PureIdExpressionJson, &model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPostChain(PureIdExpressionJson, &model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPostChain(PureIdExpressionJson, &model.Post{Content: "老王做空BITCOIN"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})

	t.Run("Empty expression should match anything", func(t *testing.T) {
		matched, err := DataExpressionMatchPostChain(EmptyExpressionJson, &model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.True(t, matched)

		matched, err = DataExpressionMatchPostChain(EmptyExpressionJson, &model.Post{Content: "随便一个字符串"})
		require.Nil(t, err)
		require.True(t, matched)

		matched, err = DataExpressionMatchPostChain(EmptyExpressionJson, &model.Post{Content: "马云马斯克马克扎克伯格"})
		require.Nil(t, err)
		require.True(t, matched)
	})

	t.Run("Wrong format expression should match nothing and throw", func(t *testing.T) {
		wrongJsonExpression := `
		{
			"id": "1"
		
		`
		matched, err := DataExpressionMatchPostChain(wrongJsonExpression, &model.Post{Content: "马斯克做空以太坊"})
		require.NotNil(t, err)
		require.False(t, matched)

		rightJsonExpressionWrongStructure := `
		{
			"id":"1",
			"expr":{
				"notTrue":{
					"id":"1.1",
					"some_evil_field: "1",
					"expr":{
						"pred":{
							"some_evil_field: "1",
							"type":"LITERAL",
							"param":{
								"text":"马斯克"
							}
						}
					}
				}
			}
		}
		`
		matched, err = DataExpressionMatchPostChain(rightJsonExpressionWrongStructure, &model.Post{Content: "马斯克做空以太坊"})
		require.NotNil(t, err)
		require.False(t, matched)

		rightJsonExpressionWrongType := `
		{
			"id": 1,
		}
		`
		matched, err = DataExpressionMatchPostChain(rightJsonExpressionWrongType, &model.Post{Content: "马斯克做空以太坊"})
		require.NotNil(t, err)
		require.False(t, matched)
	})
}
