// weblog project main.go
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
)

var conn_pool = make(map[string]net.Conn)
var log_channel = make(chan string)
var l sync.Mutex

func writeHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	log.Println(r.FormValue("log"))
	fmt.Fprintf(w, r.FormValue("log"))
	log_str := r.FormValue("log")
	log_channel <- log_str
}

func EchoFunc(conn net.Conn) {
	defer conn.Close()
	defer removePool(conn)
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			addr := conn.RemoteAddr().String()
			log.Println(addr + " disconnect")
			return
		}
	}
}

func addPool(conn net.Conn) {

	l.Lock()
	addr := conn.RemoteAddr().String()
	conn_pool[addr] = conn
	log.Println(conn_pool)
	log.Println(addr + " add to pool ")
	l.Unlock()

}

func removePool(conn net.Conn) {

	l.Lock()
	addr := conn.RemoteAddr().String()
	delete(conn_pool, addr)
	log.Println(conn_pool)
	log.Println(addr + " remove from pool")
	l.Unlock()

}

func main() {

	listener, err := net.Listen("tcp", "0.0.0.0:8088")
	if err != nil {
		fmt.Println("error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	log.Printf("running ...\n")

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal("Error accept:", err.Error())
				return
			}
			addr := conn.RemoteAddr().String()
			log.Println(addr + " connected")
			addPool(conn)
			go func() {
				EchoFunc(conn)
			}()
		}
	}()

	go func() {
		for {
			log.Println("Begin read log channel")
			log_str := <-log_channel
			log_byte := []byte(log_str + "\n")
			log.Println(conn_pool)

			for _, conn := range conn_pool {
				if conn != nil {
					log.Println("begin echo log")
					conn.Write(log_byte)
				}
			}
		}
	}()

	http.HandleFunc("/write", writeHandler)
	//http.ListenAndServe(":8089", nil)
	s := &http.Server{Addr: ":8089"}
	s.ListenAndServe()

}
