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
	Key     string
	Size    uint64
	Offsets []uint64
}

type ReadResponse struct {
	Key    string
	Offset uint64
	Data   []byte
}

type StoreUpdatedCommand struct {
	Timestamp time.Time
	Key       string
	Stat      *storage.DataStat
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

		fs.store.Write(key, bytes.NewReader(encoded), storage.UsingComputedPieceSize(), storage.UsingOffset(uint64(read)))
		read += int64(len(encoded))
	}

	stat, err := fs.store.DataStatFor(key, storage.UsingPieceSize(uint64(fs.StreamChunkSize)))
	if err != nil {
		return err
	}

	cmd := StoreUpdatedCommand{
		Key:       key,
		Timestamp: time.Now(),
		Stat:      stat,
	}

	if err := fs.broadcastCommand(cmd); err != nil {
		return err
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

func (fs *FileServer) broadcastCommand(p any) error {
	if len(fs.peers) == 0 {
		return nil
	}

	msg := Message{
		Payload:   p,
		Timestamp: time.Now(),
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
		return err
	}
	data := buf.Bytes()

	for _, peer := range fs.peers {
		if err := peer.Send(data); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileServer) send(to string, p any) error {
	peer, ok := fs.peers[to]
	if !ok {
		fs.Logger.Warn("peer not found", "addr", to)
		return nil
	}

	msg := Message{
		Payload:   p,
		Timestamp: time.Now(),
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
		return err
	}
	data := buf.Bytes()
	return peer.Send(data)
}

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

				if err := s.handleMessage(&msg, rpc.From); err != nil {
					s.Logger.Error("handling error", "msg", err.Error())
				}
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *FileServer) handleMessage(msg *Message, from string) error {
	switch v := msg.Payload.(type) {
	case ReadCommand:
		go s.handleReadCommand(&v, from)
	case StoreUpdatedCommand:
		go s.handleStoreUpdatedCommand(&v)
	case ReadResponse:
		go s.handleReadResponse(&v)
	default:
		s.Logger.Warn("unknown command message", "command", fmt.Sprintf("%+v", v))
	}
	return nil
}

func (s *FileServer) handleReadResponse(cmd *ReadResponse) {
	println(cmd)
}

func (s *FileServer) handleReadCommand(cmd *ReadCommand, from string) {
	for _, v := range cmd.Offsets {
		buf := make([]byte, cmd.Size)
		ds, err := s.store.Read(cmd.Key, storage.UsingOffset(v), storage.UsingPieceSize(cmd.Size))
		if err != nil {
			s.Logger.Error("read error", "msg", err.Error(), "piece-offset", v)
		}
		if _, err := ds.Read(buf); err != nil {
			s.Logger.Error("read error2", "msg", err.Error(), "piece-offset", v)
		}

		resp := ReadResponse{
			Key:    cmd.Key,
			Offset: v,
			Data:   buf,
		}

		if err := s.send(from, resp); err != nil {
			s.Logger.Error("send error", "msg", err.Error())
		}
	}
}

func (s *FileServer) handleStoreUpdatedCommand(cmd *StoreUpdatedCommand) {
	deltas, err := s.store.GetMissingDeltas(cmd.Key, cmd.Stat)
	if err != nil {
		s.Logger.Error("error wihle store update", "msg", err.Error())
		return
	}

	if len(deltas) == 0 {
		return
	}

	if err := s.retrieveChunks(cmd.Key, cmd.Stat.PieceSize, deltas...); err != nil {
		s.Logger.Error("chunk retrieval error", "msg", err.Error())
	}
}

func (s *FileServer) retrieveChunks(key string, size uint64, offsets ...uint64) error {
	cmd := ReadCommand{
		Key:     key,
		Size:    size,
		Offsets: offsets,
	}

	if err := s.broadcastCommand(cmd); err != nil {
		return err
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

func (s *FileServer) registerTransportCallbacks() {
	s.Transport.OnPeerConnected(func(p p2p.Peer) {
		s.nodesMu.Lock()
		defer s.nodesMu.Unlock()

		var addr = p.RemoteAddr().String()
		s.peers[addr] = p
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
	gob.Register(StoreUpdatedCommand{})
	gob.Register(ReadCommand{})
	gob.Register(ReadResponse{})
}
