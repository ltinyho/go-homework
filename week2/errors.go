package week2

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var db = &sql.DB{} // 假设初始化 db 实例
var log = logrus.New()

type user struct {
	Uid  int
	Name string
}

// 返回 user结构体和指针都可以,如果外层需要判断是否为空,根据指针或者uid是否<=0判断
func service() {
	u, err := getDataFormDb(1)
	if err != nil {
		log.Errorf("%+v", err)
		return
	}
	if u.Uid > 0 { // 用户存在
		fmt.Println(u.Uid, u.Name)
	} else { // 用户不存在
		// doSomething
	}
}

// 不返回 sql.ErrNoRows
// nil 不为空时,表示 db 执行出错
func getDataFormDb(uid int) (u *user, err error) {
	u = new(user)
	err = db.QueryRow("select uid,name from user where uid =$1", uid).Scan(&u.Uid, &u.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			// 有个疑问这里是打error等级的错误呢,还是打Info或者Warn等级的错误合适
			log.Infof("get user empty uid:%d", uid)
			return u, nil // 处理完成后降级
		} else {
			return nil, errors.Wrapf(err, "query user uid:%d err", uid)
		}
	}
	return u, nil
}
