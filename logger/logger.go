package logger

import (
	"time"
	"os"
	"syscall"
	"golang.org/x/exp/inotify"
	"fmt"
	"io"
	"bufio"
	"github.com/Manaphy91/nettee/logger/sink"
)

type Logger struct{
	sinkLst []sink.Sink
	file *os.File
	watcher *inotify.Watcher
	printLogEntry bool
	logStopped chan struct{}
}

func New(filename *string) (logger *Logger, err error){
	logger = &Logger{}
	if filename == nil {
		(*logger).file = os.Stdin
	} else {
		(*logger).file, err = os.Open(*filename)
		if err != nil {
			return nil, fmt.Errorf("Error opening file %s: %s", filename, err)
		}
		(*logger).watcher, err = inotify.NewWatcher()
		if err != nil {
			panic(err)
		}
		(*logger).watcher.AddWatch((*logger).file.Name(), syscall.IN_CLOSE_WRITE)
	}
	(*logger).printLogEntry = true
	(*logger).logStopped = make(chan struct{})
	return logger, err
}

func (logger *Logger) AddSink(sink sink.Sink) (err error){
	if sink == nil {
		return fmt.Errorf("Error you can't add to a logger a nil sink!")
	}else{
		if err := sink.Open(); err != nil {
			panic(err)
		}
		(*logger).sinkLst = append((*logger).sinkLst, sink)
		return err
	}
}

func (logger *Logger) Start() (err error) {
	if logger == nil {
		return fmt.Errorf("Error you can't call Start method of a nil logger!")
	} else {
		var readFromFile func () ([]byte, error)
		if (*logger).file == os.Stdin {

			reader := bufio.NewReader(os.Stdin)
			readFromFile = func () ([]byte, error) {
				return reader.ReadBytes('\n')
			}

			go func() {
				for {
					select {
						case <-(*logger).logStopped:
							// if logger was stopped by the user then close
							// every sink associated with it 
							for _, sinkValue := range (*logger).sinkLst {
								sinkValue.Close()
							}
							return
						default:
							readBuff, err := readFromFile()
							if err == io.EOF {
								// if os.Stdin was closed then close every sink
								// associated with logger
								for _, sinkValue := range (*logger).sinkLst {
									sinkValue.Close()
								}
								// and after that close the logger
								logger.Stop()
								return
							} else if err != nil {
								fmt.Printf("Error reading from file: %s", err)
								os.Exit(1)
							}
							entry := sink.LogEntry{readBuff, time.Now()}
							for sinkIndex, sinkValue := range (*logger).sinkLst {
								select {
									case sinkValue.InFlowChan() <- entry:

									case <-time.After(300 * time.Millisecond):
										logger.detachSink(uint16(sinkIndex))
								}
							}

							// if no one sink is still alive,
							// then stop the logger
							if len((*logger).sinkLst) == 0 {
								logger.Stop()
								return
							}

							if (*logger).printLogEntry {
								fmt.Printf(string(readBuff))
							}
					}
				}
			}()

		} else {

			var fInfo os.FileInfo
			fInfo, err = (*logger).file.Stat()
			if err != nil {
				return fmt.Errorf("Error reading stats from file: %s", err)
			}

			var pos int64 = fInfo.Size()
			readFromFile = func () ([]byte, error) {
				fInfo, err = (*logger).file.Stat()
				if err != nil {
					return nil, fmt.Errorf("Error reading stats from file: %s", err)
				}
				var tmpBuff []byte
				// to handle correctly upcoming changes of the watched file you need to know if
				if fInfo.Size() <= pos {
					// - the file was overwritten with a lighter content 
					// - or the file was overwritten with a content of the same size
					(*logger).file.Seek(0, 0)
					tmpBuff = make([]byte, fInfo.Size())
				} else {
					// - or a new content was appended to the file
					(*logger).file.Seek(pos, 0)
					tmpBuff = make([]byte, fInfo.Size() - pos)
				}
				pos = fInfo.Size()
				(*logger).file.Read(tmpBuff)
				return tmpBuff, err
			}

			go func() {
				for {
					select {
						case <-(*logger).logStopped:
							// if logger was stopped by the user then close
							// every sink associated with it 
							for _, sinkValue := range (*logger).sinkLst {
								sinkValue.Close()
							}
							return
						case <-(*logger).watcher.Event:
							readBuff, err := readFromFile()
							if err != nil {
								fmt.Printf("Error reading from file: %s", err)
								os.Exit(1)
							}
							entry := sink.LogEntry{readBuff, time.Now()}
							for sinkIndex, sinkValue := range (*logger).sinkLst {
								select {
									case sinkValue.InFlowChan() <- entry:

									case <-time.After(300 * time.Millisecond):
										logger.detachSink(uint16(sinkIndex))
								}
							}

							// if no one sink is still alive,
							// then stop the logger
							if len((*logger).sinkLst) == 0 {
								logger.Stop()
								return
							}

							if (*logger).printLogEntry {
								fmt.Printf(string(readBuff))
							}
					}
				}
			}()
		}
		return err
	}
}

func (logger *Logger) Stop() (err error) {
	if logger == nil {
		return fmt.Errorf("Error you can't call Stop method on a nil logger!")
	} else {
		select {
			case <-(*logger).logStopped:
				return fmt.Errorf("Error you can't call Stop method on a logger just stopped!")
			default:
				close((*logger).logStopped)
				(*logger).file.Close()
				return err
		}
	}
}

func (logger *Logger) SetPrintLogEntry(printLogEntry bool) {
	(*logger).printLogEntry = printLogEntry
}

func (logger *Logger) detachSink(index uint16) {
	close((*logger).sinkLst[index].InFlowChan())
	(*logger).sinkLst = append((*logger).sinkLst[:index], (*logger).sinkLst[index:]...)
}

func (logger *Logger) StopChan() (chan struct{}) {
	return (*logger).logStopped
}
