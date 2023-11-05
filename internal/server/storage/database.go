package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
)

type MetricType int

const (
	Gauge MetricType = iota
	Counter
)

type PGQueryConfig struct {
	updateQuery string
	deleteQuery string
	getOneQuery string
	getAllQuery string
}

func BuildPGConfig(tableName string) PGQueryConfig {

	updateQuery := "INSERT INTO %s (name, value)" +
		" VALUES ($1, $2)" +
		" ON CONFLICT (name) DO UPDATE" +
		" SET value = excluded.value"

	getOneQuery := "SELECT value FROM %s WHERE name = $1 LIMIT 1"

	getAllQuery := "SELECT name, value FROM %s"

	deleteQuery := "DELETE from %s where name = $1 ORDER BY name"

	config := PGQueryConfig{
		updateQuery: fmt.Sprintf(updateQuery, tableName),
		getOneQuery: fmt.Sprintf(getOneQuery, tableName),
		getAllQuery: fmt.Sprintf(getAllQuery, tableName),
		deleteQuery: fmt.Sprintf(deleteQuery, tableName),
	}

	return config
}

type PGMetricRepository struct {
	dbConfig   config.DBConfig
	metricType MetricType
	config     PGQueryConfig
	db         *sql.DB
}

func NewPGMetricRepository(dbConfig config.DBConfig, metricType MetricType, db *sql.DB) (*PGMetricRepository, error) {

	var createTableQuery string
	var tableName string
	var valueType string
	if metricType == Gauge {
		tableName = "Gauge_Metrics"
		valueType = "DOUBLE PRECISION"
		createTableQuery = "CREATE TABLE IF NOT EXISTS %s (" +
			" name VARCHAR(250) PRIMARY KEY," +
			" value %s" +
			")"
	} else if metricType == Counter {
		tableName = "Counter_Metrics"
		valueType = "BIGINT"
		createTableQuery = "CREATE TABLE IF NOT EXISTS %s (" +
			" name VARCHAR(250) PRIMARY KEY," +
			" value %s" +
			")"
	}

	createTableQuery = fmt.Sprintf(createTableQuery, tableName, valueType)

	ctx, cancel := context.WithTimeout(context.Background(), dbConfig.PingTimeout)
	defer cancel()

	_, err := db.ExecContext(ctx, createTableQuery)

	if err != nil {
		return nil, err
	}

	return &PGMetricRepository{dbConfig: dbConfig, metricType: metricType, config: BuildPGConfig(tableName), db: db}, nil
}

func NewPGMetricsStorage(config config.DBConfig) (*CommonMetricsStorage, error) {
	storage := CommonMetricsStorage{storage: make(map[string]Repository)}

	genericError := "failed to create postgres metrics storage: %w"

	db, err := sql.Open(config.DriverName, config.ConnString)
	if err != nil {
		return nil, fmt.Errorf(genericError, fmt.Errorf("failed to open database: %w", err))
	}

	repo, err := NewPGMetricRepository(config, Gauge, db)
	if err != nil {
		return nil, fmt.Errorf(genericError, fmt.Errorf("failed to create gauge metrics repository: %w", err))
	}

	storage.AddRepository("gauge", repo)

	repo, err = NewPGMetricRepository(config, Counter, db)
	if err != nil {
		return nil, fmt.Errorf(genericError, fmt.Errorf("failed to create counter metrics repository: %w", err))
	}

	storage.AddRepository("counter", repo)

	return &storage, nil
}

type BadDataError error
type DatabaseError error

type RowQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *PGMetricRepository) getCounterIncDelta(ctx context.Context, db RowQuery, name string, val string) (string, error) {
	res := db.QueryRowContext(ctx, r.config.getOneQuery, name)
	var counterVal int64
	err := res.Scan(&counterVal)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return val, nil
		}
		return "", err
	}

	delta, err := strconv.ParseInt(val, 10, 64)

	if err != nil {
		return "", err
	}

	counterVal += delta

	return strconv.FormatInt(counterVal, 10), nil
}

