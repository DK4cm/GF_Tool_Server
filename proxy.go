package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"regexp"
	"path/filepath"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"

	"github.com/xxzl0130/rsyars/pkg/util"
	"github.com/xxzl0130/rsyars/rsyars.adapter/hycdes"
	"github.com/xxzl0130/rsyars/rsyars.x/soc"
	cipher "github.com/xxzl0130/GF_Tool_Server/GF_cipher"
)

func (ar *AntiRivercrab) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response{
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

	if strings.HasSuffix(ctx.Req.URL.Path,"/Index/getDigitalSkyNbUid"){
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
		sign := ar.sign[remote].sign
		tool.signMutex.RUnlock()
		data, err := cipher.AuthCodeDecodeB64(string(body)[1:], sign, true)
		if err != nil {
			//fmt.Printf("解析用户数据失败 -> %+v", err)
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return resp
		}

		// 做处理
		tool.buildChip(data)

	}
}

func (tool *Tool) condition() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		if strings.HasSuffix(req.Host, "ppgame.com") {
			if strings.HasSuffix(req.URL.Path, "/Index/index") || strings.HasSuffix(req.URL.Path, "/Index/getDigitalSkyNbUid"){
				return true
			}
		}
		return false
	}
}

func (tool *Tool) block() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		if ar.host.MatchString(req.Host) && ar.url.MatchString(req.URL.Path) {
			return false
		}else{
			return true
		}
	}
}

func (tool *Tool) getLocalhost() (string, error) {
	conn, err := net.Dial("tcp", "www.baidu.com:80")
	if err != nil {
		return "", errors.WithMessage(err, "连接 www.baidu.com:80 失败")
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