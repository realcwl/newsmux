// Implement Twitter CRC challenge according to
// https://developer.twitter.com/en/docs/twitter-api/enterprise/account-activity-api/guides/securing-webhooks
package twitter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// TODO(chenweilunster): Move this AppSecret to parameter store
	AppSecret     = "xjsunVSsgYSS0n4Bm7Ai9ZvorzeuT7SkRAFPscC7N9arVzliGJ"
	CrcToken      = "crc_token"
	ResponseToken = "response_token"
)

// Encode the challenge using HMAC SHA256 with incoming token and AppSecret
func HandleTwitterCRC(c *gin.Context) {
	token := c.Query(CrcToken)
	h := hmac.New(sha256.New, []byte(AppSecret))
	h.Write([]byte(token))
	sha := hex.EncodeToString(h.Sum(nil))
	c.JSON(http.StatusOK, gin.H{ResponseToken: sha})
}
