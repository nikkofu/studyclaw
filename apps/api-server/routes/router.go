package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/internal/app"
	httpapi "github.com/nikkofu/studyclaw/api-server/internal/interfaces/http"
)

func SetupRouter() *gin.Engine {
	return httpapi.SetupRouter(app.NewContainer())
}
