package dao

import "com.mutantcat.cloud_step/entity"

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
	return true
}