func (r *PGMetricRepository) AddMetricsBulk(metrics []common.MetricJSON) error {
	tx, err := r.db.Begin()

	if err != nil {
		return DatabaseError(fmt.Errorf("failed to begin transaction: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
	defer cancel()

	stmt, err := tx.PrepareContext(ctx, r.config.updateQuery)

	if err != nil {
		tx.Rollback()
		return DatabaseError(fmt.Errorf("failed to prepare query %v: %w", r.config.updateQuery, err))
	}

	defer stmt.Close()

	for i, m := range metrics {
		val, err := m.GetData()
		if err != nil {
			return BadDataError(fmt.Errorf("failed go get metric %v value: %w", m.ID, err))
		}

		updateVal := val
		if r.metricType == Counter {
			updateVal, err = r.getCounterIncDelta(ctx, tx, m.ID, val)
			if err != nil {
				return DatabaseError(err)
			}
		}

		_, err = stmt.ExecContext(ctx, m.ID, updateVal)

		if err != nil {
			tx.Rollback()
			return DatabaseError(fmt.Errorf("failed to execute query %v for metric %v: %w", r.config.updateQuery, m.ID, err))
		}

		metrics[i].SetData(updateVal)
	}

	err = tx.Commit()

	if err != nil {
		return DatabaseError(fmt.Errorf("failed to commit metrics update: %w", err))
	}

	return nil
}

func (r *PGMetricRepository) AddOrUpdate(key string, val string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
	defer cancel()

	updateVal := val
	var err error
	if r.metricType == Counter {
		updateVal, err = r.getCounterIncDelta(ctx, r.db, key, val)
		if err != nil {
			return "", err
		}
	}

	_, err = r.db.ExecContext(ctx, r.config.updateQuery, key, updateVal)

	if err != nil {
		return "", fmt.Errorf("failed to execute add/update query %v: %w", r.config.updateQuery, err)
	}

	return updateVal, nil
}

func (r *PGMetricRepository) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, r.config.deleteQuery, key)

	if err != nil {
		return fmt.Errorf("failed to delete metirc: %w", err)
	}

	return nil
}

func (r *PGMetricRepository) Get(key string) (Metric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
	defer cancel()

	res := r.db.QueryRowContext(ctx, r.config.getOneQuery, key)

	var metricValue any
	var getMetric func(value any) Metric

	if r.metricType == Counter {
		metricValue = new(int64)
		getMetric = func(value any) Metric {
			v := common.Int{Val: *value.(*int64)}
			return &CounterMetric{Int: v}
		}
	} else if r.metricType == Gauge {
		metricValue = new(float64)
		getMetric = func(value any) Metric {
			v := common.Float{Val: *value.(*float64)}
			return &GaugeMetric{Float: v}
		}
	}

	if err := res.Scan(metricValue); err != nil {
		return nil, fmt.Errorf("failed to extract data from query: %w", err)
	}

	return getMetric(metricValue), nil
}

func getMetric(val string, mtype MetricType) (Metric, error) {
	var m Metric
	if mtype == Gauge {
		m = &GaugeMetric{}
	} else if mtype == Counter {
		m = &CounterMetric{}
	}

	err := m.UpdateValue(val)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (r *PGMetricRepository) ForEachMetric(callback func(name string, m Metric) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
	defer cancel()

	row, err := r.db.QueryContext(ctx, r.config.getAllQuery)

	if err != nil {
		return fmt.Errorf("failed to query all repo metrics: %w", err)
	}

	defer row.Close()

	for row.Next() {
		var name string
		var value string

		err := row.Scan(&name, &value)

		if err != nil {
			return fmt.Errorf("failed to get metric value: %w", err)
		}

		metric, err := getMetric(value, r.metricType)

		if err != nil {
			return fmt.Errorf("failed to create metric from value %v: %w", value, err)
		}

		err = callback(name, metric)

		if err != nil {
			return fmt.Errorf("user function failed for metric %v with value %v: %w", name, value, err)
		}
	}

	if err = row.Err(); err != nil {
		return fmt.Errorf("failed to iterate all metrics: %w", err)
	}

	return nil
}

func (r *PGMetricRepository) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (r *PGMetricRepository) UnmarshalJSON(data []byte) error {
	return errors.New("not implemented")
}
