package ming800_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/northbright/ming800"
	"github.com/northbright/pathhelper"
)

// MyProcessor implements ming800.Processor interface to walk ming800.
type MyProcessor struct {
}

func (p *MyProcessor) ClassHandler(class *ming800.Class) error {
	log.Printf("class: %v", class)
	return nil
}

func (p *MyProcessor) StudentHandler(class *ming800.Class, student *ming800.Student) error {
	log.Printf("class: %v, student: %v", class, student)
	return nil
}

// Run "go test -c && ./ming800.test" to load config.json and do the test.
func Example() {
	// 1. Create a "config.json" like this to load settings:
	/*{
	        "server_url": "http://localhost:8080",
		"company": "my_company",
		"user": "Frank",
		"password": "my_password"
	}*/

	// 2. Run "go test -c && ./ming800.test" to load config.json and do the test.

	type Config struct {
		ServerURL string `json:"server_url"`
		Company   string `json:"company"`
		User      string `json:"user"`
		Password  string `json:"password"`
	}

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

	log.Printf("Login() successfully.\n")

	// Walk
	// Class and student handler will be called while walking ming800.
	processor := &MyProcessor{}
	if err = s.Walk(processor); err != nil {
		err = fmt.Errorf("Walk() error: %v", err)
		return
	}

	// Logout
	if err = s.Logout(); err != nil {
		err = fmt.Errorf("Logout() error: %v", err)
		return
	}

	log.Printf("logout() successfully.\n")
	// Output:
}
