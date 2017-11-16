package cluster

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type apiClusterAdminHandler struct {
	manager *Manager
}

func (h apiClusterAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))
}
