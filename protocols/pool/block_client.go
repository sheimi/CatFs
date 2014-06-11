package pool

import (
	"errors"
	"github.com/proj-223/CatFs/config"
	"log"
	"net"
)

const (
	BLOCK_BUFFER_SIZE_PACED  = 1 << 11
	BLOCK_BUFFER_SIZE  = 1 << 10
	BLOCK_REQUEST_SIZE = 100
	BLOCK_SEND_SIZE    = 1 << 9
	DEFAULT_CHAN_SIZE  = 10
)

const (
	REQUEST_SEND_BLOCK = iota
	REQUEST_GET_BLOCK
)

const (
	BLOCK_FINISHED = iota
	BLOCK_NOT_FINISHED
)

var (
	RESPONSE_PELEASE_SEND = []byte("ack")

	ErrShutdown = errors.New("Operation Error")
)

type BlockRequest struct {
	TransID     string // It is a UUID
	RequestType byte   // It is an int
}

type BlockStruct struct {
	Finished bool
	Data     []byte
}

type BlockClient struct {
	addr string
}

func NewBlockClient(index int, conf *config.MachineConfig) *BlockClient {
	return &BlockClient{
		addr: conf.BlockServerAddr(index),
	}
}

func (self *BlockClient) SendBlockAll(data []byte, transID string) error {
	sendChan := make(chan []byte, DEFAULT_CHAN_SIZE)
	go self.SendBlock(sendChan, transID)
	sliceStart := 0
	for sliceStart+BLOCK_BUFFER_SIZE < len(data) {
		sendChan <- data[sliceStart : sliceStart+BLOCK_BUFFER_SIZE]
		sliceStart += BLOCK_BUFFER_SIZE
	}
	sendChan <- data[sliceStart:]
	close(sendChan)
	return nil
}

func (self *BlockClient) SendBlock(c chan []byte, transID string) {
	conn, err := net.Dial("tcp", self.addr)
	if err != nil {
		// if there is an error, close channel
		log.Println(err.Error())
		close(c)
		return
	}
	defer conn.Close()

	requestBytes := ToBytes(&BlockRequest{
		TransID:     transID,
		RequestType: REQUEST_SEND_BLOCK,
	})

	_, err = conn.Write(requestBytes)
	if err != nil {
		// if there is an error, close channel
		log.Println(err.Error())
		close(c)
		return
	}
	buf := make([]byte, BLOCK_REQUEST_SIZE)
	// get the ack from server
	_, err = conn.Read(buf)
	if err != nil {
		log.Printf("Error read: %s\n", err.Error())
		close(c)
		return
	}
	for {
		b, ok := <-c
		if !ok {
			// it is done
			break
		}
		_, err = conn.Write(b)
		if err != nil {
			// if there is an error, close channel
			log.Println(err.Error())
			return
		}
	}
	return
}

func (self *BlockClient) GetBlock(c chan []byte, transID string) {
	// any way, close the chanel
	defer close(c)
	conn, err := net.Dial("tcp", self.addr)
	// any way, close the conn
	defer conn.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}

	requestBytes := ToBytes(&BlockRequest{
		TransID:     transID,
		RequestType: REQUEST_GET_BLOCK,
	})
	_, err = conn.Write(requestBytes)

	buf := make([]byte, BLOCK_BUFFER_SIZE_PACED)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// Log the error
			log.Printf("Error read: %s\n", err.Error())
			// send nil to channel
			c <- nil
			return
		}
		var bs BlockStruct
		err = FromBytes(buf[:n], &bs)
		if err != nil {
			// Log the error the return
			log.Println(err.Error())
			c <- nil
			return
		}
		if bs.Finished {
			break
		}
		// TODO Question potential racing condition
		// Make the chan limited to 1
		c <- bs.Data
		// ack
		_, err = conn.Write(RESPONSE_PELEASE_SEND)
		if err != nil {
			// Log the error and return
			log.Printf("Error write: %s\n", err.Error())
			c <- nil
			return
		}
	}
}
