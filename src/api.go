package cluster

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	apiSalt           = "x0I3naOlm"
	apiLoginURL       = "/login"
	apiLoginValidTime = 1 * time.Hour
)

// Authentication middleware
type apiAuthentication struct {
	wrappedHandler http.Handler
	authKey        string
}

type apiMessage struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Data    interface{} `json:"data"`
}

func (h apiAuthentication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session")
	if cookie != nil {
		// get username
		sc := strings.Split(cookie.Value, ".")
		if len(sc) < 2 { // invalid contents
			http.Redirect(w, r, "/login?return="+r.URL.Path, 302)
			return
		}
		username := sc[len(sc)-2]
		epochString := sc[len(sc)-1]
		epoch, err := strconv.ParseInt(epochString, 10, 64)
		if err != nil {
			http.Redirect(w, r, apiLoginURL+"?return="+r.URL.Path, 302)
			return
		}
		// recreate key to see if its valid
		authSha := apiMakeKey(username, h.authKey, epoch)
		// is auth successfull?
		if cookie.Value == authSha && time.Now().Unix() < epoch {
			h.wrappedHandler.ServeHTTP(w, r)
			return
		}
	}
	http.Redirect(w, r, apiLoginURL+"?return="+r.URL.Path, 302)
}

// Authenticate user
func authenticate(h http.Handler, authKey string) apiAuthentication {
	return apiAuthentication{h, authKey}
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))

}

func apiMakeKey(username, key string, epoch int64) string {
	if epoch == 0 {
		epoch = time.Now().Add(apiLoginValidTime).Unix()
	}
	hash := sha512.New()
	hash.Write([]byte(fmt.Sprintf("%s%s%s%d", apiSalt, key, username, epoch)))
	return fmt.Sprintf("%x.%s.%d", hash.Sum(nil), username, epoch)
}

func apiWriteData(w http.ResponseWriter, statusCode int, message apiMessage) {
	w.WriteHeader(statusCode)
	messageData, err := json.Marshal(message.Data)
	message.Data = string(messageData)
	data, err := json.Marshal(message)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Failed to encode json on write"))
	}
	data = append(data, 10) // 10 = newline
	fmt.Printf("Output: %+v", string(data))
	w.Write(data)

}
