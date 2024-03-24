package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
)

// 获得所有代理
func GetAllProxies() []entity.Proxy {
	proxies := make([]entity.Proxy, 0)
	err := PublicEngine.Find(&proxies)
	if err != nil {
		return nil
	}
	return proxies
}

func GetProxyWayById(id int) string {
	var proxy entity.Proxy
	has, err := PublicEngine.ID(id).Get(&proxy)
	if err != nil {
		return ""
	}
	if !has {
		return ""
	}
	return proxy.Way
}

// 获得所有代理
func AddProxy(proxy entity.Proxy) bool {
	proxy.Id = 0
	proxy.Index = 0
	proxy.AliveNum = 0
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.Insert(&proxy)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MProxyMode.Lock()
	defer collection.MProxyMode.Unlock()
	collection.ProxyMode[proxy.Way] = proxy
	return true
}

func DeleteProxyById(id int) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.ID(id).Delete(&entity.Proxy{})
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MProxyMode.Lock()
	defer collection.MProxyMode.Unlock()
	for k, v := range collection.ProxyMode {
		if v.Id == id {
			delete(collection.ProxyMode, k)
			break
		}
	}
	return true
}

func UpdateProxyById(proxy entity.Proxy) bool {
	oldWay := GetProxyWayById(proxy.Id)
	session := PublicEngine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return false
	}
	_, err = session.ID(proxy.Id).Update(&proxy)
	if err != nil {
		session.Rollback()
		return false
	}
	err = session.Commit()
	if err != nil {
		session.Rollback()
		return false
	}
	// 更新缓存
	collection.MProxyMode.Lock()
	defer collection.MProxyMode.Unlock()
	delete(collection.ProxyMode, oldWay)
	collection.ProxyMode[proxy.Way] = proxy
	return true
}

func CheckProxyNameExist(name string) bool {
	var proxy entity.Proxy
	has, err := PublicEngine.Where("name = ?", name).Get(&proxy)
	if err != nil {
		return true
	}
	return has
}

func CheckProxyWayExist(way string) bool {
	var proxy entity.Proxy
	has, err := PublicEngine.Where("way = ?", way).Get(&proxy)
	if err != nil {
		return true
	}
	return has
}
