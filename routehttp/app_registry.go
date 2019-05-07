package routehttp

import (
	"encoding/json"
	"fmt"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"io/ioutil"
	"net/http"
	"time"
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

	mps := make([]core.MethodPath, 0, len(appInfo.Methods))
	for _, v := range appInfo.Methods {
		mps = append(mps, core.NewMethodPath(v.Reversion, v.Resource, v.Action))
	}

	pr := core.NewPeerRouter("", appInfo.AppName)
	err = router.Register(mps, pr, appInfo.Address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Registered successfully!!!"))
	}
}

func Start(address string) {
	http.HandleFunc("/", RegistryHandler)
	fmt.Println("servers start")
	server := &http.Server{
		Addr:         address,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("ListenAndServe: %v", err))
	}
}
