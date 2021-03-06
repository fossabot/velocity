package rest

import (
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/velocity-ci/velocity/backend/pkg/domain/user"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Username string    `json:"username"`
	Token    string    `json:"token"`
	Expires  time.Time `json:"expires"`
}

var jwtSigningMethod = jwt.SigningMethodHS512
var jwtStandardClaims = &jwt.StandardClaims{
	Issuer: "Velocity CI",
}

func newAuthResponse(u *user.User) *authResponse {
	now := time.Now()
	expires := time.Now().Add(time.Hour * 24 * 2)

	claims := jwtStandardClaims
	claims.ExpiresAt = expires.Unix()
	claims.NotBefore = now.Unix()
	claims.IssuedAt = now.Unix()

	token := jwt.NewWithClaims(jwtSigningMethod, claims)
	tokenString, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	return &authResponse{
		Username: u.Username,
		Token:    tokenString,
		Expires:  expires,
	}
}

type authHandler struct {
	userManager *user.Manager
}

func newAuthHandler(userManager *user.Manager) *authHandler {
	return &authHandler{
		userManager: userManager,
	}
}

func (h *authHandler) create(c echo.Context) error {
	rU := new(authRequest)
	if err := c.Bind(rU); err != nil {
		c.JSON(http.StatusBadRequest, "invalid payload")
		c.Logger().Warn(err)
		return nil
	}

	u, err := h.userManager.GetByUsername(rU.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, nil)
		return nil
	}
	if !u.ValidatePassword(rU.Password) {
		c.JSON(http.StatusUnauthorized, nil)
		return nil
	}
	c.JSON(http.StatusCreated, newAuthResponse(u))
	return nil
}
