package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
)

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
	newUrl.Id = 0
	newUrl.Parent = parent
	newUrl.Path = url
	newUrl.Alive = true
	newUrl.Retry = 0
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
	// 添加成功过后在缓存中添加
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	collection.WorkCllection[parent] = append(collection.WorkCllection[parent], newUrl)
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
	// 删除成功后删除缓存中的url
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	for k, v := range collection.WorkCllection {
		for i, u := range v {
			if u.Id == id {
				collection.WorkCllection[k] = append(v[:i], v[i+1:]...)
				break
			}
		}
	}
	return true
}

// 通过id修改url
func UpdateUrlById(id int, url string) bool {
	// 开启事务
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	// 修改url
	newUrl := entity.Url{}
	newUrl.Path = url
	_, err = session.ID(id).Update(&newUrl)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 修改成功后修改缓存中的url
	collection.MWorkCllection.Lock()
	defer collection.MWorkCllection.Unlock()
	for k, v := range collection.WorkCllection {
		for i, u := range v {
			if u.Id == id {
				collection.WorkCllection[k][i].Path = url
				break
			}
		}
	}
	return true
}
