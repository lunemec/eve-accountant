package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"

	"github.com/asdine/storm/v3"
)

type persistentRepository struct {
	db            *storm.DB
	corpNode      storm.Node
	esiRepository balance.Repository
}

const (
	journalNodeKey = "journal"

	metadataKey = "metadata"
)

type Metadata struct {
	Metadata  string `storm:"id,unique"`
	UpdatedAt time.Time
}

func New(db *storm.DB, esiRepository balance.Repository) *persistentRepository {
	return &persistentRepository{
		db:            db,
		corpNode:      corporationNode(db, esiRepository.CorporationID()),
		esiRepository: esiRepository,
	}
}

func (r *persistentRepository) CharacterID() entity.CharacterID {
	return r.esiRepository.CharacterID()
}

func (r *persistentRepository) CorporationID() entity.CorporationID {
	return r.esiRepository.CorporationID()
}

func (r *persistentRepository) WalletDivisions(ctx context.Context) ([]aggregate.Division, error) {
	return r.esiRepository.WalletDivisions(ctx)
}

func (r *persistentRepository) WalletJournal(ctx context.Context, division aggregate.Division, from, to time.Time) (chan aggregate.JournalRecord, error) {
	divisionNode := r.divisionNode(division)
	journalNode := divisionNode.From(journalNodeKey)

	lastUpdatedAt, err := r.updatedAt(divisionNode)
	if err != nil {
		return nil, errors.Wrap(err, "error checking last update date for journal")
	}
	if time.Since(lastUpdatedAt) > 30*time.Minute {
		err = r.updateFromESI(ctx, journalNode, division)
		if err != nil {
			return nil, errors.Wrap(err, "error updating local DB from ESI")
		}
		err = r.recordUpdatedAt(divisionNode)
		if err != nil {
			return nil, errors.Wrap(err, "error saving current update date")
		}
	}

	journalsChan := make(chan aggregate.JournalRecord)
	t, _ := tomb.WithContext(ctx)
	t.Go(func() error {
		defer close(journalsChan)

		var journals []aggregate.JournalRecord
		err := journalNode.Range("Date", from, to, &journals)
		if err != nil {
			return errors.Wrap(err, "error fetching journals from DB")
		}
		for _, journal := range journals {
			journalsChan <- journal
		}
		return nil
	})

	return journalsChan, nil
}

func (r *persistentRepository) updatedAt(node storm.Node) (time.Time, error) {
	var metadata Metadata
	err := node.One("Metadata", metadataKey, &metadata)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			err = node.Save(&Metadata{Metadata: metadataKey})
			if err != nil {
				return metadata.UpdatedAt, errors.Wrap(err, "error saving metadata")
			}
			return metadata.UpdatedAt, nil
		}
		return metadata.UpdatedAt, errors.Wrap(err, "error loading metadata")
	}
	return metadata.UpdatedAt, nil
}

func (r *persistentRepository) recordUpdatedAt(node storm.Node) error {
	err := node.Save(&Metadata{Metadata: metadataKey, UpdatedAt: time.Now()})
	if err != nil {
		return errors.Wrap(err, "unable to update metadata")
	}
	return nil
}

func (r *persistentRepository) updateFromESI(ctx context.Context, node storm.Node, division aggregate.Division) error {
	esiJournals, err := r.esiRepository.WalletJournal(ctx, division, time.Time{}, time.Time{})
	if err != nil {
		return errors.Wrap(err, "error calling esi")
	}
	tx, err := node.Begin(true)
	if err != nil {
		return errors.Wrap(err, "unable to begin tx")
	}
	defer tx.Rollback()
	for esiJournal := range esiJournals {
		err = tx.Save(&esiJournal)
		if err != nil {
			return errors.Wrap(err, "error updating journal data")
		}
	}

	return errors.Wrap(tx.Commit(), "error commiting tx")
}

func corporationNode(db *storm.DB, id entity.CorporationID) storm.Node {
	return db.From(fmt.Sprintf("%d", id))
}

func (r *persistentRepository) divisionNode(division aggregate.Division) storm.Node {
	return r.corpNode.From(fmt.Sprint(division.ID))
}
