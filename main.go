package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ppastene/unnofficial-transbank-sdk-go/src/common"
	"github.com/ppastene/unnofficial-transbank-sdk-go/src/webpayplus"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading env file")
	}
}

func main() {
	tokens := []string{}
	var options common.Options
	transaction := webpayplus.NewTransaction(options.ForIntegration(webpayplus.WEBPAY_PLUS_COMMERCE_CODE, webpayplus.WEBPAY_API_KEY))
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", gin.H{})
	})

	router.GET("/", func(c *gin.Context) {
		fmt.Println(c.Request.Host)
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	router.POST("/pagar", func(c *gin.Context) {
		body, ok := c.GetPostForm("amount")
		if !ok || len(body) == 0 {
			c.JSON(http.StatusBadRequest, "badrequest")
			return
		}
		amount, _ := strconv.ParseFloat(body, 64)
		if amount == 0 {
			c.JSON(http.StatusBadRequest, "badrequest")
			return
		}
		t := time.Now().Local()
		order := fmt.Sprintf("order%d%d%d%d%d%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		session := fmt.Sprintf("session%d%d%d%d%d%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		response := transaction.Create(order, session, amount, "http://"+os.Getenv("APP_URL")+"/resumen")
		if len(response.Token) > 0 {
			tokens = append(tokens, response.Token)
		}
		c.HTML(http.StatusOK, "pagar.html", gin.H{
			"token": response.Token,
			"url":   response.Url,
		})
	})

	router.GET("/resumen", func(c *gin.Context) {
		var token_ws, tbk_orden_compra, tbk_id_sesion, estado string
		if len(c.Query("token_ws")) > 0 {
			token_ws = c.Query("token_ws")
			response := transaction.Commit(token_ws)
			switch response.Status {
			case "AUTHORIZED":
				estado = "AUTORIZADA"
			case "FAILED":
				estado = "FALLIDA"
			}
			c.HTML(http.StatusOK, "resumen.html", gin.H{
				"token_ws": token_ws,
				"response": response,
				"estado":   estado,
			})
		} else if len(c.Query("TBK_ORDEN_COMPRA")) > 0 && len(c.Query("TBK_ID_SESION")) > 0 {
			tbk_orden_compra, tbk_id_sesion = c.Query("TBK_ORDEN_COMPRA"), c.Query("TBK_ID_SESION")
			c.HTML(http.StatusOK, "resumen.html", gin.H{
				"tbk_orden_compra": tbk_orden_compra,
				"tbk_id_sesion":    tbk_id_sesion,
				"estado":           "ANULADA POR EL USUARIO",
			})
		}
	})

	router.POST("/anular", func(c *gin.Context) {
		token, _ := c.GetPostForm("token")
		amount, _ := c.GetPostForm("amount")
		if len(amount) == 0 {
			c.JSON(http.StatusBadRequest, "badrequest")
			return
		}
		response := transaction.Refund(token, amount)
		c.HTML(http.StatusOK, "anular.html", gin.H{
			"response": response,
		})
	})

	router.GET("/status", func(c *gin.Context) {
		tokens := tokens
		c.HTML(http.StatusOK, "token-list.html", gin.H{
			"response": tokens,
		})
	})

	router.GET("/status/:token", func(c *gin.Context) {
		tokenParam := c.Param("token")
		for _, token := range tokens {
			if token == tokenParam {
				response := transaction.Status(token)
				c.HTML(http.StatusOK, "status.html", gin.H{
					"response": response,
				})
				return
			}
		}
		c.HTML(http.StatusNotFound, "status.html", gin.H{
			"status":  404,
			"message": "No se encuentra el token",
		})

	})

	router.POST("/resumen", func(c *gin.Context) {
		c.HTML(http.StatusOK, "resumen.html", gin.H{})
	})

	router.Run(os.Getenv("APP_URL"))
}
