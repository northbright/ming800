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
* 迭代当前所有专业与开设班级，以及每个班级的学生信息。

#### 例子：迭代ming800的所有年级，班级，学生信息

        // MyProcessor implements ming800.Processor interface to walk ming800.
        type MyProcessor struct {
        }

        func (p *MyProcessor) ClassHandler(class ming800.Class) {
                log.Printf("class: %v", class)
        }

        func (p *MyProcessor) StudentHandler(class ming800.Class, student ming800.Student) {
                log.Printf("class: %v, student: %v", class, student)
        }

        // New a session
        s, _ := ming800.NewSession(ServerURL, Company, User, Password); err != nil {

        // Login
        s.Login()

        // Walk
        processor := &MyProcessor{}
        s.Walk(processor)

        // Logout
        s.Logout()

#### 文档
* [API References](https://godoc.org/github.com/northbright/ming800)

#### License
* [MIT License](LICENSE)
