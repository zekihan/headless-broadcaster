package errorroutes

import (
	"net/http"
)

func GetNotFound(w http.ResponseWriter, r *http.Request) {
	SendErrorMessage(w, r, http.StatusNotFound)
}
