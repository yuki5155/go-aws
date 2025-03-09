package models

import (
	"strconv"
)

// CallbackRequest represents the expected request with an authorization code
type CallbackRequest struct {
	Code string `json:"code"`
}

// CallbackResponse represents the response structure
type CallbackResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Cookie represents a Set-Cookie header
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite string
}

// ToCookieString converts a Cookie to a Set-Cookie header string
func (c *Cookie) ToCookieString() string {
	cookieStr := c.Name + "=" + c.Value

	if c.HttpOnly {
		cookieStr += "; HttpOnly"
	}
	if c.Secure {
		cookieStr += "; Secure"
	}
	if c.SameSite != "" {
		cookieStr += "; SameSite=" + c.SameSite
	}
	if c.Domain != "" {
		cookieStr += "; Domain=" + c.Domain
	}
	if c.Path != "" {
		cookieStr += "; Path=" + c.Path
	}
	if c.MaxAge > 0 {
		cookieStr += "; Max-Age=" + strconv.Itoa(c.MaxAge)
	}

	return cookieStr
}
