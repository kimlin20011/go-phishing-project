// main.go
package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func main() {
	http.HandleFunc("/", handler)
	// 錯誤處理，如果有回傳錯誤就直接終止程式並return error
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
