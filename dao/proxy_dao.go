package dao

import "com.mutantcat.cloud_step/entity"

// 获得所有代理
func getAllProxies() []entity.Proxy {
	proxies := make([]entity.Proxy, 0)
	err := PublicEngine.Find(&proxies)
	if err != nil {
		return nil
	}
	return proxies
}
