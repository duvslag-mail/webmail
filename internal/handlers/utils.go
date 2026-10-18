package handlers

import "net/http"

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie("token")
	if err != nil {
		return ""
	}
	return c.Value
}
