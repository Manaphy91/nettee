package sink

import (
	"net"
	"bytes"
	"strconv"
	"time"
	"os"
	"fmt"
)

type Sink interface{
	Open() (err error)
	Close() (err error)
	InFlowChan() (chan LogEntry)
	IsOpen() (res bool)
}

type LogEntry struct{
	Buff []byte
	Date time.Time
}

type RawSocketSink struct{
	addr net.Addr
	buff []byte
	inflow chan LogEntry
	showDate bool
	openChan chan struct{}
}

func NewRawSocketSink(hostname string, port uint16, holdConn bool) (sink *RawSocketSink){
	hostname = hostname + ":" + strconv.Itoa(int(port))
	sink = &RawSocketSink{}
	(*sink).buff = []byte{}
	var err error
	if holdConn {
		(*sink).addr, err = net.ResolveTCPAddr("tcp", hostname)
		if err != nil {
			fmt.Printf("Error resolving %s network address: %s", err)
			os.Exit(1)
		}
	} else {
		(*sink).addr, err = net.ResolveUDPAddr("udp", hostname)
		if err != nil {
			fmt.Printf("Error resolving %s network address: %s", err)
			os.Exit(1)
		}
	}
	(*sink).inflow = make(chan LogEntry)
	return sink
}

func (sink *RawSocketSink) InFlowChan() (chan LogEntry){
	return (*sink).inflow
}

func (sink *RawSocketSink) Open() (err error){
	if sink.IsOpen() {
		return fmt.Errorf("Error: you can't reopen an opened sink!")
	} else {
		(*sink).openChan = make(chan struct{})
		var dial func () (conn net.Conn, err error)
		switch (*sink).addr.(type) {
			case *net.UDPAddr:
				dial = func () (conn net.Conn, err error) {
					conn, err = net.DialUDP("udp", nil, (*sink).addr.(*net.UDPAddr))
					if err != nil {
						err = fmt.Errorf("Error during UDP socket opening: %s", err)
					}
					return conn, err
				}
			case *net.TCPAddr:
				dial = func () (conn net.Conn, err error) {
					conn, err = net.DialTCP("tcp", nil, (*sink).addr.(*net.TCPAddr))
					if err != nil {
						err = fmt.Errorf("Error during TDP socket opening: %s", err)
					}
					return conn, err
				}
		}
		go func(){
			defer sink.Close()
			for {
				if !sink.IsOpen() {
					return
				} else {
					for entry := range (*sink).inflow {
						conn, err := dial()
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						var buff *bytes.Buffer
						if (*sink).showDate {
							dateString := fmt.Sprintf("[%.2d/%.2d/%d - %.2d:%.2d:%.2d] - ", entry.Date.Day(), entry.Date.Month(),
								entry.Date.Year(), entry.Date.Hour(), entry.Date.Minute(),
								entry.Date.Second())
							buff = bytes.NewBufferString(dateString)
						}else{
							buff = new(bytes.Buffer)
						}
						buff.Write(entry.Buff)
						conn.Write(buff.Bytes())
						conn.Close()
					}
				}
			}
		}()
		return err
	}
}

func (sink *RawSocketSink) IsOpen() bool {
	if (*sink).openChan == nil {
		return false
	} else {
		select {
			case <-(*sink).openChan:
				return false
			default:
				return true
		}
	}
}

func (sink *RawSocketSink) Close() (err error) {
	if !sink.IsOpen() {
		return fmt.Errorf("Error: you can't close a closed sink!")
	} else {
		close((*sink).openChan)
		(*sink).openChan = nil
		return err
	}
}

func (sink *RawSocketSink) SetShowDate(value bool) {
	(*sink).showDate = value
}
