package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// Service
type Service struct {
	mu      sync.Mutex
	typ     reflect.Type
	rcvr    reflect.Value
	methods map[string]reflect.Method
}

// Server 
type Server struct {
	mu       sync.Mutex
	services map[string]*Service 
	count    int                 
}

// Create a new server at server side
func MakeNewServer() *Server {
	res := new(Server)
	res.services = make(map[string]*Service)
	return res
}

// Load classes after creating a new server
func (server *Server) Install(object interface{}) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	oTyp := reflect.TypeOf(object)
	if oTyp.Kind() == reflect.Ptr {
		oTyp = oTyp.Elem()
	}
	name := oTyp.Name()
	_, ok := server.services[name]
	if ok == true {
		log.Printf("Class%s has been loaded\n", name)
		return errors.New("Already loaded")
	}
	server.services[name] = Register(object)
	return nil
}

// handle rpc requests
func (server *Server) handleRequest(request []byte) ([]byte, bool) {
	decBuf := bytes.NewBuffer(request)
	dec := gob.NewDecoder(decBuf)
	reqmessage := &reqMessage{}
	err := dec.Decode(reqmessage)
	if err != nil {
		log.Println("decode error")
		return nil, false
	}
	dot := strings.IndexAny(reqmessage.SrcMethod, ".")
	if dot < 0 {
		log.Println("Can't handle input strings.")
		log.Println("The form of inouts should be like Raft.Append")
		return nil, false
	}
	oName := reqmessage.SrcMethod[:dot]
	oMethod := reqmessage.SrcMethod[dot+1:]
	service, ok := server.services[oName]
	if ok == false {
		log.Println("Can't find Server object")
		return nil, false
	}
	args := reqmessage.Data
	res, ok := service.Call(oMethod, args)
	return res, ok
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/api" {
		log.Println(r.URL.Path)
		log.Println("Need /v1/api to get api")
	}
	switch r.Method {
	case "GET":
		io.WriteString(w, "GET success\n")
		log.Println("GET success")

	case "POST":
		//io.WriteString(w, "POST success\n")
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Can't get information from POST")
			return
		}
		log.Println("Get information from POST")
		res, ok := server.handleRequest(buf)
		rpyMess := replyMessage{Ok: ok, Data: res}
		replyBuf := bytes.NewBuffer(nil)
		replyEnc := gob.NewEncoder(replyBuf)
		err = replyEnc.Encode(&rpyMess)
		if err != nil {
			log.Println("Error in encoding ok and data")
			log.Println(err)
			return
		}
		_, err = w.Write(replyBuf.Bytes())
		if err != nil {
			log.Println("Can't return the result")
			return
		}
	}

}

type reqMessage struct {
	SrcMethod string // Class+name，like Raft.Append
	Data      []byte 
}

type replyMessage struct {
	Ok   bool
	Data []byte
}

type Client struct {
	Adress string
}

func (end *Client) SendRequest(name string, args interface{}) ([]byte, error) {
	reqMess := reqMessage{}
	reqMess.SrcMethod = name
	argBuf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(argBuf)
	err := enc.Encode(args)
	if err != nil {
		log.Println("Can't make input binary on client side")
		log.Println(err)
		return nil, err
	}
	reqMess.Data = argBuf.Bytes()
	reqBuf := bytes.NewBuffer(nil)
	encReq := gob.NewEncoder(reqBuf)
	err = encReq.Encode(reqMess)
	if err != nil {
		log.Println("Can't make reqMess binary on client side")
		log.Println(err)
		return nil, err
	}
	resp, err := http.Post(end.Adress, "application/x-www-form-urlencoded", reqBuf)
	if err != nil {
		log.Println("Fail to connect client")
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	resBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Can't read data from response")
		log.Println(err)
		return nil, err
	}
	return resBuf, nil
}

func Register(rcvr interface{}) *Service {
	res := &Service{}
	res.methods = make(map[string]reflect.Method)
	res.rcvr = reflect.ValueOf(rcvr)
	res.typ = reflect.TypeOf(rcvr)
	num := res.typ.NumMethod()
	for i := 0; i < num; i++ {
		method := res.typ.Method(i)
		mname := method.Name
		res.methods[mname] = method
	}
	return res
}

func (srv *Service) Call(name string, args []byte) ([]byte, bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	method, ok := srv.methods[name]
	if ok == false {
		log.Println("Can't find corresponding method")
		return nil, false
	}
	inbuf := bytes.NewBuffer(args)
	argsType := method.Type.In(1)
	//replyType := method.Type.Out(0)
	//replyType = replyType.Elem() 
	argPtr := reflect.New(argsType)
	//replyPtr := reflect.New(replyType)
	resbuf := bytes.NewBuffer(nil)
	dec := gob.NewDecoder(inbuf)
	enc := gob.NewEncoder(resbuf)
	err := dec.Decode(argPtr.Interface())
	if err != nil {
		log.Println("Error in decoding args[]byte")
		return nil, false
	}
	function := method.Func
	res := function.Call([]reflect.Value{srv.rcvr, argPtr.Elem()})
	err = enc.Encode(res[0].Interface())
	if err != nil {
		log.Println("xxx")
		log.Println(err)
		return nil, false
	}
	reply := resbuf.Bytes()
	return reply, true

}

func ResultIsOk(data []byte) bool {
	var res replyMessage
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&res)
	if err != nil {
		log.Fatalln(err)
	}
	return res.Ok
}

func ConvertToNormalType(data []byte, res interface{}) {
	resTyp := reflect.TypeOf(res)
	if resTyp.Kind() != reflect.Ptr {
		log.Println("Input should be pointer")
		log.Println("Program exits")
		return
	}
	var rpy replyMessage
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&rpy)
	if err != nil {
		log.Fatalln(err)
	}
	buf = bytes.NewBuffer(rpy.Data)
	resDec := gob.NewDecoder(buf)
	err = resDec.Decode(res)
	if err != nil {
		log.Println("Error in decoding input")
		log.Println("Might have something to do with input pointers")
		log.Println(err)
		return
	}

}
