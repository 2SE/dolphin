package routehttp

import (
	"encoding/json"
	"fmt"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/route"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"net/http"
)

type AppInfo struct {
	AppName string
	Address string
	Methods []MP
}
type MP struct {
	Reversion string
	Resource  string
	Action    string
}

func RegistryHandler(localPeerName string, boardcast func(proto.Message)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		mps := make([]common.MethodPath, len(appInfo.Methods))
		for _, v := range appInfo.Methods {
			mps = append(mps, common.NewMethodPath(v.Reversion, v.Resource, v.Action))
		}

		err = route.GetRouterInstance().Register(mps, appInfo.AppName, localPeerName, appInfo.Address)
		w.WriteHeader(http.StatusOK)
		if err != nil {
			w.Write([]byte(err.Error()))
		}
	}
}

func Start(localPeerName, address string, boardcast func(proto.Message)) {
	http.HandleFunc("/", RegistryHandler(localPeerName, boardcast))
	fmt.Println("servers start")
	err := http.ListenAndServe(address, nil)
	if err != nil {
		panic(fmt.Errorf("ListenAndServe: ", err))
	}
}
