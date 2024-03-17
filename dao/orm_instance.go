package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
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
	err = engine.Sync2(new(entity.User))
	if err != nil {
		fmt.Println("同步表结构user失败, err: ", err)
	}
	err = engine.Sync2(new(entity.Collection))
	if err != nil {
		fmt.Println("同步表结构collection失败, err: ", err)
	}
	err = engine.Sync2(new(entity.Proxy))
	if err != nil {
		fmt.Println("同步表结构proxy失败, err: ", err)
	}
	err = engine.Sync2(new(entity.SelfHelp))
	if err != nil {
		fmt.Println("同步表结构self_help失败, err: ", err)
	}
	err = engine.Sync2(new(entity.Url))
	if err != nil {
		fmt.Println("同步表结构url失败, err: ", err)
	}

	PublicEngine = engine
	initUser()
	initModes()
}

func initUser() {
	// 先获得用户数量
	count, err := PublicEngine.Count(&entity.User{})
	if err != nil {
		log.Fatal("初始化用户失败")
	}
	if count == 0 {
		user := entity.User{}
		user.Username = "admin96"
		user.Password = "admin96"
		insert, err := PublicEngine.Insert(&user)
		if err != nil || insert == 0 {
			log.Fatal("初始化用户失败")
		}
	}
}

func initModes() {
	// 获得集合与对应url集
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	collections := GetAllCollections()
	// 遍历集合表
	for _, one := range collections {
		// 获得集合的所有url并存入 WorkCollection
		collection.WorkCllection[one.Name] = getUrlsByParent(one.Name)
	}

	// 获得自助模式列表
	collection.MSelfHelpMode.Lock()
	defer collection.MSelfHelpMode.Unlock()
	selfHelps := getAllSelfHelps()
	// 遍历自助模式表
	for _, one := range selfHelps {
		// 存入SelfHelpMode
		collection.SelfHelpMode[one.Way] = one
	}

	// 获得代理列表
	collection.MProxyMode.Lock()
	defer collection.MProxyMode.Unlock()
	proxys := getAllProxies()
	// 遍历代理表
	for _, one := range proxys {
		// 存入ProxyMode
		collection.ProxyMode[one.Way] = one
	}
}
