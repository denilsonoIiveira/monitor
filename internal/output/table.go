package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"cursor-cost-monitor/internal/model"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	titleStyle  = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	boldWhite   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
)

// RenderProgressBar renders an ASCII/Unicode progress bar
func RenderProgressBar(used, total int, width int) string {
	if total <= 0 {
		total = 1
	}
	ratio := float64(used) / float64(total)
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0.0 {
		ratio = 0.0
	}

	filledLen := int(ratio * float64(width))
	emptyLen := width - filledLen

	var barColor lipgloss.Style
	switch {
	case ratio >= 0.90:
		barColor = redStyle
	case ratio >= 0.70:
		barColor = yellowStyle
	default:
		barColor = greenStyle
	}

	filled := barColor.Render(strings.Repeat("█", filledLen))
	empty := subtleStyle.Render(strings.Repeat("░", emptyLen))
	pct := fmt.Sprintf(" %5.1f%%", float64(used)/float64(total)*100)

	return filled + empty + subtleStyle.Render(pct)
}

// RenderOneShot renders a complete, pedagogical and clear terminal report
func RenderOneShot(d *model.AggregatedDashboard) string {
	var sb strings.Builder

	header := titleStyle.Render("⚡ CURSOR COST & USAGE MONITOR")
	sb.WriteString(header + "\n\n")

	// Info section
	teamStr := d.TeamName
	if teamStr == "" {
		teamStr = "Individual"
	}
	membership := strings.ToUpper(d.MembershipType)
	if membership == "" {
		membership = "PRO"
	}

	userName := d.CurrentUserSpend.Name
	if userName == "" {
		userName = d.UserEmail
	}

	sb.WriteString(fmt.Sprintf("%s %s   %s %s   %s %s\n",
		subtleStyle.Render("Usuário:"), boldWhite.Render(userName+" ("+d.CurrentUserSpend.Email+")"),
		subtleStyle.Render("Plano:"), greenStyle.Render(membership),
		subtleStyle.Render("Equipe:"), accentStyle.Render(teamStr),
	))

	cycleStr := "Não disponível"
	if !d.CycleEnd.IsZero() {
		cycleStr = fmt.Sprintf("%s até %s (%s)",
			d.CycleStart.Format("02/Jan/2006"),
			d.CycleEnd.Format("02/Jan/2006"),
			yellowStyle.Render(fmt.Sprintf("Renova em %d dias", d.DaysRemaining)),
		)
	}
	sb.WriteString(fmt.Sprintf("%s %s\n\n", subtleStyle.Render("Ciclo de Faturamento:"), boldWhite.Render(cycleStr)))

	// Metrics Table
	currentSpendUSD := d.CurrentUserSpend.SpendUSD()
	includedSpendUSD := d.CurrentUserSpend.IncludedSpendUSD()
	hardLimitUSD := d.HardLimit
	if hardLimitUSD <= 0 {
		hardLimitUSD = 100.0
	}

	statusFranquia := "Em uso"
	if d.CurrentUserSpend.TotalPercentUsed >= 100.0 {
		statusFranquia = redStyle.Render("100% (Esgotada)")
	} else {
		statusFranquia = greenStyle.Render(fmt.Sprintf("%.1f%% utilizada", d.CurrentUserSpend.TotalPercentUsed))
	}

	pctTeto := (currentSpendUSD / hardLimitUSD) * 100.0
	tetoSignificado := fmt.Sprintf("Limite máximo da equipe (%.1f%% consumido)", pctTeto)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	metricsTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))).
		Headers("MÉTRICA / CAMPO", "VALOR ATUAL", "O QUE SIGNIFICA NA PRÁTICA?").
		Row(
			boldWhite.Render("Gasto Adicional (IA)"),
			redStyle.Render(fmt.Sprintf("US$ %.2f", currentSpendUSD)),
			"Valor faturável em requisições sob demanda após a franquia",
		).
		Row(
			boldWhite.Render("Franquia Inclusa"),
			cyanStyle.Render(fmt.Sprintf("US$ %.2f", includedSpendUSD)),
			"Cota base coberta pela assinatura do assento da equipe",
		).
		Row(
			boldWhite.Render("Status da Franquia"),
			statusFranquia,
			"Indica se a cota gratuita do mês foi totalmente consumida",
		).
		Row(
			boldWhite.Render("Teto da Equipe (Hard Limit)"),
			yellowStyle.Render(fmt.Sprintf("US$ %.2f", hardLimitUSD)),
			tetoSignificado,
		).
		Row(
			boldWhite.Render("Fast Requests (Cota Rápida)"),
			boldWhite.Render(fmt.Sprintf("%d / %d", d.FastRequestsUsed, d.FastRequestsLimit)),
			"Requisições rápidas de alta prioridade disponíveis no ciclo",
		).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	sb.WriteString(accentStyle.Render("📊 TABELA DE CUSTOS E CONSUMO (SEU USUÁRIO):") + "\n")
	sb.WriteString(metricsTable.Render() + "\n\n")

	// Models & Pools Table
	if d.AutoModelsPercentUsed > 0 || d.NamedModelsPercentUsed > 0 || d.TeamPooledUsedUSD > 0 {
		modelsTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))).
			Headers("CATEGORIA / CRÉDITOS", "CONSUMO / VALOR", "DETALHES DO CONSUMO").
			Row(
				boldWhite.Render("Modelos Nativos (Composer / Grok)"),
				redStyle.Render(fmt.Sprintf("%.0f%% da cota gasta", d.AutoModelsPercentUsed)),
				"composer-2.5, grok-4.5, vega (modelos integrados do Cursor)",
			).
			Row(
				boldWhite.Render("Modelos Nomeados (Claude / GPT)"),
				redStyle.Render(fmt.Sprintf("%.0f%% da cota gasta", d.NamedModelsPercentUsed)),
				"Claude 3.5 Sonnet, GPT-4o (modelos de terceiros da API)",
			).
			Row(
				boldWhite.Render("Franquia Base do Plano"),
				cyanStyle.Render(fmt.Sprintf("US$ %.2f", d.IncludedPlanSpendUSD)),
				"Créditos mensais inclusos na assinatura do assento",
			).
			Row(
				boldWhite.Render("Bônus de Parceiros (OpenAI/Anthropic)"),
				greenStyle.Render(fmt.Sprintf("US$ %.2f", d.BonusSpendUSD)),
				"Créditos adicionais gratuitos concedidos no ciclo atual",
			).
			Row(
				boldWhite.Render("Gasto Total da Equipe (Pooled)"),
				yellowStyle.Render(fmt.Sprintf("US$ %.2f", d.TeamPooledUsedUSD)),
				fmt.Sprintf("Gasto total somado da Verttice GR (Resta US$ %.2f do teto)", d.TeamPooledRemainingUSD),
			).
			Row(
				boldWhite.Render("  ↳ Seu Consumo Adicional"),
				redStyle.Render(fmt.Sprintf("US$ %.2f", currentSpendUSD)),
				"Sua parcela faturável no gasto sob demanda da equipe",
			).
			Row(
				boldWhite.Render("  ↳ Demais Colegas da Equipe"),
				yellowStyle.Render(fmt.Sprintf("US$ %.2f", d.TeamOthersSpendUSD)),
				"Parcela gasta pelos outros membros somados (ocultos individualmente)",
			).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return headerStyle
				}
				return cellStyle
			})

		sb.WriteString(accentStyle.Render("🤖 CONSUMO POR TIPO DE MODELO & COMPOSIÇÃO DOS CRÉDITOS:") + "\n")
		sb.WriteString(modelsTable.Render() + "\n\n")
	}

	// Per-Model Cost & Usage Table (Mês Vigente)
	if len(d.ModelStats) > 0 {
		modelDetailsTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))).
			Headers("MODELO", "REQUISIÇÕES", "% DO USO", "CUSTO ESTIMADO", "BLOCOS CÓDIGO", "ÚLTIMO USO")

		for _, ms := range d.ModelStats {
			lastStr := "-"
			if !ms.LastUsed.IsZero() {
				lastStr = ms.LastUsed.Format("02/01 15:04")
			}
			modelDetailsTable.Row(
				cyanStyle.Render(ms.Name),
				boldWhite.Render(fmt.Sprintf("%d reqs", ms.RequestsCount)),
				subtleStyle.Render(fmt.Sprintf("%.1f%%", ms.PercentOfUsage)),
				redStyle.Render(fmt.Sprintf("US$ %.2f", ms.EstimatedCost)),
				subtleStyle.Render(fmt.Sprintf("%d blocos", ms.CodeBlocks)),
				subtleStyle.Render(lastStr),
			)
		}

		modelDetailsTable.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

		sb.WriteString(accentStyle.Render("🧠 DETALHAMENTO DE USO & CUSTO POR MODELO (MÊS VIGENTE):") + "\n")
		sb.WriteString(modelDetailsTable.Render() + "\n\n")
	}

	// Team Members Table
	if len(d.TeamMembers) > 1 {
		sb.WriteString(accentStyle.Render("👥 COMPARATIVO COM MEMBROS DA EQUIPE:") + "\n")

		teamTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))).
			Headers("MEMBRO", "CARGO", "GASTO ADICIONAL", "% DA FRANQUIA USADA")

		for _, m := range d.TeamMembers {
			name := m.Name
			if name == "" {
				name = m.Email
			}

			isCurrent := strings.EqualFold(m.Email, d.UserEmail) || (d.CurrentUserSpend.UserID > 0 && m.UserID == d.CurrentUserSpend.UserID)
			prefix := "   "
			nameStyle := subtleStyle
			if isCurrent {
				prefix = "👉 "
				nameStyle = boldWhite
			}

			role := strings.TrimPrefix(m.Role, "TEAM_ROLE_")
			var spendStr string
			var pctStr string

			if isCurrent {
				spendStr = redStyle.Render(fmt.Sprintf("US$ %.2f", m.SpendUSD()))
				if m.TotalPercentUsed >= 100 {
					pctStr = redStyle.Render("100% (Esgotada)")
				} else {
					pctStr = subtleStyle.Render(fmt.Sprintf("%.1f%%", m.TotalPercentUsed))
				}
			} else if d.AdminOnlyUsagePricing {
				spendStr = subtleStyle.Render("🔒 Oculto (Admin)")
				pctStr = subtleStyle.Render("🔒 Oculto")
			} else {
				spendStr = subtleStyle.Render(fmt.Sprintf("US$ %.2f", m.SpendUSD()))
				if m.TotalPercentUsed >= 100 {
					pctStr = redStyle.Render("100% (Esgotada)")
				} else {
					pctStr = subtleStyle.Render(fmt.Sprintf("%.1f%%", m.TotalPercentUsed))
				}
			}

			teamTable.Row(
				nameStyle.Render(prefix+name),
				subtleStyle.Render(role),
				spendStr,
				pctStr,
			)
		}

		teamTable.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

		sb.WriteString(teamTable.Render() + "\n\n")
	}

	if d.AdminOnlyUsagePricing {
		sb.WriteString(yellowStyle.Render("🔒 Política de Privacidade da Equipe:") + " " + subtleStyle.Render("A Verttice GR ativou a opção 'adminOnlyUsagePricing'.") + "\n")
		sb.WriteString(subtleStyle.Render("   Por política do Cursor, o consumo dos outros membros é confidencial (visível apenas para admins).") + "\n\n")
	}

	// Bottom note
	sb.WriteString(subtleStyle.Render(fmt.Sprintf("💡 Nota: A cobrança de US$ %.2f é vinculada à fatura corporativa da Verttice GR.", currentSpendUSD)) + "\n")

	return boxStyle.Render(sb.String())
}

// RenderShort outputs a single-line summary ideal for tmux / prompt
func RenderShort(d *model.AggregatedDashboard) string {
	fastUsed := d.FastRequestsUsed
	fastLimit := d.FastRequestsLimit
	if fastLimit == 0 {
		fastLimit = 500
	}
	spend := d.CurrentUserSpend.SpendUSD()
	return fmt.Sprintf("⚡ %d/%d reqs | 💰 $%.2f", fastUsed, fastLimit, spend)
}

// RenderJSON outputs raw JSON
func RenderJSON(d *model.AggregatedDashboard) (string, error) {
	bytes, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
