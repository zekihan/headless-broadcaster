package ping

import (
	"net/http"

	"github.com/zekihan/headless-broadcaster/internal/commons/web"
)

func GetPing(w http.ResponseWriter, r *http.Request) {
	web.WriteString(r.Context(), w, "Pong\n")
}
