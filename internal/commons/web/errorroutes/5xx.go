package errorroutes

import (
	"net/http"
)

func Get5xx(w http.ResponseWriter, r *http.Request) {
	SendErrorMessage(w, r, http.StatusInternalServerError)
}
