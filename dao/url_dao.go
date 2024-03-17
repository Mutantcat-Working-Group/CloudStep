package dao

import "com.mutantcat.cloud_step/entity"

// 获得Parent为某个集合的url
func getUrlsByParent(parent string) []entity.Url {
	urls := make([]entity.Url, 0)
	err := PublicEngine.Where("parent = ?", parent).Find(&urls)
	if err != nil {
		return nil
	}
	return urls
}
