package dao

import "com.mutantcat.cloud_step/entity"

// 获得Parent为某个集合的url
func GetUrlsByParent(parent string) []entity.Url {
	urls := make([]entity.Url, 0)
	err := PublicEngine.Where("parent = ?", parent).Find(&urls)
	if err != nil {
		return nil
	}
	return urls
}

func GetUrlById(id int) *entity.Url {
	url := entity.Url{}
	_, err := PublicEngine.ID(id).Get(&url)
	if err != nil {
		return nil
	}
	return &url
}

func AddUrl(url entity.Url) bool {
	_, err := PublicEngine.Insert(&url)
	if err != nil {
		return false
	}
	return true
}
