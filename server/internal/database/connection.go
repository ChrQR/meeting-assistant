package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	*Queries
	Conn *pgxpool.Pool
}

func NewConnection(ctx context.Context, dbURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Printf("Unable to initiate config: %v", err)
		return nil, err
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Printf("Failed to create database connection: %v", err)
		return nil, err
	}

	q := New(dbPool)

	return &DB{
		Queries: q,
		Conn:    dbPool,
	}, nil
}
