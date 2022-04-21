package balance

import (
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
)

type Repository interface {
	CharacterID() entity.CharacterID
	CorporationID() entity.CorporationID
	WalletDivisions() ([]aggregate.Division, error)
	WalletJournal(division aggregate.Division, from, to time.Time) ([]aggregate.JournalRecord, error)
}
