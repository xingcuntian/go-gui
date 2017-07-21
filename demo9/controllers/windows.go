package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"errors"

	"github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
)

const URL = "https://apipre.bi.sensetime.com"
const PathSeparator = `/`
const STATUS_CODE = 200

var CurCookies []*http.Cookie
var CurCookieJar *cookiejar.Jar
var tdb map[string]string
var AdminId int32

type Account struct {
	Account  string
	Password string
}

type Msg struct {
	StatusCode int32  `json:"status_code"`
	Message    string `json:"message"`
}

type UploadData struct {
	ImageKey string `json:"image_key"`
}

type MsgUpload struct {
	Msg
	Data []UploadData `json:"data"`
}

type LoginData struct {
	Id    int32  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
type MsgLogin struct {
	Msg
	Data []LoginData `json:"data"`
}

func init() {
	CurCookies = nil
	CurCookieJar, _ = cookiejar.New(nil)
	tdb = make(map[string]string)
	AdminId = 0
}

func SetEventHandler(w *window.Window) {
	//注册dump函数方便在tis脚本中打印数据
	w.DefineFunction("dump", func(args ...*sciter.Value) *sciter.Value {
		for _, v := range args {
			fmt.Print(v.String() + " ")
		}
		fmt.Println()
		return sciter.NullValue()
	})

	//login函数，用于用户登录逻辑，这里只是简单的把数据打印出来
	w.DefineFunction("login", func(args ...*sciter.Value) *sciter.Value {
		reqJson := args[0].String()
		var account Account
		err := json.Unmarshal([]byte(reqJson), &account)
		if err != nil {
			return sciter.NewValue(err.Error())
		}

		reqUrl := URL + "/sm-login"
		client := http.Client{
			CheckRedirect: nil,
			Jar:           CurCookieJar,
		}
		req, err := http.NewRequest("POST", reqUrl, strings.NewReader("email="+account.Account+"&password="+account.Password))
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		var msglogin MsgLogin
		err = json.Unmarshal([]byte(body), &msglogin)
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		if msglogin.Msg.StatusCode != STATUS_CODE {
			err = errors.New(msglogin.Msg.Message)
			return sciter.NewValue(err.Error())
		} else {
			CurCookies = CurCookieJar.Cookies(req.URL)
			w.LoadFile("views/index.html")
		}
		return sciter.NullValue()
	})

	w.DefineFunction("conference", func(args ...*sciter.Value) *sciter.Value {
		httpClient := &http.Client{
			Jar: CurCookieJar,
		}
		var httpReq *http.Request
		httpReq, _ = http.NewRequest("GET", URL+"/sm-conference?per_page=100", nil)
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			fmt.Println(err)
		}
		defer httpResp.Body.Close()
		body, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println(string(body))
		return sciter.NewValue(string(body))
	})

	//上传信息
	w.DefineFunction("upload", func(args ...*sciter.Value) *sciter.Value {
		conferenceID := args[2].String()
		fmt.Println(conferenceID)
		excelFile := args[0].String()
		excelFile = strings.TrimLeft(excelFile, "file://")
		fields := []string{"avatar", "guest_name", "mobile"}
		cellsMap, err := ReadCVS(fields, excelFile)
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		//fmt.Println(cellsMap)
		folderPath := args[1].String()
		dir, err := ioutil.ReadDir(folderPath)
		if err != nil {
			return sciter.NewValue(err.Error())
		}
		var mapString string
		//fmt.Println(len(dir))
		for _, fi := range dir {
			if fi.IsDir() {
				continue
			} else {
				keyImg, err := UploadFile(URL+"/upload", folderPath+PathSeparator+fi.Name())
				if err != nil {
					return sciter.NewValue(err.Error())
				}
				mapString = cellsMap[fi.Name()] + fields[0] + "=" + keyImg + "&conference_id=" + conferenceID + "&grade_id=1"
				fmt.Println(mapString)
				retstr, _ := SyncData(mapString)
				fmt.Println(retstr)
			}

		}
		//fmt.Println(cellsMap)
		return sciter.NewValue("ok")
	})

}