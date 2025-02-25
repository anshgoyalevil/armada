package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/armadaproject/armada/internal/common/compress"
)

type GetJobRunErrorRepository interface {
	GetJobRunError(ctx context.Context, runId string) (string, error)
}

type SqlGetJobRunErrorRepository struct {
	db           *pgxpool.Pool
	decompressor compress.Decompressor
}

func NewSqlGetJobRunErrorRepository(db *pgxpool.Pool, decompressor compress.Decompressor) *SqlGetJobRunErrorRepository {
	return &SqlGetJobRunErrorRepository{
		db:           db,
		decompressor: decompressor,
	}
}

func (r *SqlGetJobRunErrorRepository) GetJobRunError(ctx context.Context, runId string) (string, error) {
	var rawBytes []byte
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.RepeatableRead,
		AccessMode:     pgx.ReadOnly,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, "SELECT error FROM job_run WHERE run_id = $1 AND error IS NOT NULL", runId).Scan(&rawBytes)
		if err == pgx.ErrNoRows {
			return errors.Errorf("no error found for run with id %s", runId)
		}
		return err
	})
	if err != nil {
		return "", err
	}

	decompressed, err := r.decompressor.Decompress(rawBytes)
	if err != nil {
		log.WithError(err).Error("failed to decompress")
		return "", err
	}
	return string(decompressed), nil
}
