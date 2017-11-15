package cluster

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestApi(t *testing.T) {
	t.Parallel()

	// Start Manager for API
	managerAPI := NewManager("managerAPI", "secret")
	managerAPI.AddClusterNode(Node{name: "managerAPI2", addr: "127.0.0.1:9599"})
	err := managerAPI.ListenAndServe("127.0.0.1:9500")
	if err != nil {
		log.Fatal(err)
	}

	// Start HTTP
	httpAddr := "127.0.0.1:9499"
	srv := startHTTPServer(httpAddr)

	time.Sleep(1 * time.Second)

	// Get requests of public interface
	authKey := apiMakeKey("Test", "secret")
	data, err := getWithKey(authKey, "http://"+httpAddr+"/api/cluster/managerAPI/")
	if err != nil {
		t.Errorf("failed to get /api/cluster/managerAPI/, error:%s", err)
	}
	fmt.Printf("Got public data: %s", string(data))

	// Get requests of private interface without key
	data, err = getWithKey("failme", "http://"+httpAddr+"/api/cluster/managerAPI/admin/")
	if err != nil {
		t.Errorf("failed to get /api/cluster/managerAPI/admin/, error:%s", err)
	}
	fmt.Printf("Got private data: %s", string(data))

	// Get requests of private interface without key
	data, err = getWithKey(authKey, "http://"+httpAddr+"/api/cluster/managerAPI/admin/")
	if err != nil {
		t.Errorf("failed to get /api/cluster/maangerAPI/admin/, error:%s", err)
	}
	fmt.Printf("Got private data: %s", string(data))

	//time.Sleep(20 * time.Second)
	if err := srv.Shutdown(nil); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}

func startHTTPServer(addr string) *http.Server {
	srv := &http.Server{Addr: addr}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			//log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	// returning reference so caller can call Shutdown()
	return srv
}

func getWithKey(authKey, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)

	cookie := http.Cookie{Name: "session", Value: authKey}
	req.AddCookie(&cookie)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
