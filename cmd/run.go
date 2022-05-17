package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	balanceDomain "github.com/lunemec/eve-accountant/pkg/domain/balance"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/repository"
	balanceDomainExternalRepository "github.com/lunemec/eve-accountant/pkg/domain/balance/repository/external/esi"
	discordHandler "github.com/lunemec/eve-accountant/pkg/handlers/discord"
	notifierHandler "github.com/lunemec/eve-accountant/pkg/handlers/notifier"
	accountantService "github.com/lunemec/eve-accountant/pkg/services/accountant"
	authRepository "github.com/lunemec/eve-bot-pkg/repositories/auth"
	authService "github.com/lunemec/eve-bot-pkg/services/auth"

	"github.com/asdine/storm/v3"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the discord bot",
	Run:   runBot,
}

var (
	checkInterval   time.Duration
	notifyInterval  time.Duration
	notifyThreshold float64

	discordChannelID string
	discordAuthToken string

	repositoryFile string
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringArrayVarP(&authfiles, "auth_files", "a", []string{"auth.bin"}, "paths to files where to read authentication data, for multiple corporations, login repeatedly with different file names")
	runCmd.Flags().StringVarP(&sessionKey, "session_key", "s", "", "session key, use random string")
	runCmd.Flags().StringVar(&eveClientID, "eve_client_id", "", "EVE APP client id")
	runCmd.Flags().StringVar(&eveSSOSecret, "eve_sso_secret", "", "EVE APP SSO secret")
	runCmd.Flags().StringVar(&discordChannelID, "discord_channel_id", "", "ID of discord channel")
	runCmd.Flags().StringVar(&discordAuthToken, "discord_auth_token", "", "Auth token for discord")
	runCmd.Flags().DurationVar(&checkInterval, "check_interval", 30*time.Minute, "how often to check EVE ESI API (default 30min)")
	runCmd.Flags().DurationVar(&notifyInterval, "notify_interval", 24*time.Hour, "how often to spam Discord (default 24H)")
	runCmd.Flags().Float64Var(&notifyThreshold, "notify_threshold", 1000000000, "balance under which to notify (default 1 000 000 000 ISK)")

	must(runCmd.MarkFlagRequired("session_key"))
	must(runCmd.MarkFlagRequired("eve_client_id"))
	must(runCmd.MarkFlagRequired("eve_sso_secret"))
	must(runCmd.MarkFlagRequired("discord_channel_id"))
	must(runCmd.MarkFlagRequired("discord_auth_token"))
	must(runCmd.MarkFlagRequired("auth_files"))
}

func runBot(cmd *cobra.Command, args []string) {
	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("error inicializing logger: %s \n", err)
		os.Exit(1)
	}
	err = runWrapper(log, cmd, args)
	if err != nil {
		log.Fatal("error running bot", zap.Error(err))
	}
}

func runWrapper(log *zap.Logger, cmd *cobra.Command, args []string) error {
	client := httpClient()

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	db, err := storm.Open("accountant.db")
	if err != nil {
		return errors.Wrap(err, "error openning DB")
	}
	defer db.Close()

	var (
		authServices    []authService.Service
		esiRepositories []balanceDomain.Repository
	)
	for _, authfile := range authfiles {
		authRepository := authRepository.NewFileRepository(authfile)
		authService := authService.NewService(
			log,
			client,
			authRepository,
			[]byte(sessionKey),
			eveClientID,
			eveSSOSecret,
			eveCallbackURL,
			eveScopes,
		)
		authServices = append(authServices, authService)
		esiRepository, err := balanceDomainExternalRepository.New(
			log,
			client,
			authService,
		)
		if err != nil {
			return errors.Wrapf(err, "error initializing ESI repository from: %s", authfile)
		}
		esiRepositories = append(esiRepositories, repository.New(db, esiRepository))
	}
	defer closeAuth(log, authServices)

	discord, err := discordgo.New("Bot " + discordAuthToken)
	if err != nil {
		return errors.Wrap(err, "error inicializing discord client")
	}
	err = discord.Open()
	if err != nil {
		return errors.Wrap(err, "unable to connect to discord")
	}
	var t tomb.Tomb

	balanceSvc := balanceDomain.NewService(esiRepositories...)
	accountantSvc := accountantService.New(balanceSvc, entity.Amount(notifyThreshold))
	discordHandler := discordHandler.New(
		t.Context(nil),
		log,
		discord,
		discordChannelID,
		accountantSvc,
	)
	notifierHandler := notifierHandler.New(
		t.Context(nil),
		log,
		checkInterval,
		notifyInterval,
		accountantSvc,
		discordHandler.MonthlyBalanceBelowThresholdMessage,
	)

	t.Go(func() error {
		discordHandler.Start()
		return nil
	})
	t.Go(func() error {
		notifierHandler.Start()
		return nil
	})

	select {
	case <-t.Dying():
	case <-signalChan:
		t.Kill(nil)
	}
	t.Wait()

	// systemd handles reload, so we can panic on error.
	err = t.Err()
	if err != nil {
		return errors.Wrapf(err, "error running bot: %+v", err)
	}

	return nil
}

func closeAuth(log *zap.Logger, authServices []authService.Service) {
	for _, authService := range authServices {
		_, err := authService.Token()
		if err != nil {
			log.Error("error refreshing and saving auth token", zap.Error(err))
		}
	}
}
