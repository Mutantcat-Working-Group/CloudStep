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

// 通过parentid获得urls
func GetUrlsByParentId(parentId int) []entity.Url {
	urls := make([]entity.Url, 0)
	// 用集合id查询集合名
	parent := entity.Collection{}
	_, err := PublicEngine.ID(parentId).Get(&parent)
	if err != nil {
		return nil
	}
	// 用集合名查询urls
	err = PublicEngine.Where("parent = ?", parent.Name).Find(&urls)
	if err != nil {
		return nil
	}
	return urls
}

// 通过parent和url添加url（事务）
func AddUrl(parent string, url string) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 添加url
	newUrl := entity.Url{}
	newUrl.Parent = parent
	newUrl.Url = url
	_, err = session.Insert(&newUrl)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	return true
}

// 通过id删除url （事务）
func DeleteUrlById(id int) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 删除url
	url := entity.Url{}
	_, err = session.ID(id).Delete(&url)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	return true
}
