package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)


var ReEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type TabNode struct {
	TableName TableName
	Key       string
	Label     string
	Children  map[string]*TabNode
	Lp        uint8
	Access    UserType
}

// formularze children
var (
	tabFormularzeTabele = &TabNode{
		Lp:        10,
		Key:       "tabele",
		Label:     "Tabele",
		Access:    AccessAllUsers,
		TableName: "b_tabele",
	}
	tabFormularzeKolumny = &TabNode{
		Lp:        20,
		Key:       "kolumny",
		Label:     "Kolumny",
		Access:    AccessAllUsers,
		TableName: "b_kolumny",
	}
	tabFormularzeKody = &TabNode{
		Lp:        30,
		Key:       "kody",
		Label:     "Kody",
		Access:    AccessAllUsers,
		TableName: "b_kody",
	}
	tabFormularzeKody4Tabele = &TabNode{
		Lp:        40,
		Key:       "kody4tabele",
		Label:     "Kody dla Tabel",
		Access:    AccessAllUsers,
		TableName: "Kody4Tabele",
	}
	tabFormularzeBlokady = &TabNode{
		Lp:        50,
		Key:       "blokady",
		Label:     "Blokady",
		Access:    AccessAllUsers,
		TableName: "b_blokady",
	}
	tabFormularze = &TabNode{
		Lp:     10,
		Key:    "formularze",
		Label:  "Formularze",
		Access: AccessAllUsers,
		Children: map[string]*TabNode{
			"tabele":      tabFormularzeTabele,
			"kolumny":     tabFormularzeKolumny,
			"kody":        tabFormularzeKody,
			"kody4tabele": tabFormularzeKody4Tabele,
			"blokady":     tabFormularzeBlokady,
		},
	}
)

// slowniki children
var (
	tabSlownikiSlownikFomularzy = &TabNode{
		Lp:        10,
		Key:       "slownik_fomularzy",
		Label:     "Słowniki formularzy",
		Access:    AccessAllUsers,
		TableName: "SlownikiSlownikFomularzy",
	}
	tabSlownikiWartosciSlownikow = &TabNode{
		Lp:        20,
		Key:       "wartosci_slownikow",
		Label:     "Wartości słowników formularzy",
		Access:    AccessAllUsers,
		TableName: "SlownikiWartosciSlownikow",
	}
	tabSlowniki = &TabNode{
		Lp:     20,
		Key:    "slowniki",
		Label:  "Słowniki",
		Access: AccessAllUsers,
		Children: map[string]*TabNode{
			"slownik_fomularzy":  tabSlownikiSlownikFomularzy,
			"wartosci_slownikow": tabSlownikiWartosciSlownikow,
		},
	}
)

// algorytmy children
var (
	tabAlgorytmyMapaPol = &TabNode{
		Lp:        10,
		Key:       "mapa_pol",
		Label:     "Mapa pól",
		Access:    AccessAllUsers,
		TableName: "AlgorytmyMapaPol",
	}
	tabAlgorytmyAlgorytmy = &TabNode{
		Lp:        20,
		Key:       "algorytmy",
		Label:     "Algorytmy",
		Access:    AccessAllUsers,
		TableName: "AlgorytmyAlgorytmy",
	}
	tabAlgorytmyStale = &TabNode{
		Lp:        30,
		Key:       "stale",
		Label:     "Stałe",
		Access:    AccessAllUsers,
		TableName: "AlgorytmyStale",
	}
	tabAlgorytmyZakresy = &TabNode{
		Lp:        40,
		Key:       "zakresy",
		Label:     "Zakresy",
		Access:    AccessAllUsers,
		TableName: "AlgorytmyZakresy",
	}
	tabAlgorytmy = &TabNode{
		Lp:     30,
		Key:    "algorytmy",
		Label:  "Algorytmy",
		Access: AccessAllUsers,
		Children: map[string]*TabNode{
			"mapa_pol":  tabAlgorytmyMapaPol,
			"algorytmy": tabAlgorytmyAlgorytmy,
			"stale":     tabAlgorytmyStale,
			"zakresy":   tabAlgorytmyZakresy,
		},
	}
)

// ustawienia children
var (
	tabUstawieniaTestowanie = &TabNode{
		Lp:        10,
		Key:       "testowanie",
		Label:     "Testowanie",
		Access:    AccessAllUsers,
		TableName: "UstawieniaTestowanie",
	}
	tabUstawienia = &TabNode{
		Lp:     40,
		Key:    "ustawienia",
		Label:  "Ustawienia",
		Access: AccessAllUsers,
		Children: map[string]*TabNode{
			"testowanie": tabUstawieniaTestowanie,
		},
	}
)

