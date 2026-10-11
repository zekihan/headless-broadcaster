package web

import (
	"net/http"
	"strings"
)

type AcceptType string

const (
	AcceptTypeJSON  AcceptType = "json"
	AcceptTypeHTML  AcceptType = "html"
	AcceptTypePlain AcceptType = "plain"
	AcceptTypeNone  AcceptType = "none"
)

func GetAcceptType(r *http.Request) AcceptType {
	accept := r.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "application/json"):
		return AcceptTypeJSON
	case strings.Contains(accept, "text/html"):
		return AcceptTypeHTML
	case strings.Contains(accept, "text/plain"):
		return AcceptTypePlain
	}

	return AcceptTypeNone
}
