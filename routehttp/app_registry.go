package routehttp

import (
	"encoding/json"
	"fmt"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"io/ioutil"
	"net/http"
)

type AppInfo struct {
	PeerName string
	AppName  string
	Address  string
	Methods  []MP
}
type MP struct {
	Reversion string
	Resource  string
	Action    string
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

	mps := make([]core.MethodPath, len(appInfo.Methods))
	for _, v := range appInfo.Methods {
		mps = append(mps, core.NewMethodPath(v.Reversion, v.Resource, v.Action))
	}

	pr := core.NewPeerRouter(appInfo.AppName, "")
	err = router.Register(mps, pr, appInfo.Address)
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
		panic(fmt.Errorf("ListenAndServe: %v", err))
	}
}
