package rest

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/velocity-ci/velocity/backend/pkg/domain"
)

func getPagingQueryParams(c echo.Context) *domain.PagingQuery {
	pQ := domain.NewPagingQuery()
	if err := c.Bind(pQ); err != nil {
		c.JSON(http.StatusBadRequest, "invalid parameters")
		return nil
	}

	return pQ
}
