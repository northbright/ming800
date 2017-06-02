package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/northbright/ming800"
)

func showCategories(c *gin.Context) {
	var err error
	var s *ming800.Session
	var categories []ming800.Category

	defer func() {
		if err != nil {
			log.Printf("home() err: %v\n", err)
		}
	}()

	if s, err = ming800.NewSession(config.ServerURL, config.Company, config.User, config.Password); err != nil {
		err = fmt.Errorf("create new session err: %v\n", err)
		return
	}

	// Login
	if err = s.Login(); err != nil {
		err = fmt.Errorf("login err: %v\n", err)
		return
	}
	defer s.Logout()

	// Get current categories.
	if categories, _, err = s.GetCurrentCategoriesAndClasses(); err != nil {
		err = fmt.Errorf("get current categories err: %v\n", err)
		return
	}

	c.HTML(http.StatusOK, "categories.tmpl", gin.H{
		"title":      "",
		"msg":        "",
		"categories": categories,
	})
}

func showClassesOfCategory(c *gin.Context) {
	var err error
	var s *ming800.Session
	var id string
	var classes []ming800.Class
	var matchedClasses []ming800.Class

	defer func() {
		if err != nil {
			log.Printf("home() err: %v\n", err)
		}
	}()

	id = c.Param("id")
	if id == "" {
		err = fmt.Errorf("empty category ID.")
		return
	}

	if s, err = ming800.NewSession(config.ServerURL, config.Company, config.User, config.Password); err != nil {
		err = fmt.Errorf("create new session err: %v\n", err)
		return
	}

	// Login
	if err = s.Login(); err != nil {
		err = fmt.Errorf("login err: %v\n", err)
		return
	}
	defer s.Logout()

	// Get current classes.
	if _, classes, err = s.GetCurrentCategoriesAndClasses(); err != nil {
		err = fmt.Errorf("get current categories and classes err: %v\n", err)
		return
	}

	for _, class := range classes {
		if class.CategoryId == id {
			matchedClasses = append(matchedClasses, class)
		}
	}

	c.HTML(http.StatusOK, "classes.tmpl", gin.H{
		"title":   "",
		"msg":     "",
		"classes": matchedClasses,
	})
}

func showStudentsOfClass(c *gin.Context) {
	var err error
	var s *ming800.Session
	var id string
	var students []ming800.Student

	defer func() {
		if err != nil {
			log.Printf("home() err: %v\n", err)
		}
	}()

	id = c.Param("id")
	if id == "" {
		err = fmt.Errorf("empty class instance ID.")
		return
	}

	if s, err = ming800.NewSession(config.ServerURL, config.Company, config.User, config.Password); err != nil {
		err = fmt.Errorf("create new session err: %v\n", err)
		return
	}

	// Login
	if err = s.Login(); err != nil {
		err = fmt.Errorf("login err: %v\n", err)
		return
	}
	defer s.Logout()

	// Get students of the class
	if students, err = s.GetStudentsOfClass(id); err != nil {
		err = fmt.Errorf("get students of class(instance id: %v) err: %v\n", id, err)
		return
	}

	c.HTML(http.StatusOK, "students.tmpl", gin.H{
		"title":    "",
		"msg":      "",
		"students": students,
	})
}
