package dao

import "com.mutantcat.cloud_step/entity"

// 用户名固定为admin96 密码默认为admin96 存储不进行加密
func CheckUser(username string, password string) bool {
	user := entity.User{}
	have, err := PublicEngine.Where("username = ? and password = ?", username, password).Get(&user)
	if err != nil || have == false {
		return false
	}
	return true
}

func ChangePassword(password string) bool {
	session := PublicEngine.NewSession()
	defer session.Close()
	user := entity.User{}
	_, err := PublicEngine.Where("username = ?", "admin96").Get(&user)
	if err != nil {
		return false
	}
	user.Password = password
	_, err = PublicEngine.Where("username = ?", "admin96").Update(&user)
	if err != nil {
		return false
	}

	err = session.Commit()
	if err != nil {
		return false
	}
	return true
}
