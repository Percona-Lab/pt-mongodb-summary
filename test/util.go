package test

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"strings"
)

func RootDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.Replace(string(out), "\n", "", -1), nil
}

func LoadJson(filename string, destination interface{}) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dat, &destination)
	if err != nil {
		return err
	}

	return nil
}
