package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/brinestone/dfs/encoding"
	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
)

type FileServerConfig struct {
	ListenAddr      string
	KeyTransformer  storage.KeyTransformer
	StorageRoot     string
	Transport       p2p.Transport
	Context         context.Context
	Logger          *slog.Logger
	Id              string
	StreamChunkSize int64
	Encoder         encoding.Encoder
	Decoder         encoding.Decoder
}

type FileServer struct {
	FileServerConfig
	nodesMu sync.Mutex
	peers   map[string]p2p.Peer

	store        *storage.Store
	shutdownFunc func()
	ctx          context.Context
}

type ReadCommand struct {
	Key string
}

type StoreCommand struct {
	Offset   int64
	Total    int64
	Key      string
	Data     []byte
	Checksum string
}

type Message struct {
	Timestamp time.Time
	Payload   any
}

// Writes the data to the disk.
func (fs *FileServer) StoreData(key string, size int64, in io.Reader) error {
	var read int64 = 0

	for read < size {
		buf := make([]byte, fs.StreamChunkSize)
		n, err := in.Read(buf)
		if n == 0 && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		encoded, err := fs.Encoder.Encode(buf)
		if err != nil {
			return err
		}

		fs.store.Write(read, key, bytes.NewReader(encoded))
		read += int64(len(encoded))
	}
	return nil
}

func NewFileServer(config FileServerConfig) (*FileServer, error) {
	var server *FileServer

	ctx, cancel := context.WithCancel(config.Context)
	storageConfig := storage.StoreConfig{
		TransformKey: config.KeyTransformer,
		Root:         config.StorageRoot,
		Logger:       config.Logger,
	}

	server = &FileServer{
		store:            storage.NewStore(storageConfig),
		FileServerConfig: config,
		shutdownFunc:     cancel,
		ctx:              ctx,
		peers:            make(map[string]p2p.Peer),
		nodesMu:          sync.Mutex{},
	}

	server.registerTransportCallbacks()

	return server, nil
}

func (s *FileServer) Start(addrs ...string) error {
	s.Logger.Info("Server started", "addr", s.ListenAddr)
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if err := s.HostsAvailable(addrs...); err != nil {
		s.Logger.Error(err.Error())
	}
	s.loop()

	return nil
}

func (s *FileServer) Shutdown() error {
	s.Logger.Info("Server shutting down...")
	defer s.shutdownFunc()
	defer s.Logger.Info("Server shutdown successful ✅")

	if err := s.Transport.Close(); err != nil {
		return err
	}
	return nil
}

// func (fs *FileServer) broadcastCommand(p any) error {
// 	if len(fs.peers) == 0 {
// 		return nil
// 	}

// 	msg := Message{
// 		Payload:   p,
// 		Timestamp: time.Now(),
// 	}

// 	buf := new(bytes.Buffer)
// 	if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
// 		return err
// 	}
// 	data := buf.Bytes()

// 	for _, peer := range fs.peers {
// 		if err := peer.Send(data); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (s *FileServer) loop() {
	defer func() {
		s.Transport.Close()
	}()

	for {
		select {
		case rpc, ok := <-s.Transport.Consume():
			if ok {
				var msg Message
				r := bytes.NewReader(rpc.Payload)
				if err := gob.NewDecoder(r).Decode(&msg); err != nil {
					s.Logger.Error("decode error", "msg", err.Error())
					continue
				}

				if err := s.handleMessage(&msg); err != nil {
					s.Logger.Error("handling error", "msg", err.Error())
				}
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *FileServer) handleMessage(msg *Message) error {
	switch v := msg.Payload.(type) {
	case ReadCommand:
		s.Logger.Info("read command", "key", v.Key, "value", v)
	case StoreCommand:
		return s.handleStoreCommand(&v)
	default:
		s.Logger.Warn("unknown command message", "command", fmt.Sprintf("%+v", v))
	}
	return nil
}

func (s *FileServer) handleStoreCommand(v *StoreCommand) error {
	s.Logger.Info("store command", "key", v.Key, "len", len(v.Data), "offset", v.Offset)
	n, err := s.store.Write(v.Offset, v.Key, bytes.NewReader(v.Data))
	if err != nil {
		return err
	}
	s.Logger.Info("chunk written", "offset", v.Offset, "len", len(v.Data), "disk", n)

	return nil
}

// Connects to remote hosts
func (s *FileServer) HostsAvailable(nodes ...string) error {
	for _, addr := range nodes {
		if len(addr) == 0 {
			continue
		}
		go s.connectToRemoteHost(addr)
	}
	return nil
}

func (s *FileServer) connectToRemoteHost(addr string) {
	if err := s.Transport.Dial(addr); err != nil {
		s.Logger.Error("connect error", "msg", err.Error())
	}
}

func (s *FileServer) registerTransportCallbacks() {
	s.Transport.OnPeerConnected(func(p p2p.Peer) {
		s.nodesMu.Lock()
		defer s.nodesMu.Unlock()

		s.peers[p.RemoteAddr().String()] = p
		var addr string = p.RemoteAddr().String()
		s.Logger.Info("peer connected", "inbound", p.Inbound(), "addr", addr, "peer-size", len(s.peers))
	})

	s.Transport.OnPeerDisconnected(func(p p2p.Peer) {
		s.nodesMu.Lock()
		defer s.nodesMu.Unlock()

		delete(s.peers, p.RemoteAddr().String())

		var addr string = p.LocalAddr().String()

		if p.Inbound() {
			addr = p.RemoteAddr().String()
		}
		s.Logger.Info("peer dropped", "inbound", p.Inbound(), "addr", addr, "peer-size", len(s.peers))
	})
}

func init() {
	gob.Register(Message{})
	gob.Register(StoreCommand{})
}
