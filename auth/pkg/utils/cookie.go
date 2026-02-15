package utils

import "github.com/gin-gonic/gin"

func SetAuthCookies(c *gin.Context, access, refresh string) {
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)
}

func ClearAuthCookies(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
}
