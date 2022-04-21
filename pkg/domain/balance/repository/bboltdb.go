package repository

import (
	"github.com/lunemec/eve-accountant/pkg/domain/balance"
)

// TODO implement Repository
type boltDBRepository struct{}

func New(esiRepositories ...balance.Repository) *boltDBRepository {
	return nil
}
