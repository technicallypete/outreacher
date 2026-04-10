package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	dbpkg "github.com/vitruviantech/outreacher/internal/db"
	dbgen "github.com/vitruviantech/outreacher/internal/db/gen"
	"github.com/vitruviantech/outreacher/internal/tenant"
	"github.com/vitruviantech/outreacher/internal/tools"
)

func newTestServer(t *testing.T) (*server.StdioServer, func()) {
	t.Helper()

	ctx := context.Background()
	pool, err := dbpkg.NewPool(ctx, "")
	if err != nil {
		t.Skipf("no DB available: %v", err)
	}

	q := dbgen.New(pool)

	ten, err := tenant.Bootstrap(ctx, q)
	if err != nil {
		pool.Close()
		t.Skipf("tenant bootstrap failed: %v", err)
	}

	s := server.NewMCPServer("outreacher", "0.1.0")
	tools.RegisterCampaignTools(s, q, ten.OrgID, ten.CampaignID)
	tools.Register(s, q, ten.CampaignID, tools.LLMConfig{})

	return server.NewStdioServer(s), func() { pool.Close() }
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runSession(t *testing.T, messages []string) []rpcResponse {
	t.Helper()

	srv, cleanup := newTestServer(t)
	defer cleanup()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		defer outW.Close()
		srv.Listen(ctx, inR, outW) //nolint:errcheck
	}()

	go func() {
		defer inW.Close()
		for _, msg := range messages {
			inW.Write([]byte(msg + "\n")) //nolint:errcheck
		}
	}()

	var responses []rpcResponse
	scanner := bufio.NewScanner(outR)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("unmarshal response: %v\nline: %s", err, line)
		}
		responses = append(responses, resp)
		if len(responses) == len(messages) {
			break
		}
	}
	return responses
}

func TestInitialize(t *testing.T) {
	responses := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
	})

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Error != nil {
		t.Fatalf("unexpected error: %s", responses[0].Error.Message)
	}

	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion = %q, want %q", result.ProtocolVersion, "2024-11-05")
	}
	if result.ServerInfo.Name != "outreacher" {
		t.Errorf("serverInfo.name = %q, want %q", result.ServerInfo.Name, "outreacher")
	}
}

const sampleCSV10 = `First Name,Last Name,Email,Email 2,Email 3,Phone,Phone 2,Phone 3,Location,Job Title,Industry,Company,Company URL,Website,Import Date,Intent,Profile URL,Total Score,Intent Keyword,Personnalized Email message,Personnalized LinkedIn message
Mark,Hewitt,,,,,,,"Greater Boston, United States",President & CEO,IT Services and IT Consulting,EQengineered,https://www.linkedin.com/company/eqengineered/,https://eqengineered.com,"Apr 04, 2026 12:12 PM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/mark,1.80,"""enterprise modernization""",,
Dan,Gray,,,,,,,"Cincinnati Metropolitan Area, United States","Vice President, CTO",IT Services and IT Consulting,DXC Technology,https://www.linkedin.com/company/dxctechnology/,https://dxc.com,"Apr 04, 2026 12:10 PM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/dan,1.80,"""enterprise modernization""",,
Nishant,Gautam,,,,,,,United States,Founder & CEO,IT Services and IT Consulting,Innovise IT,https://www.linkedin.com/company/innovise-it/,https://innovise-it.com,"Apr 04, 2026 2:11 AM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/nishant,2.00,"""enterprise transformation""",,
Jake,Echanove,,,,,,,"Phoenix, Arizona, United States","Senior Vice President, Global Presales",IT Services and IT Consulting,Lemongrass,https://www.linkedin.com/company/lemongrass-consulting-ltd/,https://lemongrassconsulting.com,"Apr 04, 2026 2:11 AM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/jake,1.80,"""enterprise transformation""",,
Andrew,Antos,,,,,,,"San Francisco, California, United States",Founder & CEO,Software Development,Klarity,https://www.linkedin.com/company/klarityai/,https://klarity.ai,"Apr 04, 2026 2:09 AM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/andrew,2.00,"""enterprise transformation""",,
Nischal,Nadhamuni,,,,,,,"San Francisco, California, United States",CTO & Founder,Software Development,Klarity,https://www.linkedin.com/company/klarityai/,https://klarity.ai,"Apr 04, 2026 2:08 AM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/nischal,2.10,"""enterprise transformation""",,
Jahmaal,Marshall,,,,,,,"Washington, District of Columbia, United States",Founder | CEO,Mental Health Care,"Listen Then Speak, LLC",https://www.linkedin.com/company/listen-then-speak-llc/,https://listenthenspeak.com,"Apr 04, 2026 2:08 AM",Just engaged with a <a href='https://example.com' target='_blank'>LinkedIn post</a>,https://www.linkedin.com/in/jahmaal,1.80,"""enterprise transformation""",,
Chris,Andrew,,,,,,,"Park City, Utah, United States",CEO / Co-Founder,Software Development,Scrunch,https://www.linkedin.com/company/scrunchai/,https://scrunch.excelsa.xyz,"Apr 03, 2026 7:23 PM",Just engaged with a <a href='https://example.com' target='_blank'>competitor</a>,https://www.linkedin.com/in/chris,1.80,https://www.linkedin.com/company/toptal/,,
Noah,Heck,,,,,,,"Raleigh-Durham-Chapel Hill Area, United States",Head of Product,Software Development,Tracker,https://www.linkedin.com/company/tracker-rms/,https://tracker-rms.com,"Apr 03, 2026 12:44 PM",Top 5% most active in your ICP (LinkedIn),https://www.linkedin.com/in/noah,2.00,,,
Lee,Christoff,,,,,,,"Greensboro Area, United States",Chief Financial Officer,IT Services and IT Consulting,Noregon Systems,https://www.linkedin.com/company/noregon-systems/,https://noregon.com,"Apr 03, 2026 12:44 PM",Top 5% most active in your ICP (LinkedIn),https://www.linkedin.com/in/lee,2.00,,,`

