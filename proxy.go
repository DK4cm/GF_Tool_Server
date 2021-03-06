package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
	//"fmt"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"

	cipher "github.com/xxzl0130/GF_Tool_Server/GF_cipher"
)

func (tool *Tool) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response{
	type Uid struct {
		Sign            string `json:"sign"`
	}

	// 获取远程地址
	var remote string
	if resp.Request != nil {
		s := strings.Split(resp.Request.RemoteAddr,":")
		remote = s[0]
	}else{
		return resp
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("读取响应数据失败 -> %+v", err)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		return resp
	}

	if strings.HasSuffix(ctx.Req.URL.Path,"/Index/getDigitalSkyNbUid") || strings.HasSuffix(ctx.Req.URL.Path, "/Index/getUidTianxiaQueue") || strings.HasSuffix(ctx.Req.URL.Path,"/Index/getUidEnMicaQueue"){
		// 解析sign
		data, err := cipher.AuthCodeDecodeB64Default(string(body)[1:])
		if err != nil {
			//fmt.Printf("解析Uid数据失败 -> %+v", err)
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return resp
		}
		uid := Uid{}
		if err := json.Unmarshal([]byte(data), &uid); err != nil {
			//fmt.Printf("解析JSON数据失败 -> %+v", err)
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return resp
		}
		info := SignInfo{
			sign: uid.Sign,
			time: time.Now().Unix(),
		}
		tool.signMutex.Lock()
		tool.sign[remote] = info
		tool.signMutex.Unlock()
		//fmt.Printf("解析Uid成功")
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		return resp
	}else if strings.HasSuffix(ctx.Req.URL.Path,"/Index/index"){
		// 获取详细数据
		tool.signMutex.RLock()
		signInfo, isPresent := tool.sign[remote]
		tool.signMutex.RUnlock()
		if !isPresent{
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return resp
		}
		sign := signInfo.sign
		data, err := cipher.AuthCodeDecodeB64(string(body)[1:], sign, true)
		if err != nil {
			//fmt.Printf("解析用户数据失败 -> %+v", err)
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return resp
		}
		//_ = ioutil.WriteFile("response.json", []byte(data), 0)
		// 先保存数据
		tool.saveUserInfo(data)
	}

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return resp
}

func (tool *Tool) saveUserInfo(data string){
	info := GF_Simple_Json{}
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return
	}
	userinfo := UserInfo{
		rule : "02",
		name : info.Info.Name,
		uid : info.Info.User_id,
		allData : data,
		time : time.Now().Unix(),
	}
	//fmt.Printf("uid:%v, name:%v\n",info.Info.User_id,info.Info.Name)
	tool.infoMutex.Lock()
	tool.userinfo[userinfo.uid] = userinfo
	tool.infoMutex.Unlock()
}

func (tool *Tool) condition() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		//fmt.Printf("请求 -> %v\n", path(req))
		if strings.HasSuffix(req.Host, "ppgame.com") || strings.HasSuffix(req.Host, "sn-game.txwy.tw")  || strings.HasSuffix(req.Host, "girlfrontline.co.kr") || strings.HasSuffix(req.Host, "sunborngame.com") || strings.HasSuffix(req.Host, "sn-game.txwy.tw") {
			if strings.HasSuffix(req.URL.Path, "/Index/index") || strings.HasSuffix(req.URL.Path, "/Index/getDigitalSkyNbUid") || strings.HasSuffix(req.URL.Path, "/Index/getUidTianxiaQueue") || strings.HasSuffix(req.URL.Path,"/Index/getUidEnMicaQueue"){
				return true
			}
		}
		return false
	}
}

func (tool *Tool) block() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		if strings.HasSuffix(req.Host, "ppgame.com") {
			if strings.HasSuffix(req.URL.Path, "/Index/index") || strings.HasSuffix(req.URL.Path, "/Index/getDigitalSkyNbUid") || strings.HasSuffix(req.URL.Path, "/Index/getUidTianxiaQueue") ||
			  strings.HasSuffix(req.URL.Path, ".txt") || strings.HasSuffix(req.URL.Path, "/Index/version") ||
			  strings.HasSuffix(req.URL.Path, "auth") || strings.HasSuffix(req.Host, "res.ppgame.com") || strings.HasSuffix(req.URL.Path, "index.php") {
				return false
			}
			return true
		}
		return true
	}
}

func (tool *Tool) getLocalhost() (string, error) {
	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		return "", errors.WithMessage(err, "连接 114.114.114.114 失败")
	}
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", errors.WithMessage(err, "解析本地主机地址失败")
	}
	return host, nil
}

func path(req *http.Request) string {
	if req.URL.Path == "/" {
		return req.Host
	}
	return req.Host + req.URL.Path
}