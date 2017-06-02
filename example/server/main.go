package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/northbright/pathhelper"
)

var (
	redisAddr     = ":6379"
	redisPassword = ""
	serverRoot    = ""
	templatesPath = ""
	staticPath    = ""
	configFile    = ""
	config        Config
)

type Config struct {
	ServerURL string `json:"server_url"`
	Company   string `json:"company"`
	User      string `json:"user"`
	Password  string `json:"password"`
}

func init() {
	serverRoot, _ = pathhelper.GetCurrentExecDir()
	templatesPath = path.Join(serverRoot, "templates")
	staticPath = path.Join(serverRoot, "static")
	configFile = path.Join(serverRoot, "config.json")
}

func loadConfig(f string) (Config, error) {
	var buf []byte
	var err error

	c := Config{}

	// Load Conifg
	if buf, err = ioutil.ReadFile(f); err != nil {
		return c, err
	}

	if err = json.Unmarshal(buf, &c); err != nil {
		return c, err
	}

	return c, err
}

func main() {
	var err error

	defer func() {
		if err != nil {
			log.Printf("main() err: %v\n", err)
		}
	}()

	if config, err = loadConfig(configFile); err != nil {
		err = fmt.Errorf("loadConfig() err: %v", err)
		return
	}

	r := gin.Default()

	// Serve Static files.
	r.Static("/static/", staticPath)

	// Load Templates.
	r.LoadHTMLGlob(fmt.Sprintf("%v/*", templatesPath))

	// Pages
	r.GET("/", showCategories)
	r.GET("/category/:id/classes", showClassesOfCategory)
	r.GET("/class/:id/students", showStudentsOfClass)

	r.Run(":80")
}
