package main

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func TestBasicFunction(t *testing.T) {
	args := Args{1, 2}
	end := Client{Adress: "http://localhost:9090/v1/api"}
	resBuf, err := end.SendRequest("xxx.Add", args)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	fmt.Println(ok)
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if ok != true {
		t.Error("Error Result")
	}
	if *res != 3 {
		t.Error("Error Result")
	}
}

func TestTwoFunctions(t *testing.T) {
	args := Args{1, 2}
	end := Client{Adress: "http://localhost:9090/v1/api"}
	resBuf, err := end.SendRequest("xxx.Add", args)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	fmt.Println(ok)
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if ok != true {
		t.Error("Error Result")
	}
	if *res != 3 {
		t.Error("Error Result")
	}
	resBuf, err = end.SendRequest("yyy.Set", 0)
	ok = ResultIsOk(resBuf)
	if ok != true {
		t.Error("Error in first OK result in second function")
	}
	var resString string
	ConvertToNormalType(resBuf, &resString)
	if resString != "" {
		log.Println(resString)
		t.Error("Error in second returned buf")
	}
	resBuf, err = end.SendRequest("yyy.Inc", 1)
	if err != nil {
		log.Fatalln(err)
	}
	ok = ResultIsOk(resBuf)
	if ok != true {
		t.Error("Error in second OK result")
	}
	res2 := new(int)
	ConvertToNormalType(resBuf, res2)
	if *res2 != 1 {
		log.Println(*res2)
		t.Error("Error in second returned buf")
	}

}

func TestMultiThread(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1000)
	end := Client{Adress: "http://localhost:9090/v1/api"}
	end.SendRequest("yyy.Set", 0)
	for i := 0; i < 1000; i++ {
		go parallel(&end, wg)
	}
	wg.Wait()
	resBuf, err := end.SendRequest("yyy.Inc", 0)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	if ok != true {
		log.Fatalln("Can't return correct value")
	}
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if *res != 1000 {
		t.Error("Can't return expected result")
	}
}

func parallel(end *Client, wg *sync.WaitGroup) {
	resBuf, err := end.SendRequest("yyy.Inc", 1)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	if ok != true {
		log.Fatalln("Can't return correct value")
	}
	res := new(int)
	ConvertToNormalType(resBuf, res)
	wg.Done()
}
