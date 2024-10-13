package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"log/slog"
	"sync"

	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
)

type FileServerConfig struct {
	ListenAddr     string
	KeyTransformer storage.KeyTransformer
	StorageRoot    string
	Transport      p2p.Transport
	Context        context.Context
	Logger         *slog.Logger
	Id             string
}

type FileServer struct {
	FileServerConfig
	nodesMu *sync.Mutex
	peers   map[string]p2p.Peer

	store        *storage.Store
	shutdownFunc func()
	ctx          context.Context
}

type BaseCommand struct {
	Key string
}

type ReadCommand struct {
	BaseCommand
}

type StoreCommand struct {
	BaseCommand
	Data []byte
}

type Message struct {
	Payload interface{}
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	msg := Message{
		Payload: "storage bytes",
	}

	for _, peer := range fs.peers {
		go func() {
			buf := new(bytes.Buffer)
			if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
				fs.Logger.Error("encode error", "msg", err.Error())
				return
			}

			if err := peer.Send(buf.Bytes()); err != nil {
				fs.Logger.Error("send error", "msg", err.Error())
			}
		}()
	}

	return nil
}

func NewFileServer(config FileServerConfig) *FileServer {
	var server *FileServer
	defer func() {
		if server != nil && !gobTypesRegistered {
			server.registerGobTypes()
		}
	}()

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
		nodesMu:          new(sync.Mutex),
	}

	server.registerOnPeerCallbacks()

	return server
}

func (s *FileServer) Start(addrs ...string) error {
	s.Logger.Info("Server started", "address", s.ListenAddr)
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

// func (fs *FileServer) broadcast(p *Message) error {
// 	if len(fs.peers) == 0 {
// 		return nil
// 	}

// 	peers := make([]io.Writer, 0)
// 	for _, peer := range fs.peers {
// 		peers = append(peers, peer)
// 	}

// 	m2 := io.MultiWriter(peers...)
// 	return gob.NewEncoder(m2).Encode(p)
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
	case *ReadCommand:
		s.Logger.Info("read command", "key", v.Key, "value", v)
	case *StoreCommand:
		s.Logger.Info("store command", "key", v.Key, "len", len(v.Data))
	default:
		s.Logger.Warn("uknown command message", "command", v)
	}
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

func (s *FileServer) registerOnPeerCallbacks() {
	s.Transport.OnPeerConnected(func(p p2p.Peer) {
		s.nodesMu.Lock()
		defer s.nodesMu.Unlock()

		s.peers[p.RemoteAddr().String()] = p
		var addr string = p.RemoteAddr().String()

		// if !p.Inbound() {
		// 	addr = p.LocalAddr().String()
		// }
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

var mxGobTypesRegistered sync.RWMutex
var gobTypesRegistered = false

func (s *FileServer) registerGobTypes() {
	s.Logger.Info("Registering GOB types")
	mxGobTypesRegistered.Lock()
	gobTypesRegistered = true
	mxGobTypesRegistered.Unlock()
}
