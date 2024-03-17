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
