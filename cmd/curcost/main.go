package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cursor-cost-monitor/internal/api"
	"cursor-cost-monitor/internal/config"
	"cursor-cost-monitor/internal/output"
	"cursor-cost-monitor/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flagToken   string
	flagAuthID  string
	flagCookie  string
	flagJSON    bool
	flagShort   bool
	flagWatch   bool
	flagRefresh int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "curcost",
		Short: "Cursor Cost & Usage Monitor CLI/TUI",
		Long: `curcost é uma ferramenta em Go para monitoramento de custos, cotas de requisição e ciclo de faturamento da sua conta do Cursor.
Detecta automaticamente sua sessão ativa no ambiente local ou aceita configuração personalizada.`,
		RunE: runRoot,
	}

	rootCmd.PersistentFlags().StringVarP(&flagToken, "token", "t", "", "Cursor access token (ou WorkosCursorSessionToken)")
	rootCmd.PersistentFlags().StringVar(&flagAuthID, "auth-id", "", "Cursor Stripe/WorkOS auth id (ex: google-oauth2|user_xxx)")
	rootCmd.PersistentFlags().StringVar(&flagCookie, "cookie", "", "WorkosCursorSessionToken cookie completo")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Exibir resultado em formato JSON")
	rootCmd.PersistentFlags().BoolVar(&flagShort, "short", false, "Exibir formato compacto em uma linha (ideal para tmux/prompt)")
	rootCmd.PersistentFlags().BoolVarP(&flagWatch, "watch", "w", false, "Iniciar modo interativo contínuo (TUI)")
	rootCmd.PersistentFlags().IntVarP(&flagRefresh, "refresh", "r", 30, "Intervalo de atualização em segundos para o modo watch")

	// Subcommands
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Inicia o dashboard interativo de terminal (TUI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			flagWatch = true
			return runRoot(cmd, args)
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Exibe o status atual de consumo e faturamento em modo one-shot",
		RunE: func(cmd *cobra.Command, args []string) error {
			flagWatch = false
			return runRoot(cmd, args)
		},
	}

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Gerenciamento de autenticação e credenciais",
	}

	authStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Mostra a origem e o status das credenciais detectadas",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := config.ResolveCredentials(flagToken, flagAuthID, flagCookie)
			if err != nil {
				return fmt.Errorf("nenhuma credencial válida encontrada: %w", err)
			}
			fmt.Println("🔑 Credenciais detectadas com sucesso:")
			fmt.Printf("  • Origem:     %s\n", creds.Source)
			if creds.Email != "" {
				fmt.Printf("  • Email:      %s\n", creds.Email)
			}
			if creds.AuthID != "" {
				fmt.Printf("  • Auth ID:    %s\n", creds.AuthID)
			}
			tokenPreview := creds.AccessToken
			if len(tokenPreview) > 20 {
				tokenPreview = tokenPreview[:10] + "..." + tokenPreview[len(tokenPreview)-8:]
			}
			fmt.Printf("  • Token:      %s\n", tokenPreview)
			return nil
		},
	}

	authLoginCmd := &cobra.Command{
		Use:   "set-token <token>",
		Short: "Salva manualmente um token de sessão no arquivo de configuração",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				cfg = &config.Config{}
			}
			cfg.Token = args[0]
			if flagAuthID != "" {
				cfg.AuthID = flagAuthID
			}
			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("falha ao salvar configuração: %w", err)
			}
			cfgPath, _ := config.ConfigFile()
			fmt.Printf("✅ Token salvo com sucesso em %s\n", cfgPath)
			return nil
		},
	}

	authCmd.AddCommand(authStatusCmd, authLoginCmd)
	rootCmd.AddCommand(watchCmd, statusCmd, authCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	creds, err := config.ResolveCredentials(flagToken, flagAuthID, flagCookie)
	if err != nil {
		return err
	}

	client := api.NewClient(creds.Cookie, creds.AccessToken)

	// Se modo watch/TUI
	if flagWatch {
		m := tui.NewModel(client, creds.Email, flagRefresh)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	// Modo one-shot
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dash, err := client.FetchDashboard(ctx, creds.Email)
	if err != nil {
		return fmt.Errorf("erro ao consultar métricas da conta: %w", err)
	}

	if flagJSON {
		jsonStr, err := output.RenderJSON(dash)
		if err != nil {
			return err
		}
		fmt.Println(jsonStr)
		return nil
	}

	if flagShort {
		fmt.Println(output.RenderShort(dash))
		return nil
	}

	fmt.Println(output.RenderOneShot(dash))
	return nil
}
