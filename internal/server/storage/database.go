// Package storage Contains server metrics storage implementations.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/errtypes"

	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
)

type MetricType int

const (
	Gauge MetricType = iota
	Counter
)

type PGQueryConfig struct {
	deleteTable string
	update      string
	delete      string
	getOne      string
	getAll      string
	deleteAll   string
}

func BuildPGQueryConfig(tableName string) PGQueryConfig {

	updateQuery := "INSERT INTO %s (name, value, delta, type)" +
		" VALUES ($1, $2, $3, $4)" +
		" ON CONFLICT (name) DO UPDATE" +
		" SET value = excluded.value," +
		" delta = excluded.delta"

	getOneQuery := "SELECT value, delta FROM %s WHERE name = $1 AND type = $2 LIMIT 1"

	getAllQuery := "SELECT name, value, delta, type FROM %s"

	deleteQuery := "DELETE FROM %s where name = $1"

	deleteAllQuery := "DELETE FROM %s"

	deleteTableQuery := "DROP table %s"

	config := PGQueryConfig{
		update:      fmt.Sprintf(updateQuery, tableName),
		getOne:      fmt.Sprintf(getOneQuery, tableName),
		getAll:      fmt.Sprintf(getAllQuery, tableName),
		delete:      fmt.Sprintf(deleteQuery, tableName),
		deleteTable: fmt.Sprintf(deleteTableQuery, tableName),
		deleteAll:   fmt.Sprintf(deleteAllQuery, tableName),
	}

	return config
}

type PGMetricRepository struct {
	log           logging.Logger
	db            *sql.DB
	retryExecutor common.RetryExecutor
	queryConfig   PGQueryConfig
	dbConfig      config.DBConfig
}

func NewPGMetricRepository(dbConfig config.DBConfig, retryExecutor common.RetryExecutor, log logging.Logger) (*PGMetricRepository, error) {
	db, err := sql.Open(dbConfig.DriverName, dbConfig.ConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	createTableQuery := "CREATE TABLE IF NOT EXISTS Metrics(" +
		" name VARCHAR(250) PRIMARY KEY," +
		" type VARCHAR(50)," +
		" value DOUBLE PRECISION," +
		" delta BIGINT," +
		" CONSTRAINT EITHER_VALUE check(value IS NOT NULL OR delta IS NOT NULL)" +
		")"

	ctx, cancel := context.WithTimeout(context.Background(), dbConfig.PingTimeout)
	defer cancel()

	_, err = db.ExecContext(ctx, createTableQuery)
	if err != nil {
		return nil, err
	}
	return &PGMetricRepository{dbConfig: dbConfig, queryConfig: BuildPGQueryConfig("Metrics"), db: db, retryExecutor: retryExecutor, log: log}, nil
}

func (r *PGMetricRepository) Close() error {
	return r.db.Close()
}

func (r *PGMetricRepository) HealthCheck() error {
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)

		defer cancel()

		return r.db.PingContext(ctx)
	}

	return r.retryExecutor.RetryOnError(work)
}

type RowQuery interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *PGMetricRepository) getCounterIncDelta(ctx context.Context, db RowQuery, name string, val string) (string, error) {
	res := db.QueryRowContext(ctx, r.queryConfig.getOne, name, metrics.CounterMetricType)
	var deltaVal sql.NullInt64
	var valValue sql.NullFloat64

	err := r.retryExecutor.RetryOnError(func() error {
		return res.Scan(&valValue, &deltaVal)
	})

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

	delta += deltaVal.Int64

	return strconv.FormatInt(delta, 10), nil
}

func (r *PGMetricRepository) AddMetricsBulk(metricsData []metrics.Metric) error {
	work := func() error {
		tx, err := r.db.Begin()

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to begin transaction: %w", err))
		}

		defer func() {
			errd := tx.Rollback()
			if errd != nil {
				r.log.Errorf("Failed to rollback tx: %v", errd)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		stmt, err := tx.PrepareContext(ctx, r.queryConfig.update)

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to prepare query %v: %w", r.queryConfig.update, err))
		}

		defer func() {
			errs := stmt.Close()
			if errs != nil {
				r.log.Errorf("Failed to close prepared statement: %v", errs)
			}
		}()

		for i := range metricsData {
			m := &metricsData[i]
			var val string
			val, err = m.GetData()
			if err != nil {
				return errtypes.MakeBadDataError(fmt.Errorf("invalid data for metric '%v': %w", m.ID, err))
			}

			if m.MType == metrics.CounterMetricType {
				var updateVal string
				updateVal, err = r.getCounterIncDelta(ctx, tx, m.ID, val)
				if err != nil {
					return errtypes.MakeServerError(err)
				}
				err = m.SetData(updateVal)
				if err != nil {
					return errtypes.MakeServerError(err)
				}
			}

			_, err = stmt.ExecContext(ctx, m.ID, m.Value, m.Delta, m.MType)

			if err != nil {
				return errtypes.MakeServerError(fmt.Errorf("failed to execute query '%v' for metric '%v': %w", r.queryConfig.update, m.ID, err))
			}
		}

		err = tx.Commit()

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to commit metrics update: %w", err))
		}

		return nil
	}

	return r.retryExecutor.RetryOnError(work)
}

