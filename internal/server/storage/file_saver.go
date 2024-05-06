package storage

import (
	"os"
	"sync"
	"time"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

type StorageSaver interface {
	Save() error
}

type PeriodicSaver struct {
	ticker       *time.Ticker
	done         chan struct{}
	storageSaver StorageSaver
	log          logging.Logger
	period       time.Duration
	wg           sync.WaitGroup
}

type FileSaver struct {
	metricsStorage Repository
	logger         logging.Logger
	fileName       string
}

func (s *FileSaver) Save() error {
	const perms = 0644
	f, err := os.OpenFile(s.fileName, os.O_CREATE|os.O_WRONLY, perms)
	defer func() {
		if f != nil {
			err = f.Sync()
			if err != nil {
				s.logger.Errorf("Failed to sync storage file")
			}

			err = f.Close()
			if err != nil {
				s.logger.Errorf("Failed to close storage file")
			}
		}
	}()

	if err != nil {
		return err
	}

	return s.metricsStorage.Save(f)
}

func (s *PeriodicSaver) Run() {
	s.wg.Add(1)
	s.ticker = time.NewTicker(s.period)
	go func() {
		defer s.wg.Done()
	out:
		for {
			select {
			case <-s.done:
				err := s.storageSaver.Save()
				if err != nil {
					s.log.Errorf("Saving data before exit failed: %v", err)
				}
				break out
			case <-s.ticker.C:
				err := s.storageSaver.Save()
				if err != nil {
					s.log.Errorf("Saving data failed: %v", err)
				}
			}
		}
	}()
}

func (s *PeriodicSaver) Stop() {
	close(s.done)
	s.wg.Wait()
}

func NewPeriodicSaver(period time.Duration, log logging.Logger, storageSaver StorageSaver) *PeriodicSaver {
	return &PeriodicSaver{period: period, done: make(chan struct{}), log: log, storageSaver: storageSaver}
}

func NewFileSaver(m Repository, fileName string, log logging.Logger) *FileSaver {
	s := FileSaver{logger: log, fileName: fileName, metricsStorage: m}
	return &s
}
