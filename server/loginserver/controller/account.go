package controller

import (
	"os"
	"strings"
	"time"

	"github.com/goinggo/mapstructure"
	"github.com/llr104/slgserver/constant"
	"github.com/llr104/slgserver/db"
	"github.com/llr104/slgserver/log"
	"github.com/llr104/slgserver/middleware"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/loginserver/model"
	"github.com/llr104/slgserver/server/loginserver/proto"
	"github.com/llr104/slgserver/util"
	"go.uber.org/zap"
)

var DefaultAccount = Account{}

type Account struct{}

func (this *Account) InitRouter(router *net.Router) {
	group := router.Group("account").Use(middleware.ElapsedTime(), middleware.Log())
	group.AddRouter("login", this.login)
	group.AddRouter("reLogin", this.reLogin)
	group.AddRouter("logout", this.logout, middleware.CheckLogin())
	group.AddRouter("serverList", this.serverList, middleware.CheckLogin())
}

func (this *Account) login(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.LoginReq{}
	rspObj := &proto.LoginRsp{}
	if err := mapstructure.Decode(req.Body.Msg, reqObj); err != nil {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	rsp.Body.Msg = rspObj

	reqObj.Username = strings.TrimSpace(reqObj.Username)
	if reqObj.Username == "" || reqObj.Password == "" {
		rsp.Body.Code = constant.InvalidParam
		return
	}

	user := &model.User{}
	found, err := db.MasterDB.Table(user).Where("username=?", reqObj.Username).Get(user)
	if err != nil {
		log.DefaultLog.Error("Lỗi truy vấn tài khoản đăng nhập",
			zap.String("username", reqObj.Username),
			zap.Error(err))
		rsp.Body.Code = constant.DBError
		return
	}
	if !found {
		log.DefaultLog.Info("Không tìm thấy tài khoản", zap.String("username", reqObj.Username))
		rsp.Body.Code = constant.UserNotExist
		return
	}

	valid, needsUpgrade := util.VerifyPassword(reqObj.Password, user.Passwd, user.Passcode)
	if !valid {
		log.DefaultLog.Info("Mật khẩu không chính xác", zap.String("username", user.Username))
		rsp.Body.Code = constant.PwdIncorrect
		return
	}

	if needsUpgrade {
		this.upgradeLegacyPassword(user, reqObj.Password)
	}

	now := time.Now()
	session := util.NewSession(user.UId, now)
	sessionString := session.String()
	log.DefaultLog.Info("Đăng nhập thành công", zap.String("username", user.Username))

	history := &model.LoginHistory{
		UId:      user.UId,
		CTime:    now,
		Ip:       reqObj.Ip,
		Hardware: reqObj.Hardware,
		State:    model.Login,
	}
	if _, err = db.MasterDB.Insert(history); err != nil {
		log.DefaultLog.Error("Không thể ghi lịch sử đăng nhập", zap.Error(err))
	}

	lastLogin := &model.LoginLast{}
	found, err = db.MasterDB.Table(lastLogin).Where("uid=?", user.UId).Get(lastLogin)
	if err != nil {
		log.DefaultLog.Error("Không thể đọc lần đăng nhập gần nhất", zap.Error(err))
		rsp.Body.Code = constant.DBError
		return
	}

	if found {
		lastLogin.IsLogout = 0
		lastLogin.Ip = reqObj.Ip
		lastLogin.LoginTime = now
		lastLogin.Session = sessionString
		lastLogin.Hardware = reqObj.Hardware
		if _, err = db.MasterDB.ID(lastLogin.Id).
			Cols("is_logout", "ip", "login_time", "session", "hardware").
			Update(lastLogin); err != nil {
			log.DefaultLog.Error("Không thể cập nhật lần đăng nhập gần nhất", zap.Error(err))
			rsp.Body.Code = constant.DBError
			return
		}
	} else {
		lastLogin = &model.LoginLast{
			UId:       user.UId,
			LoginTime: now,
			Ip:        reqObj.Ip,
			Session:   sessionString,
			Hardware:  reqObj.Hardware,
			IsLogout:  0,
		}
		if _, err = db.MasterDB.Insert(lastLogin); err != nil {
			log.DefaultLog.Error("Không thể tạo bản ghi đăng nhập gần nhất", zap.Error(err))
			rsp.Body.Code = constant.DBError
			return
		}
	}

	rspObj.Session = sessionString
	rspObj.Password = ""
	rspObj.Username = user.Username
	rspObj.UId = user.UId
	rsp.Body.Code = constant.OK
	net.ConnMgr.UserLogin(req.Conn, sessionString, lastLogin.UId)
}

func (this *Account) upgradeLegacyPassword(user *model.User, password string) {
	passwordHash, err := util.HashPassword(password)
	if err != nil {
		log.DefaultLog.Error("Không thể tạo bcrypt khi nâng cấp tài khoản",
			zap.Int("uid", user.UId), zap.Error(err))
		return
	}

	data := map[string]interface{}{
		"passwd":   passwordHash,
		"passcode": "",
		"mtime":    time.Now(),
	}
	if _, err = db.MasterDB.Table(user).Where("uid=?", user.UId).Update(data); err != nil {
		log.DefaultLog.Error("Không thể nâng cấp mật khẩu tài khoản",
			zap.Int("uid", user.UId), zap.Error(err))
	}
}

func (this *Account) reLogin(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.ReLoginReq{}
	rspObj := &proto.ReLoginRsp{}
	if err := mapstructure.Decode(req.Body.Msg, reqObj); err != nil || reqObj.Session == "" {
		rsp.Body.Code = constant.SessionInvalid
		return
	}

	rsp.Body.Msg = rspObj
	rspObj.Session = reqObj.Session

	session, err := util.ParseSession(reqObj.Session)
	if err != nil || !session.IsValid() {
		rsp.Body.Code = constant.SessionInvalid
		return
	}

	lastLogin := &model.LoginLast{}
	found, err := db.MasterDB.Table(lastLogin).Where("uid=?", session.Id).Get(lastLogin)
	if err != nil {
		rsp.Body.Code = constant.DBError
		return
	}
	if !found || lastLogin.Session != reqObj.Session {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	if lastLogin.Hardware != reqObj.Hardware {
		rsp.Body.Code = constant.HardwareIncorrect
		return
	}

	rsp.Body.Code = constant.OK
	net.ConnMgr.UserLogin(req.Conn, reqObj.Session, lastLogin.UId)
}

func (this *Account) logout(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.LogoutReq{}
	rspObj := &proto.LogoutRsp{}
	if err := mapstructure.Decode(req.Body.Msg, reqObj); err != nil {
		rsp.Body.Code = constant.InvalidParam
		return
	}

	rsp.Body.Msg = rspObj
	rspObj.UId = reqObj.UId
	rsp.Body.Code = constant.OK
	log.DefaultLog.Info("Đăng xuất", zap.Int("uid", reqObj.UId))

	now := time.Now()
	history := &model.LoginHistory{UId: reqObj.UId, CTime: now, State: model.Logout}
	if _, err := db.MasterDB.Insert(history); err != nil {
		log.DefaultLog.Error("Không thể ghi lịch sử đăng xuất", zap.Error(err))
	}

	lastLogin := &model.LoginLast{}
	found, err := db.MasterDB.Table(lastLogin).Where("uid=?", reqObj.UId).Get(lastLogin)
	if err == nil && found {
		lastLogin.IsLogout = 1
		lastLogin.LogoutTime = now
		_, _ = db.MasterDB.ID(lastLogin.Id).Cols("is_logout", "logout_time").Update(lastLogin)
	}
	net.ConnMgr.UserLogout(req.Conn)
}

func (this *Account) serverList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.ServerListReq{}
	_ = mapstructure.Decode(req.Body.Msg, reqObj)

	gatePublicURL := strings.TrimSpace(os.Getenv("GATE_PUBLIC_URL"))
	server := proto.Server{Id: 1, Slg: gatePublicURL, Chat: gatePublicURL}
	rspObj := &proto.ServerListRsp{Lists: []proto.Server{server}}

	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}
