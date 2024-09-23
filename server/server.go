package server

import (
	"context"
	"log"
	"sync"

	"github.com/brinestone/dfs/p2p"
	"github.com/brinestone/dfs/storage"
	// "github.com/rodaine/table"
)

var logger = log.New(log.Writer(), "[FileServer]\t", log.LstdFlags)

type FileServerConfig struct {
	ListenAddr   string
	TransformKey storage.KeyTransformer
	StorageRoot  string
	Transport    p2p.Transport
	Context      context.Context
	Nodes        []string
	Logger       *log.Logger
}

type FileServer struct {
	FileServerConfig
	peerLock sync.Mutex
	nodes    map[string]p2p.Peer

	store        *storage.Store
	shutdownFunc func()
	ctx          context.Context
}

func NewFileServer(config FileServerConfig) *FileServer {
	// defer func() {
	// 	tbl := table.New("Property", "Value")
	// }()
	ctx, cancel := context.WithCancel(config.Context)
	if config.Logger != nil {
		logger = config.Logger
	}
	storageConfig := storage.StoreConfig{
		TransformKey: config.TransformKey,
		Root:         config.StorageRoot,
		Logger:       logger,
	}
	return &FileServer{
		store:            storage.NewStore(storageConfig),
		FileServerConfig: config,
		shutdownFunc:     cancel,
		ctx:              ctx,
		nodes:            make(map[string]p2p.Peer),
	}
}

func (s *FileServer) Start() error {
	logger.Printf("Server listening on %s\n", s.ListenAddr)
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNodes()
	s.loop()

	return nil
}

func (s *FileServer) Shutdown() error {
	logger.Println("Server shutting down...")
	defer s.shutdownFunc()
	defer logger.Println("Server shutdown successful ✅")
	if err := s.Transport.Close(); err != nil {
		return err
	}
	return nil
}

func (s *FileServer) loop() {
	for {
		select {
		case msg, ok := <-s.Transport.Consume():
			if ok {
				logger.Println(msg)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *FileServer) bootstrapNodes() error {
	for _, addr := range s.Nodes {
		if len(addr) == 0 {
			continue
		}
		go func(a string) {
			if err := s.Transport.Dial(a); err != nil {
				logger.Printf("%+v ❌", err)
			}

		}(addr)
	}
	return nil
}
