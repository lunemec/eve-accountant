package discord

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	quickchartgo "github.com/henomis/quickchart-go"
	"github.com/lunemec/eve-accountant/pkg/domain/balance/entity"
	"github.com/pkg/errors"
)

// iskGraphHandler will be called every time a new
// message is created on any channel that the autenticated bot has access to.
func (h *discordHandler) iskGraphHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// React before starting the balance calculation (it takes quite few seconds to fetch everything).
	err := h.discord.MessageReactionAdd(m.ChannelID, m.ID, `⏱️`)
	if err != nil {
		h.error(errors.Wrap(err, "error reacting with :stopwatch: emoji"), m.ChannelID)
	}

	dateStart, dateEnd, err := h.parseDateStartDateEnd(args)
	if err != nil {
		h.error(err, m.ChannelID)
		return
	}

	rawBalance, err := h.accountantSvc.BalanceByDayByDivisionByType(h.ctx, dateStart, dateEnd)
	if err != nil {
		h.error(errors.Wrap(err, "error calculating balance"), m.ChannelID)
		return
	}

	var (
		bil                = 1000000000.0
		days               []string
		allDivisions       = make(map[entity.DivisionName]struct{})
		allTypes           = make(map[entity.RefType]struct{})
		byDivisionBalances = make(map[string][]float64)
	)
	// Collect all divisions and types in this data period.
	for _, dayData := range rawBalance {
		for divisionName, byType := range dayData.Income {
			allDivisions[divisionName] = struct{}{}
			for refType := range byType {
				allTypes[refType] = struct{}{}
			}
		}
		for divisionName, byType := range dayData.Expenses {
			allDivisions[divisionName] = struct{}{}
			for refType := range byType {
				allTypes[refType] = struct{}{}
			}
		}
	}
	for _, dayData := range rawBalance {
		for divisionName := range allDivisions {
			var divisionBalance float64

			byTypeIncome, _ := dayData.Income[entity.DivisionName(divisionName)]
			for _, amount := range byTypeIncome {
				divisionBalance += float64(amount)
			}

			byTypeExpenses, _ := dayData.Expenses[entity.DivisionName(divisionName)]
			for _, amount := range byTypeExpenses {
				divisionBalance += float64(amount)
			}
			byDivisionBalances[string(divisionName)] = append(byDivisionBalances[string(divisionName)], divisionBalance/bil)
		}

		days = append(days, dayData.Timestamp.Format("2006-01-02"))
	}

	datasetConfig := `{
		"label": "%s",
		"data": %s,
		"fill": true,
		"spanGaps": false,
		"lineTension": 0,
		"pointRadius": 3,
		"pointHoverRadius": 3,
		"pointStyle": "circle",
		"borderDash": [
		0,
		0
		],
		"barPercentage": 0.9,
		"categoryPercentage": 0.8,
		"type": "line",
		"borderWidth": 3,
		"hidden": false
	}`

	datasetConfigs := []string{}
	for divisionName, divisionBalance := range byDivisionBalances {
		divisionBalanceB, err := json.Marshal(divisionBalance)
		if err != nil {
			h.error(errors.Wrap(err, "error encoding division income"), m.ChannelID)
			return
		}
		datasetConfigs = append(datasetConfigs, fmt.Sprintf(datasetConfig, divisionName+" Balance", string(divisionBalanceB)))
	}

	chartConfig := `{
		"type": "line",
		"data": {
			"datasets": [%s],
			"labels": %s
		},
		"options": {
			"title": {
			"display": false,
			"position": "top",
			"fontSize": 12,
			"fontFamily": "sans-serif",
			"fontColor": "#666666",
			"fontStyle": "bold",
			"padding": 10,
			"lineHeight": 1.2,
			"text": "Chart title"
			},
			"layout": {
			"padding": {
				"left": 0,
				"right": 0,
				"top": 0,
				"bottom": 0
			}
			},
			"legend": {
			"display": true,
			"position": "top",
			"align": "center",
			"fullWidth": true,
			"reverse": false,
			"labels": {
				"fontSize": 12,
				"fontFamily": "sans-serif",
				"fontColor": "#666666",
				"fontStyle": "normal",
				"padding": 10
			}
			},
			"scales": {
			"xAxes": [
				{
				"scaleLabel": {
					"display": true,
					"labelString": "Day",
					"lineHeight": 1.2,
					"fontColor": "#666666",
					"fontFamily": "sans-serif",
					"fontSize": 12,
					"fontStyle": "normal",
					"padding": 4
				},
				"id": "X1",
				"display": true,
				"position": "bottom",
				"type": "time",
				"stacked": false,
				"time": {
					"unit": "day",
					"stepSize": 1,
					"displayFormats": {
					"millisecond": "yyyy-MM-DD",
					"second": "yyyy-MM-DD",
					"minute": "yyyy-MM-DD",
					"hour": "yyyy-MM-DD",
					"day": "yyyy-MM-DD",
					"week": "yyyy-MM-DD",
					"month": "yyyy-MM-DD",
					"quarter": "yyyy-MM-DD",
					"year": "yyyy-MM-DD"
					}
				},
				"distribution": "linear",
				"gridLines": {
					"display": true,
					"color": "rgba(0, 0, 0, 0.1)",
					"borderDash": [
					0,
					0
					],
					"lineWidth": 1,
					"drawBorder": true,
					"drawOnChartArea": true,
					"drawTicks": true,
					"tickMarkLength": 10,
					"zeroLineWidth": 1,
					"zeroLineColor": "rgba(0, 0, 0, 0.25)",
					"zeroLineBorderDash": [
					0,
					0
					]
				},
				"angleLines": {
					"display": true,
					"color": "rgba(0, 0, 0, 0.1)",
					"borderDash": [
					0,
					0
					],
					"lineWidth": 1
				},
				"pointLabels": {
					"display": true,
					"fontColor": "#666",
					"fontSize": 10,
					"fontStyle": "normal"
				},
				"ticks": {
					"display": true,
					"fontSize": 12,
					"fontFamily": "sans-serif",
					"fontColor": "#666666",
					"fontStyle": "normal",
					"padding": 0,
					"stepSize": null,
					"minRotation": 0,
					"maxRotation": 50,
					"mirror": false,
					"reverse": false
				}
				}
			],
			"yAxes": [
				{
				"stacked": true,
				"scaleLabel": {
					"display": true,
					"labelString": "Balance",
					"lineHeight": 1.2,
					"fontColor": "#666666",
					"fontFamily": "sans-serif",
					"fontSize": 12,
					"fontStyle": "normal",
					"padding": 4
				},
				"id": "Y1",
				"display": true,
				"position": "left",
				"type": "linear",
				"time": {
					"unit": false,
					"stepSize": 1,
					"displayFormats": {
					"millisecond": "h:mm:ss.SSS a",
					"second": "h:mm:ss a",
					"minute": "h:mm a",
					"hour": "hA",
					"day": "MMM D",
					"week": "ll",
					"month": "MMM YYYY",
					"quarter": "[Q]Q - YYYY",
					"year": "YYYY"
					}
				},
				"distribution": "linear",
				"gridLines": {
					"display": true,
					"color": "rgba(0, 0, 0, 0.1)",
					"borderDash": [
					0,
					0
					],
					"lineWidth": 1,
					"drawBorder": true,
					"drawOnChartArea": true,
					"drawTicks": true,
					"tickMarkLength": 10,
					"zeroLineWidth": 1,
					"zeroLineColor": "rgba(0, 0, 0, 0.25)",
					"zeroLineBorderDash": [
					0,
					0
					]
				},
				"angleLines": {
					"display": true,
					"color": "rgba(0, 0, 0, 0.1)",
					"borderDash": [
					0,
					0
					],
					"lineWidth": 1
				},
				"pointLabels": {
					"display": true,
					"fontColor": "#666",
					"fontSize": 10,
					"fontStyle": "normal"
				},
				"ticks": {
					"display": true,
					"fontSize": 12,
					"fontFamily": "sans-serif",
					"fontColor": "#666666",
					"fontStyle": "normal",
					"padding": 0,
					"stepSize": 5,
					"minRotation": 0,
					"maxRotation": 50,
					"mirror": false,
					"reverse": false
				}
				}
			]
			},
			"plugins": {
			"datalabels": {
				"display": false,
				"align": "center",
				"anchor": "center",
				"backgroundColor": "#eee",
				"borderColor": "#ddd",
				"borderRadius": 6,
				"borderWidth": 1,
				"padding": 4,
				"color": "#666666",
				"font": {
				"family": "sans-serif",
				"size": 10,
				"style": "normal"
				}
			},
			"tickFormat": ""
			},
			"responsive": true,
			"tooltips": {
			"mode": "index"
			},
			"hover": {
			"mode": "index"
			},
			"cutoutPercentage": 50,
			"rotation": -1.5707963267948966,
			"circumference": 6.283185307179586,
			"startAngle": -1.5707963267948966
		}
	}`

	qc := quickchartgo.New()
	datesB, err := json.Marshal(days)
	if err != nil {
		h.error(errors.Wrap(err, "error encoding days for chart"), m.ChannelID)
		return
	}
	qc.Config = fmt.Sprintf(chartConfig, strings.Join(datasetConfigs, ","), string(datesB))
	qc.Width = 1920
	qc.Height = 1080
	qc.Version = "2.9.4"

	chartURL, err := qc.GetShortUrl()
	if err != nil {
		h.error(errors.Wrap(err, "error generating chart url"), m.ChannelID)
		return
	}

	for _, messages := range h.iskGraphMessages(dateStart, dateEnd, chartURL) {
		_, err = h.discord.ChannelMessageSendComplex(m.ChannelID, messages)
		if err != nil {
			h.error(errors.Wrap(err, "error sending balance message"), m.ChannelID)
			return
		}
	}
}

func (h *discordHandler) iskGraphMessages(
	dateStart, dateEnd time.Time,
	chartURL string,
) []*discordgo.MessageSend {
	title := fmt.Sprintf("%s %s", balanceMsg, titleWithDate(dateStart, dateEnd))
	var messages = []*discordgo.MessageSend{
		{
			Embed: &discordgo.MessageEmbed{
				Title: title,
				Color: 0xffffff,
				Image: &discordgo.MessageEmbedImage{
					URL: chartURL,
				},
			},
		},
	}

	return messages
}
