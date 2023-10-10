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
	period       time.Duration
	wg           sync.WaitGroup
	done         chan struct{}
	log          logging.Logger
	storageSaver StorageSaver
}

type FileSaver struct {
	logger         logging.Logger
	fileName       string
	metricsStorage MetricsStorage
}

func (s *FileSaver) Save() error {
	f, err := os.OpenFile(s.fileName, os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		if f != nil {
			f.Sync()
			err := f.Close()
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
			case <-time.After(s.period):
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

func NewFileSaver(m MetricsStorage, fileName string, log logging.Logger) *FileSaver {
	s := FileSaver{logger: log, fileName: fileName, metricsStorage: m}
	return &s
}