func (r *PGMetricRepository) AddOrUpdate(key string, val string, mtype string) (string, error) {
	_, err := metrics.NewMetric(key, val, mtype)
	if err != nil {
		return "", errtypes.MakeBadDataError(err)
	}

	var updateValOut *string
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		updateVal := val
		updateValOut = &updateVal
		if mtype == metrics.CounterMetricType {
			updateVal, err = r.getCounterIncDelta(ctx, r.db, key, val)
			if err != nil {
				return errtypes.MakeServerError(err)
			}
		}

		var metric metrics.Metric
		metric, err = metrics.NewMetric(key, updateVal, mtype)
		if err != nil {
			return errtypes.MakeServerError(err)
		}
		_, err = r.db.ExecContext(ctx, r.queryConfig.update, metric.ID, metric.Value, metric.Delta, metric.MType)

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to execute add/update query %v: %w", r.queryConfig.update, err))
		}

		return nil
	}

	err = r.retryExecutor.RetryOnError(work)
	if err != nil {
		return "", err
	}

	return *updateValOut, nil
}

func (r *PGMetricRepository) Delete(key string) error {
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		_, err := r.db.ExecContext(ctx, r.queryConfig.delete, key)

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to delete metirc: %w", err))
		}

		return nil
	}

	return r.retryExecutor.RetryOnError(work)
}

func (r *PGMetricRepository) DeleteAll() error {
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		_, err := r.db.ExecContext(ctx, r.queryConfig.deleteAll)

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to delete all metircs: %w", err))
		}

		return nil
	}

	return r.retryExecutor.RetryOnError(work)
}

func (r *PGMetricRepository) Get(key string, mtype string) (metrics.Metric, error) {
	if !metrics.IsValidMetricType(mtype) {
		return metrics.Metric{}, errtypes.MakeServerError(fmt.Errorf("invalid metric type '%v'", mtype))
	}

	var metricOut metrics.Metric
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		res := r.db.QueryRowContext(ctx, r.queryConfig.getOne, key, mtype)

		var delta sql.NullInt64
		var value sql.NullFloat64
		if err := res.Scan(&value, &delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errtypes.MakeNotFoundError(fmt.Errorf("metric '%v' of type '%v' not found", key, mtype))
			}
			return fmt.Errorf("failed to extract data from query: %w", err)
		}

		if mtype == metrics.CounterMetricType {
			metricOut = metrics.NewCounterMetric(key, delta.Int64)
		} else if mtype == metrics.GaugeMetricType {
			metricOut = metrics.NewGaugeMetric(key, value.Float64)
		}
		return nil
	}

	return metricOut, r.retryExecutor.RetryOnError(work)
}

func (r *PGMetricRepository) GetAll() ([]metrics.Metric, error) {
	result := make([]metrics.Metric, 0)
	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		row, err := r.db.QueryContext(ctx, r.queryConfig.getAll)

		if err != nil {
			return errtypes.MakeServerError(fmt.Errorf("failed to query all repo metrics: %w", err))
		}

		defer func() {
			err = row.Close()
			if err != nil {
				r.log.Errorf("Failed to close row: %v", err)
			}
		}()

		for row.Next() {
			var name string
			var value sql.NullString
			var delta sql.NullString
			var mtype string

			err = row.Scan(&name, &delta, &value, &mtype)

			if err != nil {
				return errtypes.MakeServerError(fmt.Errorf("failed to get metric: %w", err))
			}

			var data string
			if value.Valid {
				data = value.String
			} else {
				data = delta.String
			}

			var m metrics.Metric
			m, err = metrics.NewMetric(name, data, mtype)

			if err != nil {
				return errtypes.MakeServerError(fmt.Errorf("failed to create metric from value %v: %w", value, err))
			}

			result = append(result, m)
		}

		if err = row.Err(); err != nil {
			return fmt.Errorf("failed to iterate all metrics: %w", err)
		}

		return nil
	}

	return result, r.retryExecutor.RetryOnError(work)
}

func (r *PGMetricRepository) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (r *PGMetricRepository) UnmarshalJSON(data []byte) error {
	return errors.New("not implemented")
}

func (r *PGMetricRepository) Save(w io.Writer) error {
	return errors.New("not implemented")
}

func (r *PGMetricRepository) Load(reader io.Reader) error {
	return errors.New("not implemented")
}

func (r *PGMetricRepository) Release() error {

	work := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), r.dbConfig.PingTimeout)
		defer cancel()

		_, err := r.db.ExecContext(ctx, r.queryConfig.deleteTable)

		if err != nil {
			return err
		}

		return nil
	}

	return r.retryExecutor.RetryOnError(func() error {
		return work()
	})
}

func NewDefaultDBRetryExecutor(stopCtx context.Context) *common.CommonRetryExecutor {
	errs := []error{
		pgx.ErrDeadConn,
		pgx.ErrAcquireTimeout,
		context.DeadlineExceeded,
	}
	const interval = 2
	const retries = 3
	return common.NewCommonRetryExecutor(stopCtx, interval*time.Second, retries, errs)
}
