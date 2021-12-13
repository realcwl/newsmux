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
	// TODO(chenweilunster): Move this APP_SECRET to parameter store
	APP_SECRET     = "xjsunVSsgYSS0n4Bm7Ai9ZvorzeuT7SkRAFPscC7N9arVzliGJ"
	CRC_TOKEN      = "crc_token"
	RESPONSE_TOKEN = "response_token"
)

// Encode the challenge using HMAC SHA256 with incoming token and APP_SECRET
func HandleTwitterCRC(c *gin.Context) {
	token := c.Query(CRC_TOKEN)
	h := hmac.New(sha256.New, []byte(APP_SECRET))
	h.Write([]byte(token))
	sha := hex.EncodeToString(h.Sum(nil))
	c.JSON(http.StatusOK, gin.H{RESPONSE_TOKEN: sha})
}