// root
var TabsBDGRMetodyka = &TabNode{
	Key: "metodyka",
	Children: map[string]*TabNode{
		"formularze": tabFormularze,
		"slowniki":   tabSlowniki,
		"algorytmy":  tabAlgorytmy,
		"ustawienia": tabUstawienia,
	},
}


type StawkiVATZO struct {
	StawkaVATZO        string         `db:"stawka_vat_zo"`
	WartoscStawkiVATZO float64        `db:"wartosc_stawki_vat_zo"`
	Tytul              string         `db:"tytul"`
	Opis               sql.NullString `db:"opis"`
	Uwagi              sql.NullString `db:"opis"`
}

type StawkiVATRR struct {
	StawkaVATRR        string         `db:"stawka_vat_rr"`
	WartoscStawkiVATRR float64        `db:"wartosc_stawki_vat_rr"`
	Tytul              string         `db:"tytul"`
	Opis               sql.NullString `db:"opis"`
	Uwagi              sql.NullString `db:"opis"`
}

type UTGRWspolczynnikiSO struct {
	KodSOC  string `db:"kod_soc"`
	OpisSOC string `db:"opis_soc"`
}

type FRKody struct {
	TabelaKod string `db:"tabela_kod"`
	Nazwa     string `db:"nazwa"`
	Tabela    string `db:"tabela"`
	Kod       string `db:"kod"`
}


func (app *Application) TableSysBTabeleGet(year, endpoint string, yearDB YearDB) TableSchema {
	columnTabela := TableColumn{Name: "Tabela", Tooltip: "Darek wymysli", Width: 30, IsPK: true, DataType: "string"}
	columnTytul := TableColumn{Name: "Tytuł", Tooltip: "Darek wymysli", Width: 90, IsPK: false, DataType: "string"}
	columnLp := TableColumn{Name: "Lp", Tooltip: "Darek wymysli", Width: 30, IsPK: false, DataType: "int"}
	columnSymbol := TableColumn{Name: "Symbol", Tooltip: "Darek wymysli", Width: 80, IsPK: false, DataType: "string"}
	columnOpis := TableColumn{Name: "Opis", Tooltip: "Darek wymysli", Width: 80, IsPK: false, DataType: "string"}
	columnUwagi := TableColumn{Name: "Uwagi", Tooltip: "Darek wymysli", Width: 80, IsPK: false, DataType: "string"}

	tableSchema := TableSchema{
		Type:      SYSTEM_DEFINITON,
		TableName: "b_tabela",
		Year:      year,
		Columns:   []TableColumn{columnTabela, columnTytul, columnLp, columnSymbol, columnOpis, columnUwagi},
	}

	rows, err := app.DBManager.YQueryx(yearDB, "b_tabele_select_all")
	if err != nil {
		app.Logger.Error(err.Error())
		return tableSchema
	}
	defer rows.Close()

	var tableRows []TableRow
	for rows.Next() {
		var b BTabele
		if err := rows.Scan(&b); err != nil {
			app.Logger.Error("scan failed", "error", err)
			return TableSchema{}
		}
		tableRows = append(tableRows, TableRow{
			Cells: []TableCell{
				{Column: &columnTabela, Value: b.Tabela, Name: "tabela", Editable: 1},
				{Column: &columnTytul, Value: b.Tytul, Name: "tytul", Editable: 1},
				{Column: &columnLp, Value: strconv.FormatInt(b.LP, 10), Name: "lp", Editable: 1},
				{Column: &columnSymbol, Value: b.Symbol, Name: "symbol", Editable: 1},
				{Column: &columnOpis, Value: b.Opis.String, Name: "opis", Editable: 1},
				{Column: &columnUwagi, Value: b.Uwagi.String, Name: "uwagi", Editable: 1},
			},
		})
	}

	if err := rows.Err(); err != nil {
		app.Logger.Error(fmt.Sprintln("rows iteration failed:", err.Error()))
		return tableSchema
	}

	tableSchema.Rows = tableRows

	return tableSchema
}	

