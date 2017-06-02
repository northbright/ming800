# ming800

[![Build Status](https://travis-ci.org/northbright/ming800.svg?branch=master)](https://travis-ci.org/northbright/ming800)
[![Go Report Card](https://goreportcard.com/badge/github.com/northbright/ming800)](https://goreportcard.com/report/github.com/northbright/ming800)
[![GoDoc](https://godoc.org/github.com/northbright/ming800?status.svg)](https://godoc.org/github.com/northbright/ming800)

ming800是一个[Golang](https://golang.org)包，提供适用于旧单机版本明日管理系统的API接口。

#### 适用版本
* 旧版单机安装版本（2012年）
* 只有1个校区（总部）

#### 已知问题
* 按姓名，电话号码搜索学生时，最大搜索结果数量为15（原因：明日系统网页最多显示15个结果）。

#### 工作方式
* 抓取网页结果，并且使用正则表达式得到数据。

#### 功能
* 按电话号码，姓名搜索学生信息
* 获取当前所有专业与开设班级
* 获取一个班级所有学生信息
* 获得当前在读所有学生信息

#### 文档
* [API References](https://godoc.org/github.com/northbright/ming800)

#### License
* [MIT License](LICENSE)
