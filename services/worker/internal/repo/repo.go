package repo

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	User    UserRepository
	Session SessionRepository
}

func New(p *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:    NewUserRepository(p),
		Session: NewSessionRepository(p),
	}
}
