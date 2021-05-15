# -go-phishing-project

## import go module
```
go mod init <project-name>
go get -d <module-name>
```

## note
在 `Go` 裡面每個 `package` 都要有一個獨立的資料夾


## 實作說明
* golang實作釣魚網站教程 
* 本實作練習透過實作一個模擬github釣魚網站，擷取使用者的資訊
* 並將其存於本地端資料庫中，admin使用者可透過提供預設帳號密碼，在瀏覽器頁面中讀取擷取的帳號資訊
### 資料庫
* 透過go中的ledis套件實作db，存取擷取的下來的帳號密碼

### 讀取command line值
* 透過flag package可以讀取command line值，將他存進變數
* 例如：
```
go run main.go --port=:80 --phishURL=https://phish-github.com
```
可以直接用來設定phishURL與port參數

### GCP ssh 使用
* ssh 建立
```
ssh-keygen -C <user-name>
```
* ssh連線
```
ssh <user-name>@35.229.159.117 -i ~/.ssh/id_rsa
ssh shun@35.229.159.117 -i ~/.ssh/id_rsa
```

### 複製目前資料夾東西到遠端資料夾

```
scp -i ~/.ssh/id_rsa -r . shun@35.229.159.117:~/go-phishing
scp -i ~/.ssh/id_rsa -r ~/Documents/implement/go-phishing-project shun@35.229.159.117:~/go-phishing
```