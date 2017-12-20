package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/northbright/ming800"
	"github.com/northbright/pathhelper"
	"github.com/northbright/validate"
)

// Config represents the config of this app.
type Config struct {
	ServerURL string `json:"server_url"`
	Company   string `json:"company"`
	User      string `json:"user"`
	Password  string `json:"password"`
}

// MyProcessor implements ming800.WalkProcessor interface to walk ming800.
type MyProcessor struct {
}

// ClassHandler is the handler when a class is found.
func (p *MyProcessor) ClassHandler(class ming800.Class) {}

// StudentHandler is the handler when a student is found.
func (p *MyProcessor) StudentHandler(class ming800.Class, student ming800.Student) {
	if !validate.ValidMobilePhoneNumInChina(student.PhoneNum) {
		row := []string{class.Name, student.Name, student.PhoneNum}
		log.Printf("%s,%s,%s", row[0], row[1], row[2])
	}
}

func main() {
	var (
		err                    error
		buf                    []byte
		currentDir, configFile string
		s                      *ming800.Session
		config                 Config
	)

	defer func() {
		if err != nil {
			log.Printf("%v", err)
		}
	}()

	currentDir, _ = pathhelper.GetCurrentExecDir()
	configFile = path.Join(currentDir, "config.json")

	// Load Conifg
	if buf, err = ioutil.ReadFile(configFile); err != nil {
		err = fmt.Errorf("load config file error: %v", err)
		return
	}

	if err = json.Unmarshal(buf, &config); err != nil {
		err = fmt.Errorf("parse config err: %v", err)
		return
	}

	// New a session
	if s, err = ming800.NewSession(config.ServerURL, config.Company, config.User, config.Password); err != nil {
		err = fmt.Errorf("NewSession() error: %v", err)
		return
	}

	// Login
	if err = s.Login(); err != nil {
		err = fmt.Errorf("Login() error: %v", err)
		return
	}

	// Walk
	processor := &MyProcessor{}
	// Class and student handler will be called while walking ming800.
	if err = s.Walk(processor); err != nil {
		err = fmt.Errorf("Walk() error: %v", err)
		return
	}

	// Logout
	if err = s.Logout(); err != nil {
		err = fmt.Errorf("Logout() error: %v", err)
		return
	}
}
