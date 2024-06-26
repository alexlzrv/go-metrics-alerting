package server

import (
	"context"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/server/grpc"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/middleware"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/server/config"
	"github.com/mayr0y/animated-octo-couscous.git/internal/pkg/storage"
	"github.com/sirupsen/logrus"
)

func StartListener(parent context.Context, c *config.ServerConfig) {
	logrus.Info("Init store...")
	logrus.Infof("ServerAddress: %v", c.ServerAddress)
	logrus.Infof("StoreInterval: %v", c.StoreInterval)
	logrus.Infof("Restore: %v", c.Restore)
	logrus.Infof("FileStoragePath: %v", c.FileStoragePath)

	ctx, stop := signal.NotifyContext(parent,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	var (
		metricStore storage.Store
		err         error
	)

	if c.DatabaseDSN != "" {
		metricStore, err = storage.NewDBMetrics(c.DatabaseDSN)
	} else if c.FileStoragePath != "" {
		metricStore, err = storage.NewMetricsFile(c.FileStoragePath, time.Duration(c.StoreInterval)*time.Second)
	} else {
		metricStore = storage.NewMetrics()
	}

	if err != nil {
		logrus.Errorf("Error init store: %v", err)
		return
	}

	defer metricStore.Close()

	logrus.Info("Init store successfully")

	var (
		mux = chi.NewRouter()
		srv = &http.Server{
			Addr:    c.ServerAddress,
			Handler: mux,
		}
		grpcSrv = grpc.Server{
			Address: c.GRPCAddress,
		}
	)

	mux.Use(
		middleware.LoggingMiddleware,
		middleware.CryptMiddleware(c.SignKeyByte),
	)

	if c.PrivateKey != "" {
		privateKey, err := c.GetPrivateKey()
		if err != nil {
			logrus.Errorf("Error get private key: %v", err)
		}
		mux.Use(middleware.DecryptMiddleware(privateKey))
	}

	RegisterHandlers(mux, metricStore)

	if c.Restore {
		if err = metricStore.LoadMetrics(c.FileStoragePath); err != nil {
			logrus.Errorf("Error update metric from file %v", err)
		}
	}

	if c.StoreInterval > 0 {
		storeInterval := time.NewTicker(time.Duration(c.StoreInterval) * time.Second)
		defer storeInterval.Stop()
		go func() {
			for range storeInterval.C {
				err = metricStore.SaveMetrics(c.FileStoragePath)
				if err != nil {
					logrus.Errorf("Error save metric from file %v", err)
				}
			}
		}()
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		logrus.Info("Server is running...")
		if err = srv.ListenAndServe(); err != nil {
			logrus.Fatalf("Error with server running: %v", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcSrv.Start(ctx, metricStore); err != nil {
			logrus.Fatalf("error on listen and serve GRPC server: %s", err)
		}
	}()

	if err = srv.Shutdown(ctx); err != nil {
		logrus.Errorf("server shutdown %v", err)
		return
	}

	wg.Wait()
}
