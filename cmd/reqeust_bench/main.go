package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func main() {

	group := sync.WaitGroup{}
	rand.Seed(time.Now().Unix())
	for i := 0; i < 1000000; i++ {
		time.Sleep(time.Second / 10)
		group.Add(1)
		go func() {
			defer group.Done()
			params := struct {
				Op     string   `json:"op"`
				Params []string `json:"params"`
			}{}
			params.Op = "cp"
			s := randomString(5)
			params.Params = []string{s, "QmayKQWJgWmr46DqWseADyUanmqxh662hPNdAkdHRjiQQH"}
			bs, _ := json.Marshal(params)

			req, err := http.NewRequest("POST", "http://127.0.0.1:10087/fs", bytes.NewReader(bs))
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Close = true
			now := time.Now()
			res, err := http.DefaultClient.Do(req)
			if err == nil {
				bs, _ = ioutil.ReadAll(res.Body)
				after := time.Now()
				fmt.Println(string(bs), after.Sub(now).String())
			} else {
				fmt.Println(err.Error())
			}
		}()
	}
	group.Wait()
}

const (
	chars = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// RandomStrings returns a slice of randomly generated strings.
func RandomStrings(maxlen uint, n int) []string {
	ss := make([]string, 0)
	for i := 0; i < n; i++ {
		ss = append(ss, randomString(maxlen))
	}
	return ss
}

func randomString(l uint) string {
	s := make([]byte, l)
	for i := 0; i < int(l); i++ {
		s[i] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}
