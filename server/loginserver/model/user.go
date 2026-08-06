package model

import "time"

type User struct {
	UId      int       `xorm:"uid pk autoincr"`
	Username string    `xorm:"username" validate:"min=3,max=20"`
	Passcode string    `xorm:"passcode"`
	Passwd   string    `xorm:"passwd"`
	Hardware string    `xorm:"hardware"`
	Status   int       `xorm:"status"`
	Ctime    time.Time `xorm:"ctime"`
	Mtime    time.Time `xorm:"mtime"`
	IsOnline bool      `xorm:"-"`
}

func (this *User) TableName() string {
	return "tb_user_info"
}
