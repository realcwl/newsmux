package twitter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleTwitterCRC(t *testing.T) {
	router := gin.Default()
	router.GET("/crc", HandleTwitterCRC)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/crc?crc_token=abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"response_token":"d89f5d4bb3f3f37ae72ad1b75c03a0312e1e34987fa313fc2bc4c7a413452a31"}`, w.Body.String())
}
