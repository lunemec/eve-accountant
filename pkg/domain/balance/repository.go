package balance

import (
	"context"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
)

type Repository interface {
	CharacterID() entity.CharacterID
	CorporationID() entity.CorporationID
	WalletDivisions(ctx context.Context) ([]aggregate.Division, error)
	WalletJournal(ctx context.Context, division aggregate.Division, from, to time.Time) (chan aggregate.JournalRecord, error)
}
