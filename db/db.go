// ----- db.go -----
package db

import (
	"log"
	// 把 LedisDB import 進來
	"github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
)

var db *ledis.DB

//將釣到的資料加入key為fishes的list
func Insert(s string) {
	// fishes 是這個 list 的名字（key）
	fishes := []byte("fishes")

	// 把字串 s 加到 fishes 裡面
	db.RPush(fishes, []byte(s))
}

//讀取db中資料
func SelectAll() []string {
	fishes := []byte("fishes")

	// 取得 list 的長度 -> nFish
	nFish, _ := db.LLen(fishes)

	// 從 list 裡面取得所有資料
	datas, _ := db.LRange(fishes, 0, int32(nFish))

	// 因為取出來的每一筆資料型別都是 []byte
	// 把每筆資料都轉成 string 放到 strs 裡面
	strs := []string{}
	for _, data := range datas {
		strs = append(strs, string(data))
	}

	return strs
}

func Connect() {
	// 建立一個設定檔，把資料的儲存位置設定到 ./db-data
	cfg := config.NewConfigDefault()
	cfg.DataDir = "./db_data"

	// 要求建立連線
	l, _ := ledis.Open(cfg)
	_db, err := l.Select(0)

	if err != nil {
		panic(err)
	}

	// 成功建立連線，把 db 存到全域變數，之後其他地方會用到
	db = _db
	log.Println("Connect to db successfully")
}
