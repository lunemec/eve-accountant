package esi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	authService "github.com/lunemec/eve-bot-pkg/services/auth"
	"gopkg.in/tomb.v2"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type repository struct {
	authService authService.Service

	esi *goesi.APIClient

	characterID   entity.CharacterID
	corporationID entity.CorporationID
}

func New(log *zap.Logger, client *http.Client, authService authService.Service) (*repository, error) {
	v, err := authService.Verify()
	if err != nil {
		return nil, errors.Wrap(err, "token verify error")
	}

	esi := goesi.NewAPIClient(client, "EVE Accountant")
	r := &repository{
		authService: authService,
		esi:         esi,
	}

	characterInfo, _, err := esi.ESI.CharacterApi.GetCharactersCharacterId(r.ctx(context.Background()), v.CharacterID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get character public info")
	}
	log.Info("ESI Repository initialized", zap.Reflect("character", characterInfo))

	r.characterID = entity.CharacterID(v.CharacterID)
	r.corporationID = entity.CorporationID(characterInfo.CorporationId)
	return r, nil
}

func (r *repository) ctx(ctx context.Context) context.Context {
	return context.WithValue(ctx, goesi.ContextOAuth2, r.authService)
}

func (r *repository) CharacterID() entity.CharacterID {
	return r.characterID
}

func (r *repository) CorporationID() entity.CorporationID {
	return r.corporationID
}

func (r *repository) WalletDivisions(ctx context.Context) ([]aggregate.Division, error) {
	ctx = r.ctx(ctx)
	esiDivisions, _, err := r.esi.ESI.CorporationApi.GetCorporationsCorporationIdDivisions(
		ctx,
		int32(r.corporationID),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get corporation divisions")
	}
	divisions := make([]aggregate.Division, 0, len(esiDivisions.Wallet))
	for _, division := range esiDivisions.Wallet {
		divisions = append(divisions,
			aggregate.Division{
				ID:   entity.DivisionID(division.Division),
				Name: entity.DivisionName(division.Name),
			},
		)
	}
	return divisions, nil
}

func (r *repository) WalletJournal(ctx context.Context, division aggregate.Division, _, _ time.Time) (chan aggregate.JournalRecord, error) {
	ctx = r.ctx(ctx)
	t, ctx := tomb.WithContext(ctx)
	journals := make(chan aggregate.JournalRecord)

	t.Go(func() error {
		defer close(journals)

		journalPage, resp, err := r.esi.ESI.WalletApi.GetCorporationsCorporationIdWalletsDivisionJournal(
			ctx,
			int32(r.corporationID),
			int32(division.ID),
			nil,
		)
		if err != nil {
			return errors.Wrapf(err, "unable to get wallet journal for division: %s (%d)", division.Name, division.ID)
		}
		sendJournalPageToSliceAggregateJournalRecord(journalPage, journals)

		pages, err := strconv.Atoi(resp.Header.Get("X-Pages"))
		if err != nil {
			return errors.Wrap(err, "error converting X-Pages to integer")
		}
		// Fetch additional pages if any (starting page above is 1).
		for i := 2; i <= pages; i++ {
			journalPage, _, err := r.esi.ESI.WalletApi.GetCorporationsCorporationIdWalletsDivisionJournal(
				ctx,
				int32(r.corporationID),
				int32(division.ID),
				&esi.GetCorporationsCorporationIdWalletsDivisionJournalOpts{
					Page: optional.NewInt32(int32(i)),
				},
			)
			if err != nil {
				return errors.Wrapf(err, "unable to get wallet journal page: %d for division: %s (%d)", i, division.Name, division.ID)
			}
			sendJournalPageToSliceAggregateJournalRecord(journalPage, journals)
		}
		return nil
	})
	return journals, nil
}

func sendJournalPageToSliceAggregateJournalRecord(in []esi.GetCorporationsCorporationIdWalletsDivisionJournal200Ok, out chan aggregate.JournalRecord) {
	if len(in) == 0 {
		return
	}
	for _, inJournalRecord := range in {
		out <- mapWalletsDivisionJournalToAggregateJournalRecord(inJournalRecord)
	}
}

func mapWalletsDivisionJournalToAggregateJournalRecord(in esi.GetCorporationsCorporationIdWalletsDivisionJournal200Ok) aggregate.JournalRecord {
	return aggregate.JournalRecord{
		Amount:        entity.Amount(in.Amount),
		Balance:       entity.Balance(in.Balance),
		ContextId:     entity.ContextId(in.ContextId),
		ContextIdType: entity.ContextIdType(in.ContextIdType),
		Date:          in.Date,
		Description:   entity.Description(in.Description),
		FirstPartyId:  entity.FirstPartyId(in.FirstPartyId),
		Id:            entity.Id(in.Id),
		Reason:        entity.Reason(in.Reason),
		RefType:       entity.RefType(in.RefType),
		SecondPartyId: entity.SecondPartyId(in.SecondPartyId),
		Tax:           entity.Tax(in.Tax),
		TaxReceiverId: entity.TaxReceiverId(in.TaxReceiverId),
	}
}
