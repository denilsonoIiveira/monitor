package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cursor-cost-monitor/internal/api"
	"cursor-cost-monitor/internal/model"
	"cursor-cost-monitor/internal/output"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type tickMsg time.Time
type dataLoadedMsg struct {
	dashboard *model.AggregatedDashboard
	err       error
}

type Model struct {
	client          *api.Client
	userEmail       string
	refreshInterval time.Duration
	data            *model.AggregatedDashboard
	err             error
	loading         bool
	spinner         spinner.Model
	showTeam        bool
	width           int
	height          int
}

func NewModel(client *api.Client, userEmail string, refreshSec int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(BaseColor)

	interval := time.Duration(refreshSec) * time.Second
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}

	return Model{
		client:          client,
		userEmail:       userEmail,
		refreshInterval: interval,
		loading:         true,
		spinner:         s,
		showTeam:        true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchDataCmd(),
		m.tickCmd(),
	)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dash, err := m.client.FetchDashboard(ctx, m.userEmail)
		return dataLoadedMsg{
			dashboard: dash,
			err:       err,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchDataCmd()
		case "t":
			m.showTeam = !m.showTeam
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(m.fetchDataCmd(), m.tickCmd())

	case dataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.data = msg.dashboard
			m.err = nil
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	// Title Bar
	headerTitle := HeaderTitleStyle.Render(" CURSOR MONITOR ")
	statusBadge := StatusLiveStyle.Render("● LIVE")
	if m.loading {
		statusBadge = m.spinner.View() + " Atualizando..."
	}

	headerLine := fmt.Sprintf("%s  %s", headerTitle, statusBadge)
	if m.data != nil {
		headerLine += fmt.Sprintf("   %s", MetricTitleStyle.Render("Última checagem: "+m.data.FetchedAt.Format("15:04:05")))
	}
	sb.WriteString(headerLine + "\n\n")

	if m.err != nil {
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(RedColor).
			Padding(1, 2).
			Render(fmt.Sprintf("❌ Erro ao buscar dados: %v\nPressione [r] para tentar novamente.", m.err))
		sb.WriteString(errBox + "\n")
		return sb.String()
	}

	if m.data == nil {
		sb.WriteString(m.spinner.View() + " Carregando dados do Cursor...\n")
		return sb.String()
	}

	d := m.data

	// Row of Summary Cards
	// Card 1: Gasto Adicional
	spend := d.CurrentUserSpend.SpendUSD()
	limit := d.HardLimit
	if limit <= 0 {
		limit = 100.0
	}
	spendBar := output.RenderProgressBar(int(spend*100), int(limit*100), 20)
	c1Content := fmt.Sprintf("%s\n%s\n%s",
		MetricTitleStyle.Render("GASTO ADICIONAL (IA)"),
		MetricValueStyle.Foreground(RedColor).Render(fmt.Sprintf("US$ %.2f", spend)),
		spendBar+MetricTitleStyle.Render(fmt.Sprintf(" (teto US$ %.0f)", limit)),
	)
	c1 := CardStyle.Width(34).Render(c1Content)

	// Card 2: Franquia Inclusa
	incSpend := d.CurrentUserSpend.IncludedSpendUSD()
	var franquiaStatus string
	if d.CurrentUserSpend.TotalPercentUsed >= 100 {
		franquiaStatus = lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render("100% (Esgotada)")
	} else {
		franquiaStatus = lipgloss.NewStyle().Foreground(GreenColor).Bold(true).Render(fmt.Sprintf("%.1f%% Usada", d.CurrentUserSpend.TotalPercentUsed))
	}
	c2Content := fmt.Sprintf("%s\n%s\n%s",
		MetricTitleStyle.Render("FRANQUIA INCLUSA (PLANO)"),
		MetricValueStyle.Foreground(AccentColor).Render(fmt.Sprintf("US$ %.2f", incSpend)),
		MetricTitleStyle.Render("Status: ")+franquiaStatus,
	)
	c2 := CardStyle.Width(32).Render(c2Content)

	// Card 3: Billing Cycle & Account
	teamName := d.TeamName
	if teamName == "" {
		teamName = "Individual"
	}
	c3Content := fmt.Sprintf("%s\n%s\n%s",
		MetricTitleStyle.Render("CICLO DE FATURAMENTO"),
		MetricValueStyle.Render(fmt.Sprintf("%d dias restantes", d.DaysRemaining)),
		MetricTitleStyle.Render("Fim: ")+lipgloss.NewStyle().Foreground(YellowColor).Render(d.CycleEnd.Format("02/Jan/2006")),
	)
	c3 := CardStyle.Width(30).Render(c3Content)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, c1, c2, c3) + "\n\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	// Models & Credits Breakdown Table
	if d.AutoModelsPercentUsed > 0 || d.NamedModelsPercentUsed > 0 || d.TeamPooledUsedUSD > 0 {
		modelsTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(BaseColor)).
			Headers("CATEGORIA / CRÉDITOS", "CONSUMO / VALOR", "DETALHES DO CONSUMO").
			Row(
				lipgloss.NewStyle().Bold(true).Render("Modelos Nativos (Composer / Grok)"),
				lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render(fmt.Sprintf("%.0f%% da cota gasta", d.AutoModelsPercentUsed)),
				"composer-2.5, grok-4.5, vega (modelos integrados do Cursor)",
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("Modelos Nomeados (Claude / GPT)"),
				lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render(fmt.Sprintf("%.0f%% da cota gasta", d.NamedModelsPercentUsed)),
				"Claude 3.5 Sonnet, GPT-4o (modelos de terceiros da API)",
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("Franquia Base do Plano"),
				lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", d.IncludedPlanSpendUSD)),
				"Créditos mensais inclusos na assinatura do assento",
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("Bônus de Parceiros (OpenAI/Anthropic)"),
				lipgloss.NewStyle().Foreground(GreenColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", d.BonusSpendUSD)),
				"Créditos adicionais gratuitos concedidos no ciclo atual",
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("Gasto Total da Equipe (Pooled)"),
				lipgloss.NewStyle().Foreground(YellowColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", d.TeamPooledUsedUSD)),
				fmt.Sprintf("Gasto total somado da Verttice GR (Resta US$ %.2f do teto)", d.TeamPooledRemainingUSD),
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("  ↳ Seu Consumo Adicional"),
				lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", spend)),
				"Sua parcela faturável no gasto sob demanda da equipe",
			).
			Row(
				lipgloss.NewStyle().Bold(true).Render("  ↳ Demais Colegas da Equipe"),
				lipgloss.NewStyle().Foreground(YellowColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", d.TeamOthersSpendUSD)),
				"Parcela gasta pelos outros membros somados (ocultos individualmente)",
			)

		modelsTable.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(BaseColor).Render("🤖 Consumo por Tipo de Modelo & Composição dos Créditos:") + "\n")
		sb.WriteString(modelsTable.Render() + "\n\n")
	}

	// Per-Model Cost & Usage Table (Mês Vigente)
	if len(d.ModelStats) > 0 {
		modelDetailsTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(BaseColor)).
			Headers("MODELO", "REQUISIÇÕES", "% DO USO", "CUSTO ESTIMADO", "BLOCOS CÓDIGO", "ÚLTIMO USO")

		for _, ms := range d.ModelStats {
			lastStr := "-"
			if !ms.LastUsed.IsZero() {
				lastStr = ms.LastUsed.Format("02/01 15:04")
			}
			modelDetailsTable.Row(
				lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render(ms.Name),
				lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d reqs", ms.RequestsCount)),
				lipgloss.NewStyle().Foreground(MutedColor).Render(fmt.Sprintf("%.1f%%", ms.PercentOfUsage)),
				lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", ms.EstimatedCost)),
				lipgloss.NewStyle().Foreground(MutedColor).Render(fmt.Sprintf("%d blocos", ms.CodeBlocks)),
				lipgloss.NewStyle().Foreground(MutedColor).Render(lastStr),
			)
		}

		modelDetailsTable.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(BaseColor).Render("🧠 Detalhamento de Uso & Custo por Modelo (Mês Vigente):") + "\n")
		sb.WriteString(modelDetailsTable.Render() + "\n\n")
	}

	// Team Members Detail Table
	if m.showTeam && len(d.TeamMembers) > 0 {
		teamTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(BaseColor)).
			Headers("MEMBRO", "CARGO", "GASTO ADICIONAL", "% DA FRANQUIA USADA")

		for _, mem := range d.TeamMembers {
			name := mem.Name
			if name == "" {
				name = mem.Email
			}

			isCurrent := strings.EqualFold(mem.Email, d.UserEmail) || (d.CurrentUserSpend.UserID > 0 && mem.UserID == d.CurrentUserSpend.UserID)
			prefix := "   "
			nameStyle := lipgloss.NewStyle().Foreground(MutedColor)
			if isCurrent {
				prefix = "👉 "
				nameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
			}

			role := strings.TrimPrefix(mem.Role, "TEAM_ROLE_")
			var spendVal string
			var pctVal string

			if isCurrent {
				spendVal = lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render(fmt.Sprintf("US$ %.2f", mem.SpendUSD()))
				if mem.TotalPercentUsed >= 100 {
					pctVal = lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render("100% (Esgotada)")
				} else {
					pctVal = lipgloss.NewStyle().Foreground(MutedColor).Render(fmt.Sprintf("%.1f%%", mem.TotalPercentUsed))
				}
			} else if d.AdminOnlyUsagePricing {
				spendVal = lipgloss.NewStyle().Foreground(MutedColor).Render("🔒 Oculto (Admin)")
				pctVal = lipgloss.NewStyle().Foreground(MutedColor).Render("🔒 Oculto")
			} else {
				spendVal = lipgloss.NewStyle().Foreground(MutedColor).Render(fmt.Sprintf("US$ %.2f", mem.SpendUSD()))
				if mem.TotalPercentUsed >= 100 {
					pctVal = lipgloss.NewStyle().Foreground(RedColor).Bold(true).Render("100% (Esgotada)")
				} else {
					pctVal = lipgloss.NewStyle().Foreground(MutedColor).Render(fmt.Sprintf("%.1f%%", mem.TotalPercentUsed))
				}
			}

			teamTable.Row(
				nameStyle.Render(prefix+name),
				lipgloss.NewStyle().Foreground(MutedColor).Render(role),
				spendVal,
				pctVal,
			)
		}

		teamTable.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(BaseColor).Render("👥 Membros da Equipe:") + "\n")
		sb.WriteString(teamTable.Render() + "\n")

		if d.AdminOnlyUsagePricing {
			sb.WriteString("\n" + MetricTitleStyle.Render("🔒 Política de Privacidade: 'adminOnlyUsagePricing' ativo. Gastos de outros membros são visíveis apenas para Administradores.") + "\n")
		}
	}

	// Footer Help
	sb.WriteString("\n" + HelpStyle.Render(fmt.Sprintf(
		"Atalhos: [r] Atualizar agora • [t] Alternar tabela do time • [q] Sair • Auto-refresh: cada %ds",
		int(m.refreshInterval.Seconds()),
	)) + "\n")

	return sb.String()
}
