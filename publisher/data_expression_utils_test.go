package publisher

import (
	"encoding/json"
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
	`

	jsonWithEmptyExpression = `
	{
		"id": "1"
	}
	`

	jsonWithPureIdExpression = `
	{
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
							},
							{
								"id": "1.1.3",
								"expr": {
									"notTrue": {
										"id": "1.1.3.1"
									}
								}
							},
							{
								"id": "1.1.4"
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
				}, 
				{
					"id": "1.3"
				}
			]
		}
	}
	`
)

func TestDataExpressionUnmarshal(t *testing.T) {
	t.Run("Test unmarshal 1", func(t *testing.T) {
		jsonStr := jsonStringForTest
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

		matched, err := DataExpressionMatch(res, model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res, model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatch(res, model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatch(res, model.Post{Content: "老王做空bitcoin"})
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

	t.Run("Test matching from json string with pure id expression", func(t *testing.T) {
		matched, err := DataExpressionMatchPost(jsonWithPureIdExpression, model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPost(jsonWithPureIdExpression, model.Post{Content: "老王做空以太坊"})
		require.Nil(t, err)
		require.Equal(t, true, matched)

		matched, err = DataExpressionMatchPost(jsonWithPureIdExpression, model.Post{Content: "老王做空比特币"})
		require.Nil(t, err)
		require.Equal(t, false, matched)

		matched, err = DataExpressionMatchPost(jsonWithPureIdExpression, model.Post{Content: "老王做空bitcoin"})
		require.Nil(t, err)
		require.Equal(t, true, matched)
	})

	t.Run("Empty expression should match anything", func(t *testing.T) {
		matched, err := DataExpressionMatchPost(jsonWithEmptyExpression, model.Post{Content: "马斯克做空以太坊"})
		require.Nil(t, err)
		require.True(t, matched)

		matched, err = DataExpressionMatchPost(jsonWithEmptyExpression, model.Post{Content: "随便一个字符串"})
		require.Nil(t, err)
		require.True(t, matched)

		matched, err = DataExpressionMatchPost(jsonWithEmptyExpression, model.Post{Content: "马云马斯克马克扎克伯格"})
		require.Nil(t, err)
		require.True(t, matched)
	})
}
