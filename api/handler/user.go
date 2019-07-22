package handler

import (
	"encoding/json"
	"github.com/132yse/acgzone-server/api/db"
	"github.com/132yse/acgzone-server/api/def"
	"github.com/132yse/acgzone-server/api/util"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"strconv"
	"log"
	"time"
	auth "github.com/nilslice/jwt"
	"io"
)

const DOMAIN = "clicli.us"

func Register(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, _ := ioutil.ReadAll(r.Body)
	ubody := &def.User{}

	if err := json.Unmarshal(req, ubody); err != nil {
		sendMsg(w,401,"参数解析失败")
		return
	}

	if res, _ := db.GetUser(ubody.Name, 0); res != nil {
		sendMsg(w,401,"用户名已存在")
		return
	}

	if err := db.CreateUser(ubody.Name, ubody.Pwd, ubody.Level, ubody.QQ, ubody.Desc); err != nil {
		sendMsg(w,401,"数据库错误")
		return
	} else {
		sendMsg(w,200,"注册成功啦")
	}

}

func Login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, _ := ioutil.ReadAll(r.Body)
	ubody := &def.User{}

	if err := json.Unmarshal(req, ubody); err != nil {
		sendMsg(w,401,"参数解析失败")
		return
	}

	resp, err := db.GetUser(ubody.Name, 0)
	pwd := util.Cipher(ubody.Pwd)

	if err != nil || len(resp.Pwd) == 0 || pwd != resp.Pwd {
		sendMsg(w,401,"用户名或密码错误")
		return
	} else {
		level := resp.Level
		claims := map[string]interface{}{"exp": time.Now().Add(time.Hour * 24).Unix(), "level": level}
		token, err := auth.New(claims)
		if err != nil {
			return
		}

		resStr, _ := json.Marshal(struct {
			Token string `json:"token"`
		}{Token: token})

		t := http.Cookie{Name: "token", Value: token, Path: "/", MaxAge: 86400, Domain: DOMAIN}
		http.SetCookie(w, &t)
		qq := http.Cookie{Name: "uqq", Value: resp.QQ, Path: "/", MaxAge: 86400, Domain: DOMAIN}
		http.SetCookie(w, &qq)
		uid := http.Cookie{Name: "uid", Value: strconv.Itoa(resp.Id), Path: "/", MaxAge: 86400, Domain: DOMAIN}
		http.SetCookie(w, &uid)
		l := http.Cookie{Name: "level", Value: strconv.Itoa(resp.Level), Path: "/", MaxAge: 86400, Domain: DOMAIN}
		http.SetCookie(w, &l)

		io.WriteString(w, string(resStr))
	}

}

func Logout(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	i := http.Cookie{Name: "uid", Path: "/", Domain: DOMAIN, MaxAge: -1}
	q := http.Cookie{Name: "uqq", Path: "/", Domain: DOMAIN, MaxAge: -1}
	l := http.Cookie{Name: "level", Path: "/", Domain: DOMAIN, MaxAge: -1}
	t := http.Cookie{Name: "token", Path: "/", Domain: DOMAIN, MaxAge: -1}
	http.SetCookie(w, &i)
	http.SetCookie(w, &q)
	http.SetCookie(w, &t)
	http.SetCookie(w, &l)
	sendMsg(w,200,"退出成功啦")
}

func UpdateUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	pint, _ := strconv.Atoi(p.ByName("id"))
	req, _ := ioutil.ReadAll(r.Body)
	ubody := &def.User{}
	if err := json.Unmarshal(req, ubody); err != nil {
		sendMsg(w,200,"参数解析失败")
		return
	}

	old, _ := db.GetUser("", pint)

	if !AuthToken(w, r, old.Level) {
		return
	}
	if old.Name != ubody.Name {
		if res, _ := db.GetUser(ubody.Name, 0); res != nil {
			sendMsg(w,401,"用户名已存在~")
			return
		}
	}

	if resp, err := db.UpdateUser(pint, ubody.Name, ubody.Pwd, ubody.Level, ubody.QQ, ubody.Desc); err != nil {
		sendMsg(w,401,"数据库错误")
		return
	} else {
		ret := &def.User{Id: resp.Id, Name: resp.Name, Level: resp.Level, QQ: resp.QQ, Desc: resp.Desc}
		sendUserResponse(w, ret, 200, "更新成功啦")
	}
}

func DeleteUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if !AuthToken(w, r, 4) {
		return
	}
	uid, _ := strconv.Atoi(p.ByName("id"))
	err := db.DeleteUser(uid)
	if err != nil {
		sendMsg(w,401,"数据库错误")
		return
	} else {
		sendMsg(w,200,"删除成功")
	}
}

func GetUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := r.URL.Query().Get("uname")
	uid, _ := strconv.Atoi(r.URL.Query().Get("uid"))
	resp, err := db.GetUser(uname, uid)
	if err != nil {
		log.Printf("%s", err)
		sendMsg(w,401,"数据库错误")
		return
	}
	res := &def.User{Id: resp.Id, Name: resp.Name, Level: resp.Level, QQ: resp.QQ, Desc: resp.Desc}
	sendUserResponse(w, res, 200, "")

}

func GetUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	level, _ := strconv.Atoi(r.URL.Query().Get("level"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	resp, err := db.GetUsers(level, page, pageSize)
	if err != nil {
		sendMsg(w,401,"数据库错误")
		return
	} else {
		res := &def.Users{Users: resp}
		sendUsersResponse(w, res, 200)
	}
}

func SearchUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	key := r.URL.Query().Get("key")

	resp, err := db.SearchUsers(key)
	if err != nil {
		sendMsg(w,401,"数据库错误")
		return
	} else {
		res := &def.Users{Users: resp}
		sendUsersResponse(w, res, 200)
	}
}
