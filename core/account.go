package core

import (
	"errors"
	"github.com/2se/dolphin/config"
)

var ReqCheck = &RequestCheck{}

var (
	ErrCheckFirst = errors.New("need login/register first")
)

//账户体系，不用接口，这里用接口会带来很多对象开销
type RequestCheck struct {
	check     bool
	LoginMP   MethodPath
	WhiteList []MethodPath
}

func InitRequestCheck(loginCnf *config.MethodPathConfig, whiteList []*config.MethodPathConfig) {
	ReqCheck = &RequestCheck{
		LoginMP:   NewMethodPath(loginCnf.Version, loginCnf.Resource, loginCnf.Action),
		check:     true,
		WhiteList: make([]MethodPath, len(whiteList)),
	}
	for k, v := range whiteList {
		ReqCheck.WhiteList[k] = NewMethodPath(v.Version, v.Resource, v.Action)
	}
}

func (rc *RequestCheck) NeedCheck() bool {
	return rc.check
}
func (rc *RequestCheck) CheckFirst(mp MethodPath) error {
	if mp.String() == rc.LoginMP.String() {
		return nil
	}
	for _, v := range rc.WhiteList {
		if v.String() == mp.String() {
			return nil
		}
	}
	return ErrCheckFirst
}

func (rc *RequestCheck) CheckLogin(mp MethodPath) bool {
	return mp.String() == rc.LoginMP.String()
}
