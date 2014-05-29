package data

import (
	"github.com/proj-223/CatFs/config"
	proc "github.com/proj-223/CatFs/protocols"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

const (
	DEFAULT_CHAN_SIZE = 10
)

type DataServer struct {
	pool *proc.ClientPool
	conf *config.MachineConfig
	// index of this data server
	index       int
	blockServer *proc.BlockServer
}

// Prepare send a block to datanode
func (self *DataServer) PrepareSendBlock(param *proc.PrepareBlockParam, lease *proc.CatLease) error {
	// send prepare to next data server
	deliverChan, err := self.prepareNext(param)
	if err != nil {
		return err
	}
	writeDiskChan := make(chan []byte, DEFAULT_CHAN_SIZE)
	done := make(chan bool)
	lease = proc.NewCatLease()
	trans := proc.NewReadTransaction(lease.ID, done, deliverChan, writeDiskChan)
	// init data receiver
	self.blockServer.StartTransaction(trans)
	return nil
}

// Wait util blocks reach destination
// The block will be sent by a pipeline
func (self *DataServer) SendingBlock(param *proc.SendingBlockParam, succ *bool) error {
	panic("to do")
}

// Get the block from data server
// Will start an tcp connect to request block
func (self *DataServer) GetBlock(param *proc.GetBlockParam, lease *proc.CatLease) error {
	panic("to do")
}

func (self *DataServer) addr() string {
	return self.conf.DataServerAddr(self.index)
}

// go routine to init the data rpc server
func (self *DataServer) initRPCServer(done chan error) {
	server := rpc.NewServer()
	server.Register(proc.DataProtocol(self))
	l, err := net.Listen("tcp", self.addr())
	if err != nil {
		done <- err
		return
	}
	log.Printf(RPC_START_MSG, self.index, self.addr())
	err = http.Serve(l, server)
	done <- err
}

func (self *DataServer) initBlockServer(done chan error) {
	err := self.blockServer.Serve()
	done <- err
}
