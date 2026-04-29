package route

import (
	"github.com/ServalNeko/sendgrid-dev/api/v3/mail/send"
	"github.com/labstack/echo"
)

func Init() *echo.Echo {
	e := echo.New()

	// Routes
	v3 := e.Group("/v3/mail")
	{
		v3.GET("/send", send.GetSend())
		v3.POST("/send", send.PostSend())
	}

	return e
}
