package dao

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"xorm.io/xorm"
)

var PublicEngine *xorm.Engine

func init() {
	engine, err := xorm.NewEngine("sqlite3", "./cloud_step.db")
	if err != nil {
		log.Print(err)
	}

	//同步数据库中的用户表
	err = engine.Sync2(new(User))
	if err != nil {
		fmt.Println("同步表结构失败, err: ", err)
	}

	PublicEngine = engine
	initUser()
}

func initUser() {
	// 先获得用户数量
	count, err := PublicEngine.Count(&User{})
	if err != nil {
		log.Fatal("初始化用户失败")
	}
	if count == 0 {
		user := User{}
		user.Username = "admin96"
		user.Password = "admin96"
		insert, err := PublicEngine.Insert(&user)
		if err != nil || insert == 0 {
			log.Fatal("初始化用户失败")
		}
	}
}
