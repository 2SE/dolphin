package test

import (
	"fmt"
	"github.com/2se/dolphin/route"
	"github.com/2se/dolphin/routehttp"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestRouteHttpStart(t *testing.T) {
	route.InitRoute("123")
	routehttp.Start("127.0.0.1:10086")
}

func TestRouteHttpTest(t *testing.T) {
	client := &http.Client{}

	appInfo := &routehttp.AppInfo{
		AppName: "testApp",
		Address: "127.0.0.1:10087",
		Methods: []routehttp.MethodPath{
			{1, 2, 3},
			{1, 2, 4},
			{1, 2, 5},
		},
	}
	type MethodPath struct {
		Reversion byte
		Resource  byte
		Action    byte
	}
	req, err := http.NewRequest("POST", "127.0.0.1:10086", strings.NewReader("name=cjb"))
	if err != nil {
		// handle error
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "name=anny")

	resp, err := client.Do(req)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	fmt.Println(string(body))
}
