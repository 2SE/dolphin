package core

import (
	"errors"
	"github.com/2se/dolphin/config"
)

var AccCheck *AccountCheck

var (
	ErrCheckFirst = errors.New("need login/register first")
)

//账户体系，不用接口，这里用接口会带来很多对象开销
type AccountCheck struct {
	check    bool
	LoginMP  MethodPath
	SendCode MethodPath
}

func InitAccountCheck(loginCnf, sendCodeCnf *config.MethodPathConfig) {
	AccCheck = &AccountCheck{
		LoginMP:  NewMethodPath(loginCnf.Version, loginCnf.Resource, loginCnf.Action),
		SendCode: NewMethodPath(sendCodeCnf.Version, sendCodeCnf.Resource, sendCodeCnf.Action),
		check:    true,
	}
}

func (ac *AccountCheck) NeedCheck() bool {
	return ac.check
}
func (ac *AccountCheck) CheckFirst(mp MethodPath) error {
	if mp.String() != ac.LoginMP.String() || mp.String() != ac.SendCode.String() {
		return ErrCheckFirst
	}
	return nil
}

func (ac *AccountCheck) CheckLogin(mp MethodPath) bool {
	return mp.String() == ac.LoginMP.String()
}
