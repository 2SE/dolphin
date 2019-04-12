package config

import (
	"errors"
	"github.com/BurntSushi/toml"
	"os"
	"path"
	"path/filepath"
)

const (
	configFileName = "config.toml"
	homeConfigPath = ".config/dolphin.d"
	etcConfigPath  = "/etc/dolphin.d"
)

var (
	ConfFileNotFound = errors.New("config file not found")
)

// Load 载入dolphin配置文件
// 1. 用户指定路径，如果传入的配置文件为空，则按照以下顺序查找配置文件
// 2. 查询应用程序同路径下是否存在配置文件
// 3. 查询$HOME/.config/dolphin.d/config.toml
// 4. 查询/etc/dolphin.d/config.toml
func Load(filePath string) (*Config, error) {
	var (
		err error
		cnf Config
	)

	if filePath, err = findPath(filePath); err != nil {
		return nil, err
	}

	if _, err := toml.DecodeFile(filePath, &cnf); err != nil {
		return nil, err
	}

	return &cnf, nil
}

// 1. 用户指定路径
// 2. 查询应用程序同路径下是否存在配置文件
// 3. 查询$HOME/.config/dolphin.d/config.toml
// 4. 查询/etc/dolphin.d/config.toml
func findPath(givenPath string) (string, error) {
	if len(givenPath) > 0 {
		return givenPath, nil
	}

	var err error

	if givenPath, err = findInExecPath(); err != nil {
		return "", err
	} else if givenPath != "" {
		return givenPath, nil
	}

	if givenPath, err = findInHome(); err != nil {
		return "", err
	} else if givenPath != "" {
		return givenPath, nil
	}

	if givenPath, err = findInEtc(); err != nil {
		return "", err
	} else if givenPath != "" {
		return givenPath, nil
	}

	return "", ConfFileNotFound
}

func findInExecPath() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	givenPath := path.Join(dir, configFileName)
	if exist, err := checkExist(givenPath); exist {
		return givenPath, nil
	} else {
		return "", err
	}
}

func findInHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	givenPath := path.Join(home, homeConfigPath, configFileName)
	if exist, err := checkExist(givenPath); exist {
		return givenPath, nil
	} else {
		return "", err
	}
}

func findInEtc() (string, error) {
	givenPath := path.Join(etcConfigPath, configFileName)
	if exist, err := checkExist(givenPath); exist {
		return givenPath, nil
	} else {
		return "", err
	}
}

func checkExist(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
