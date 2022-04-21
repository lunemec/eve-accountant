package balance

import (
	"time"

	"github.com/lunemec/eve-accountant/pkg/domain/balance/aggregate"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/pkg/errors"
)

type Service interface {
	Balance(from, to time.Time) (*aggregate.Balance, error)
}

type balanceService struct {
	repositories []Repository
}

func NewService(repositories ...Repository) *balanceService {
	return &balanceService{
		repositories: repositories,
	}
}

func (s *balanceService) Balance(from, to time.Time) (*aggregate.Balance, error) {
	totalBalance := aggregate.NewBalance()

	for _, repository := range s.repositories {
		balance, err := s.balanceForRepository(repository, from, to)
		if err != nil {
			return totalBalance, errors.Wrapf(err, "error loading balance for corporation: %d", repository.CorporationID())
		}
		totalBalance.Sum(balance)
	}
	return s.groupTypes(totalBalance), nil
}

func (s *balanceService) balanceForRepository(repository Repository, from, to time.Time) (*aggregate.Balance, error) {
	balance := aggregate.NewBalance()

	divisions, err := repository.WalletDivisions()
	if err != nil {
		return balance, errors.Wrapf(err, "error listing divisions for corporation: %d", repository.CorporationID())
	}
	for _, division := range divisions {
		journalRecords, err := repository.WalletJournal(division, from, to)
		if err != nil {
			return balance, errors.Wrap(err, "unable to list journal records")
		}

		for _, journalRecord := range journalRecords {
			balance.AmountByType[journalRecord.RefType] += journalRecord.Amount
			if journalRecord.Amount > 0 {
				balance.Income += journalRecord.Amount
			}
			if journalRecord.Amount < 0 {
				balance.Expenses += journalRecord.Amount
			}
		}
	}

	return balance, nil
}

var (
	marketTransactionType = entity.RefType("Market Transaction")
	contractPriceType     = entity.RefType("Contracts")
	playerWalletAction    = entity.RefType("Player Wallet Action")
	cloneTaxType          = entity.RefType("Clone Tax")
	industryTaxType       = entity.RefType("Industry Tax")
	piTaxType             = entity.RefType("PI Tax")
	rewardType            = entity.RefType("Krab Tax")
	jobCostType           = entity.RefType("Job Costs")
	feeType               = entity.RefType("Fee")

	refTypeGroups = map[entity.RefType]entity.RefType{
		entity.RefType("market_transaction"):                marketTransactionType,
		entity.RefType("contract_price"):                    contractPriceType,
		entity.RefType("player_donation"):                   playerWalletAction,
		entity.RefType("corporation_account_withdrawal"):    playerWalletAction,
		entity.RefType("jump_clone_activation_fee"):         cloneTaxType,
		entity.RefType("jump_clone_installation_fee"):       cloneTaxType,
		entity.RefType("industry_job_tax"):                  industryTaxType,
		entity.RefType("reprocessing_tax"):                  industryTaxType,
		entity.RefType("planetary_export_tax"):              piTaxType,
		entity.RefType("planetary_import_tax"):              piTaxType,
		entity.RefType("contract_deposit_refund"):           contractPriceType,
		entity.RefType("contract_auction_bid_refund"):       contractPriceType,
		entity.RefType("insurance"):                         rewardType,
		entity.RefType("bounty_prizes"):                     rewardType,
		entity.RefType("corporate_reward_payout"):           rewardType,
		entity.RefType("project_discovery_reward"):          rewardType,
		entity.RefType("agent_mission_reward"):              rewardType,
		entity.RefType("agent_mission_time_bonus_reward"):   rewardType,
		entity.RefType("ess_escrow_transfer"):               rewardType,
		entity.RefType("researching_technology"):            jobCostType,
		entity.RefType("researching_time_productivity"):     jobCostType,
		entity.RefType("copying"):                           jobCostType,
		entity.RefType("researching_material_productivity"): jobCostType,
		entity.RefType("reaction"):                          jobCostType,
		entity.RefType("manufacturing"):                     jobCostType,
		entity.RefType("alliance_maintainance_fee"):         feeType,
		entity.RefType("office_rental_fee"):                 feeType,
		entity.RefType("contract_sales_tax"):                contractPriceType,
		entity.RefType("contract_brokers_fee_corp"):         contractPriceType,
		entity.RefType("contract_auction_bid_corp"):         contractPriceType,
		entity.RefType("contract_deposit_corp"):             contractPriceType,
		entity.RefType("contract_reward_deposited_corp"):    contractPriceType,
		entity.RefType("contract_price_payment_corp"):       contractPriceType,
		entity.RefType("brokers_fee"):                       marketTransactionType,
		entity.RefType("transaction_tax"):                   marketTransactionType,
		entity.RefType("market_escrow"):                     marketTransactionType,
	}
)

func (s *balanceService) groupTypes(balance *aggregate.Balance) *aggregate.Balance {
	balanceOut := aggregate.NewBalance()
	balanceOut.Income = balance.Income
	balanceOut.Expenses = balance.Expenses

	for typeIn, amountIn := range balance.AmountByType {
		typeOut, ok := refTypeGroups[typeIn]
		if !ok {
			typeOut = typeIn
		}
		balanceOut.AmountByType[typeOut] += amountIn
	}

	return balanceOut
}