// TestImportLeads seeds the DB for subsequent tests via the MCP tool.
// It must run before TestSearchLeads_NewStatus and TestGetLead.
func TestImportLeads(t *testing.T) {
	csvArg, _ := json.Marshal(sampleCSV10)
	responses := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"import_leads","arguments":{"csv":` + string(csvArg) + `}}}`,
	})

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[1].Error != nil {
		t.Fatalf("unexpected error: %s", responses[1].Error.Message)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(responses[1].Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in response")
	}

	var summary struct {
		Companies int `json:"companies"`
		Signals   int `json:"signals"`
		Keywords  int `json:"keywords"`
		Leads     int `json:"leads"`
		Skipped   int `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &summary); err != nil {
		t.Fatalf("unmarshal summary: %v\ntext: %s", err, result.Content[0].Text)
	}
	// Import is idempotent — on a fresh DB we get new leads/companies,
	// on a re-run deduplication returns 0. Just verify the tool responded.
	t.Logf("import summary: companies=%d signals=%d keywords=%d leads=%d skipped=%d",
		summary.Companies, summary.Signals, summary.Keywords, summary.Leads, summary.Skipped)
}

func TestSearchLeads_NewStatus(t *testing.T) {
	responses := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_leads","arguments":{"status":"new"}}}`,
	})

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[1].Error != nil {
		t.Fatalf("unexpected error: %s", responses[1].Error.Message)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(responses[1].Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content, got none")
	}

	var leads []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &leads); err != nil {
		t.Fatalf("unmarshal leads: %v", err)
	}
	if len(leads) == 0 {
		t.Fatal("expected at least one lead with status=new")
	}
	for _, l := range leads {
		if l.Status != "new" {
			t.Errorf("lead %d has status=%q, want %q", l.ID, l.Status, "new")
		}
	}
}

// TestImportFromFile reads 10 random rows from the Gojiberry sample CSV (mounted
// at /imports at test time) and imports them via the MCP tool.
func TestImportFromFile(t *testing.T) {
	const csvPath = "/fixtures/test/gojiberry-selected-contacts.csv"
	f, err := os.Open(csvPath)
	if err != nil {
		t.Skipf("import file not mounted (%s): %v", csvPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	all, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(all) < 2 {
		t.Fatal("csv has no data rows")
	}

	header := all[0]
	data := all[1:]
	rand.Shuffle(len(data), func(i, j int) { data[i], data[j] = data[j], data[i] })
	n := 10
	if len(data) < n {
		n = len(data)
	}
	selected := append([][]string{header}, data[:n]...)

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.WriteAll(selected) //nolint:errcheck
	w.Flush()

	csvArg, _ := json.Marshal(buf.String())
	responses := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"import_leads","arguments":{"csv":` + string(csvArg) + `}}}`,
	})

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[1].Error != nil {
		t.Fatalf("unexpected error: %s", responses[1].Error.Message)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(responses[1].Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var summary struct {
		Companies int `json:"companies"`
		Signals   int `json:"signals"`
		Keywords  int `json:"keywords"`
		Leads     int `json:"leads"`
		Skipped   int `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	t.Logf("import summary: companies=%d signals=%d keywords=%d leads=%d skipped=%d",
		summary.Companies, summary.Signals, summary.Keywords, summary.Leads, summary.Skipped)

	if summary.Companies == 0 && summary.Leads == 0 {
		t.Error("expected at least companies or leads to be imported")
	}
}

// TestGetLead searches for any lead then fetches it by ID.
func TestGetLead(t *testing.T) {
	// First search to get a valid ID.
	searchResp := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_leads","arguments":{}}}`,
	})
	if len(searchResp) != 2 || searchResp[1].Error != nil {
		t.Fatalf("search_leads failed")
	}
	var searchResult struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(searchResp[1].Result, &searchResult); err != nil {
		t.Fatalf("unmarshal search result: %v", err)
	}
	var leads []struct{ ID int `json:"id"` }
	if err := json.Unmarshal([]byte(searchResult.Content[0].Text), &leads); err != nil || len(leads) == 0 {
		t.Fatal("no leads found to test get_lead")
	}

	id := leads[0].ID
	getResp := runSession(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_lead","arguments":{"id":%d}}}`, id),
	})

	if len(getResp) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(getResp))
	}
	if getResp[1].Error != nil {
		t.Fatalf("unexpected error: %s", getResp[1].Error.Message)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(getResp[1].Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) == 0 || result.Content[0].Text == "" {
		t.Fatal("expected lead data in response")
	}
}