func (app *Application) MetodykaGet(w http.ResponseWriter, r *http.Request) {
	year := r.PathValue("year")
	path := r.PathValue("path")

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	var segments []string
	if path != "" {
		segments = strings.Split(path, "/")

		if len(segments) > 0 && segments[len(segments)-1] == "" {
			segments = segments[:len(segments)-1]
		}
	}

	tmplBaseData, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}

	tmplBaseData.Module = TmplModuleBDGR

	if !TabsBDGRMetodyka.HasAccessToPath(segments, tmplBaseData.User.Role) {
		app.Forbidden(w, r)
		return
	}

	baseUrl := fmt.Sprintf("/app/%s/bdgr/metodyka", year)
	tmplBaseData.TabRows = TabsBDGRMetodyka.TabRowsBuild(baseUrl, segments, tmplBaseData.User.Role)
	tableName := TabsBDGRMetodyka.TableNameGet(segments)

	tmplBaseData.Table = app.YearSystemTableCreate(tableName, year, r.URL.Path, YearDB(yearInt))
	app.Render(w, r, http.StatusOK, TMPL_GRID, tmplBaseData)
}

func (app *Application) YearSystemTableCreate(tableName, yearString, url string, yearDB YearDB) TableSchema {
	var tableSchema TableSchema
	switch tableName {
	case "b_tabele":
		tableSchema = app.TableSysBTabeleGet(yearString, url, yearDB)
	case "b_kody_w_tabeli":

	case "b_typy_tabel":

	case "b_rodzaje_tabel":

	case "b_podtabele":

	case "b_typy_jm":

	case "b_jm":

	case "b_typy_slownikow":

	case "b_slowniki":

	case "b_kolumny":

	case "b_stawki_vat_zo":

	case "b_stawki_vat_rr":

	case "utgr_wspolczynniki_so":

	case "fr_kody":

	case "b_kody":

	case "b_blokady":

	case "b_kody__podtabele":

	case "b_bdgrobmsp":

	case "b_statusy":

	case "b_etapy":

	case "pkd_pkd":

	case "teryt_teryt":

	case "teryt_simc":

	}

	return tableSchema
}

func (root *TabNode) TabRowsBuild(baseUrl string, segments []string, userType UserType) []TmplTabsRow {
	var rows []TmplTabsRow
	currentNode := root
	currentUrl := baseUrl

	// Build first row (root level)
	if len(currentNode.Children) > 0 {
		var items []TmplTabItem
		for key, child := range currentNode.Children {
			if child.Access&userType == 0 {
				continue
			}
			selected := len(segments) > 0 && segments[0] == key
			items = append(items, TmplTabItem{
				Label:      child.Label,
				URLSegment: key,
				Selected:   selected,
				Lp:         child.Lp, // capture order
			})
		}
		// Sort by Lp field
		sort.Slice(items, func(i, j int) bool {
			return items[i].Lp < items[j].Lp
		})
		if len(items) > 0 {
			rows = append(rows, TmplTabsRow{Items: items, BaseUrl: currentUrl})
		}
	}

	// Navigate through segments
	for i, segment := range segments {
		next, exists := currentNode.Children[segment]
		if !exists {
			break
		}
		currentNode = next
		currentUrl = currentUrl + "/" + segment

		// Build row for this level's children
		if len(currentNode.Children) > 0 {
			var items []TmplTabItem
			for key, child := range currentNode.Children {
				if child.Access&userType == 0 {
					continue
				}
				selected := i+1 < len(segments) && segments[i+1] == key
				items = append(items, TmplTabItem{
					Label:      child.Label,
					URLSegment: key,
					Selected:   selected,
					Lp:         child.Lp,
				})
			}
			// Sort by Lp field
			sort.Slice(items, func(i, j int) bool {
				return items[i].Lp < items[j].Lp
			})
			if len(items) > 0 {
				rows = append(rows, TmplTabsRow{Items: items, BaseUrl: currentUrl})
			}
		}
	}

	return rows
}

func (root *TabNode) HasAccessToPath(segments []string, userType UserType) bool {
	current := root

	for _, segment := range segments {
		if current.Children == nil {
			return false
		}

		next, exists := current.Children[segment]
		if !exists {
			return false
		}

		// Check access at this level
		if next.Access&userType == 0 {
			return false
		}

		current = next
	}

	return true
}

func (root *TabNode) TableNameGet(segments []string) string {
	current := root
	for _, segment := range segments {
		current, _ = current.Children[segment]
	}

	if current.TableName == "" {
		return ""
	}

	return string(current.TableName)
}



