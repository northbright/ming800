# ming800

[![Build Status](https://travis-ci.org/northbright/ming800.svg?branch=master)](https://travis-ci.org/northbright/ming800)
[![Go Report Card](https://goreportcard.com/badge/github.com/northbright/ming800)](https://goreportcard.com/report/github.com/northbright/ming800)
[![GoDoc](https://godoc.org/github.com/northbright/ming800?status.svg)](https://godoc.org/github.com/northbright/ming800)

ming800是一个[Golang](https://golang.org)包，提供适用于旧单机版本明日管理系统的API接口。

#### 适用版本
* 旧版单机安装版本（2012年）
* 只有1个校区（总部）

#### 工作方式
* 抓取网页结果，并且使用正则表达式得到数据。

#### 功能
* 获取当前所有专业与开设班级，以及每个班级的学生信息。

#### 例子（迭代ming800的所有年级，班级，学生信息）

        // New a session
        if s, err = ming800.NewSession(ServerURL, Company, User, Password); err != nil {
                err = fmt.Errorf("NewSession() error: %v", err)
                return
        }

        // Login
        if err = s.Login(); err != nil {
                err = fmt.Errorf("Login() error: %v", err)
                return
        }

        // Walk
        // Write your own class and student handler functions.
        classHandler := func(class ming800.Class) {
                log.Printf("class: %v", class)
        }

        studentHandler := func(class ming800.Class, student ming800.Student) {
                log.Printf("class: %v, student: %v", class, student)
        }

        // Class and student handler will be called while walking ming800.
        if err = s.Walk(classHandler, studentHandler); err != nil {
                err = fmt.Errorf("Walk() error: %v", err)
                return
        }

        // Logout
        if err = s.Logout(); err != nil {
                err = fmt.Errorf("Logout() error: %v", err)
                return
        }
 

#### 文档
* [API References](https://godoc.org/github.com/northbright/ming800)

#### License
* [MIT License](LICENSE)
