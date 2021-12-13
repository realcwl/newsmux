package twitter

import (
	"io/ioutil"
	"net/http"

	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
)

func HandleTwitterMessage(c *gin.Context) {
	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, "fail to get request body"+err.Error())
		return
	}
	Logger.Log.Info("result is", string(jsonData))
}
