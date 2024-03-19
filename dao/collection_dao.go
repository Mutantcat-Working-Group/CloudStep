package dao

import (
	C "com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
)

func GetAllCollections() []entity.Collection {
	collections := make([]entity.Collection, 0)
	err := PublicEngine.Find(&collections)
	if err != nil {
		return nil
	}
	return collections
}

// 检查集合名是否存在
func CheckCollectionNameExist(name string) bool {
	count, err := PublicEngine.Where("name = ?", name).Count(&entity.Collection{})
	if err != nil {
		return false
	}
	return count > 0
}

func AddCollection(collectionName string, urls []entity.Url) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	collection := entity.Collection{}
	collection.Name = collectionName
	_, err = session.Insert(&collection)
	if err != nil {
		session.Rollback()
		return false
	}
	for _, url := range urls {
		url.Parent = collectionName
		url.Alive = true
		url.Retry = 0
		_, err = session.Insert(&url)
		if err != nil {
			session.Rollback()
			return false
		}
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 数据库中添加无误之后 添加进缓存中
	C.MWorkCllection.Lock()
	defer C.MWorkCllection.Unlock()
	C.WorkCllection[collection.Name] = urls
	return true
}
