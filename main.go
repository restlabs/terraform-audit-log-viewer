package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

const terraformAPIURL = "https://app.terraform.io/api/v2/organization/audit-trail"

type AuditEvent struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Auth      Auth      `json:"auth"`
	Request   Request   `json:"request"`
	Resource  Resource  `json:"resource"`
}

type Auth struct {
	AccessorID       string `json:"accessor_id"`
	Description      string `json:"description"`
	Type             string `json:"type"`
	ImpersonatorID   string `json:"impersonator_id"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
}

type Request struct {
	ID string `json:"id"`
}

type Resource struct {
	ID     string      `json:"id"`
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Meta   interface{} `json:"meta"`
}

type ResponseData struct {
	Data []AuditEvent `json:"data"`
}

var rootCmd = &cobra.Command{
	Use:   "tfaudit",
	Short: "View terraform cloud audit events in your terminal",
	Run:   run,
}

var (
	orgToken string
	since    int
)

func main() {
	rootCmd.Flags().StringVarP(&orgToken, "token", "t", "", "Terraform Cloud organization token")
	rootCmd.Flags().IntVarP(&since, "since", "s", 1, "Audit events since (in number of days)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if orgToken == "" {
		fmt.Println("Error: organization token must be provided")
		os.Exit(1)
	}

	sinceTimestamp := time.Now().AddDate(0, 0, -since).Format(time.RFC3339)

	app := tview.NewApplication()
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	flex.AddItem(tview.NewTextView().SetText("Use arrow keys to scroll | Press ESC to exit").SetDynamicColors(true), 1, 1, false)

	fmt.Println("Fetching audit events...")
	responseData, err := fetchAuditEvents(orgToken, sinceTimestamp)
	if err != nil {
		fmt.Printf("Error fetching audit events: %v\n", err)
		os.Exit(1)
	}

	if len(responseData.Data) == 0 {
		fmt.Println("No audit events found")
		os.Exit(0)
	}

	lineNumber := 1
	for _, event := range responseData.Data {
		row := table.GetRowCount()
		table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", lineNumber)).SetTextColor(tcell.ColorGray))
		table.SetCell(row, 1, tview.NewTableCell(event.Timestamp.String()).SetTextColor(tcell.ColorDarkMagenta))
		table.SetCell(row, 2, tview.NewTableCell(event.Type).SetTextColor(tcell.ColorDarkCyan))
		table.SetCell(row, 3, tview.NewTableCell(event.ID).SetTextColor(tcell.ColorOlive))
		table.SetCell(row, 4, tview.NewTableCell(event.Auth.Description).SetTextColor(tcell.ColorTeal))
		table.SetCell(row, 5, tview.NewTableCell(event.Resource.Type).SetTextColor(tcell.ColorDarkBlue))
		table.SetCell(row, 6, tview.NewTableCell(event.Resource.Action).SetTextColor(tcell.ColorDarkGreen))
		lineNumber++
	}

	flex.AddItem(table, 0, 1, true)
	app.SetRoot(flex, true).SetFocus(table)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.Stop()
		}
		return event
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func fetchAuditEvents(token, since string) (*ResponseData, error) {
	requestURL, err := url.Parse(terraformAPIURL)
	if err != nil {
		return nil, err
	}

	query := requestURL.Query()
	query.Set("since", since)
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}
