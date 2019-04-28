package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "github.com/2se/dolphin/mock"
	"github.com/2se/dolphin/routehttp"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestRouteHttpTest(t *testing.T) {
	//run dolphin/cmd/example/server.go fisrt
	routehttp.Start("127.0.0.1:10086")
	appInfo := &routehttp.AppInfo{
		PeerName: "",
		AppName:  "app",
		Address:  "127.0.0.1:16012",
		Methods: []routehttp.MP{
			{"1", "2", "3"},
			{"1", "2", "4"},
			{"1", "2", "5"},
		},
	}
	appJson, err := json.Marshal(appInfo)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(appJson))
	resp, err := http.Post("http://127.0.0.1:10086", "application/json; charset=utf-8", bytes.NewReader(appJson))
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(body))
}
