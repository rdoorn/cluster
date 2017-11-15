package cluster

import (
	"crypto/sha512"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const loginPage = "<html><head><title>Login</title></head><body><form action=\"/login/managerFIVE\" method=\"post\"> <input type=\"username\" name=\"username\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"

type apiClusterHandler struct {
}

type apiClusterAdminHandler struct {
}

type apiLogin struct {
	authKey string
}

// Authentication middleware
type apiAuthentication struct {
	wrappedHandler http.Handler
	authKey        string
	loginURL       string
}

func (h apiAuthentication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			http.Redirect(w, r, h.loginURL+"?return="+r.URL.Path, 301)
			//fmt.Fprint(w, err)
			//fmt.Fprint(w, loginPage)

			return
		}
	}
	if cookie != nil {
		// get username
		sc := strings.Split(cookie.Value, ".")
		username := sc[len(sc)-1]

		// recreate key to see if its valid
		authSha := apiMakeKey(username, h.authKey)

		if cookie.Value == authSha {
			h.wrappedHandler.ServeHTTP(w, r)
			return
		}
	}
	http.Redirect(w, r, h.loginURL+"?return="+r.URL.Path, 302)
	//fmt.Fprint(w, loginPage)
	//fmt.Printf("session cookie is set to: %s\n", cookie)
}

// Authenticate user
func authenticate(h http.Handler, authKey string, loginURL string) apiAuthentication {
	return apiAuthentication{h, authKey, loginURL}
}

// pro
func (h apiClusterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World!!!")
}

func (h apiClusterAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))

}

func (h apiLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Fprint(w, loginPage)
		//fmt.Fprint(w, err)
		return
	}
	cookie := &http.Cookie{
		Name:    "session",
		Value:   apiMakeKey(r.FormValue("username"), h.authKey),
		Path:    "/",
		Expires: time.Now().Add(1 * time.Hour),
	}
	http.SetCookie(w, cookie)
}

func apiMakeKey(username, key string) string {
	hash := sha512.New()
	hash.Write([]byte(fmt.Sprintf("%s%s", key, username)))
	return fmt.Sprintf("%x.%s", hash.Sum(nil), username)
}
