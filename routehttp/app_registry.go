package routehttp

import (
	"encoding/json"
	"fmt"
	"github.com/2se/dolphin/route"
	"io/ioutil"
	"log"
	"net/http"
)

type AppInfo struct {
	AppName string
	Address string
	Methods []MethodPath
}
type MethodPath struct {
	Reversion byte
	Resource  byte
	Action    byte
}

func RegistryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var appInfo *AppInfo
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &appInfo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	mps := make([]route.MethodPath, len(appInfo.Methods))
	for _, v := range appInfo.Methods {
		mps = append(mps, route.MethodPath{v.Reversion, v.Resource, v.Action})
	}

	err = route.GetRouterInstance().Register(mps, appInfo.AppName, "", appInfo.Address)
	w.WriteHeader(http.StatusOK)
	if err != nil {
		w.Write([]byte(err.Error()))
	}
}

func Start(address string) {
	http.HandleFunc("/", RegistryHandler)
	fmt.Println("servers start")
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
