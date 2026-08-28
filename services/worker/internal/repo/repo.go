package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	User        UserRepository
	Session     SessionRepository
	UserSession UserSessionRepository
	IngestJob   IngestJobRepository
}

func New(p *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:        NewUserRepository(p),
		Session:     NewSessionRepository(p),
		UserSession: NewUserSessionRepository(p),
		IngestJob:   NewIngestJobRepository(p),
	}
}
