package main

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"embed"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	html "html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form"
	"github.com/jmoiron/sqlx"
	"github.com/lmittmann/tint"
	
	// _ "modernc.org/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	gob.Register(User{})
}

//go:embed frontend/*
var FS_FRONTEND embed.FS

//go:embed sql_both/*.sql
var FS_SQL_BOTH embed.FS

//go:embed sql_master/*.sql
var FS_SQL_MASTER embed.FS

//go:embed sql_year/*.sql
var FS_SQL_YEAR embed.FS

func SqlPraseQueriesBoth(fsys embed.FS, name string) string {
	file, err := fsys.ReadFile("sql_both/" + name + ".sql")
	if err != nil {
		panic(err)
	}

	return string(file)
}

type SqlCache struct {
	DB      *sqlx.DB
	Queries map[string]*sqlx.Stmt
}

func CacheSqlQueriesFS(fsys embed.FS, dir string, db *sqlx.DB) *SqlCache {
	c := &SqlCache{DB: db, Queries: make(map[string]*sqlx.Stmt)}

	files, err := fsys.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		path := dir + "/" + file.Name()
		content, err := fsys.ReadFile(path)
		if err != nil {
			panic(err)
		}

		key := strings.TrimSuffix(file.Name(), ".sql")
		stmt, err := db.Preparex(string(content))
		if err != nil {
			panic(err)
		}

		c.Queries[key] = stmt
	}

	return c
}

func (c *SqlCache) stmt(name string) *sqlx.Stmt {
	stmt, ok := c.Queries[name]
	if !ok {
		panic("query not found: " + name)
	}
	return stmt
}

func (c *SqlCache) Queryx(name string, args ...any) (*sqlx.Rows, error) {
	return c.stmt(name).Queryx(args...)
}

func (c *SqlCache) QueryRowx(name string, args ...any) *sqlx.Row {
	return c.stmt(name).QueryRowx(args...)
}

func (c *SqlCache) Exec(name string, args ...any) (sql.Result, error) {
	return c.stmt(name).Exec(args...)
}

func (c *SqlCache) ExecFromString(query string, args ...any) (sql.Result, error) {
	return c.DB.Exec(query, args...)
}

var (
	sql_enable_fk = SqlPraseQueriesBoth(FS_SQL_BOTH, "enable_foreign_keys")
)

type YearDB int64

type DBManager struct {
	Logger       *slog.Logger
	MasterCache  *SqlCache
	yearCacheMap map[YearDB]*SqlCache
}

func (m *DBManager) MQueryx(queryName string, args ...any) (*sqlx.Rows, error) {
	return m.MasterCache.Queryx(queryName, args...)
}

func (m *DBManager) MQueryRowx(queryName string, args ...any) *sqlx.Row {
	return m.MasterCache.QueryRowx(queryName, args...)
}

func (m *DBManager) YQueryx(year YearDB, queryName string, args ...any) (*sqlx.Rows, error) {
	return m.yearCacheMap[year].Queryx(queryName, args...)
}

func (m *DBManager) YQueryRowx(year YearDB, queryName string, args ...any) *sqlx.Row {
	return m.yearCacheMap[year].QueryRowx(queryName, args...)
}

func (m *DBManager) YExec(year YearDB, queryName string, args ...any) (sql.Result, error) {
	return m.yearCacheMap[year].Exec(queryName, args...)
}

func (m *DBManager) YExecFromString(year YearDB, query string, args ...any) (sql.Result, error) {
	return m.yearCacheMap[year].DB.Exec(query, args...)
}

func (m *DBManager) Disconnect() {
	if err := m.MasterCache.DB.Close(); err != nil {
		m.Logger.Error(err.Error())
	}

	for _, sqlCache := range m.yearCacheMap {
		if err := sqlCache.DB.Close(); err != nil {
			m.Logger.Error(err.Error())
		}
	}
}

func (m *DBManager) Connect(dbDirPath string) {

	paths, err := filepath.Glob(dbDirPath + "*.db")
	if err != nil {
		panic(err)
	}

	for _, path := range paths {
		db, err := sqlx.Open("sqlite3", path)
		if err != nil {
			panic(err)
		}

		dbName := strings.TrimSuffix(filepath.Base(path), ".db")

		if dbName == "master" {
			m.MasterCache = CacheSqlQueriesFS(FS_SQL_MASTER, "sql_master", db)
			_, err := m.MasterCache.ExecFromString(sql_enable_fk)
			if err != nil {
				panic(err)
			}

			continue
		}

		value, err := strconv.Atoi(dbName)
		if err != nil {
			panic(err)
		}

		yearString := YearDB(value)

		m.yearCacheMap[yearString] = CacheSqlQueriesFS(FS_SQL_YEAR, "sql_year", db)
		_, err = m.YExecFromString(yearString, sql_enable_fk)
		if err != nil {
			panic(err)
		}
	}
}

var tmpl_funcs = html.FuncMap{
	"UserTypeName": func(ut UserType) string {
		switch ut {
		case UserAdmin:
			return "Administrator"
		case UserMethodolgist:
			return "Metodyk"
		case UserManager:
			return "Kierownik"
		case UserNormal:
			return "Pracownik"
		default:
			return "Nieznany"
		}
	},
	"HasAccess": func(userType, allowedTypes UserType) bool {
		return userType&allowedTypes != 0
	},
	"AdminOnly":          func() UserType { return AccessAdminOnly },
	"AdminMethodologist": func() UserType { return AccessAdminMethodologist },
	"AllUsers":           func() UserType { return AccessAllUsers },
}

func TmplCompse(template_names ...string) *html.Template {
	paths := []string{}
	for _, name := range template_names {
		paths = append(paths, "frontend/"+name+".html")
	}

	t := html.New("base").Funcs(tmpl_funcs)
	return html.Must(t.ParseFS(FS_FRONTEND, paths...))
}

var (
	TMPL_LOGIN       = TmplCompse("user_login")
	TMPL_APP         = TmplCompse("base", "main_choose_year", "nav_top")
	TMPL_APP_YEAR    = TmplCompse("base_year", "nav_top", "nav_left", "main_choose_module")
	TMPL_MOCK        = TmplCompse("base", "mock", "nav_top")
	TMPL_LIST_GR     = TmplCompse("base_year", "nav_top", "nav_left", "main_statusy")
	TMPL_GRID        = TmplCompse("base_year", "nav_top", "nav_left", "main_grid", "tables", "table_inputs")
	TMPL_DYNAMIC_ROW = TmplCompse("table_dynamic_row", "table_inputs")
)

type UserType uint8

func (u UserType) HasAccess(allowedTypes UserType) bool {
	return u&allowedTypes != 0
}

const (
	UserNormal UserType = 1 << iota
	UserManager
	UserMethodolgist
	UserAdmin
)

const (
	AccessAdminOnly          UserType = UserAdmin
	AccessAdminMethodologist UserType = UserAdmin | UserMethodolgist
	AcesssAdminManager       UserType = UserAdmin | UserManager
	AccessAllUsers           UserType = UserAdmin | UserMethodolgist | UserManager | UserNormal
)

type User struct {
	Login              string `db:"login"`
	Rola               string `db:"rola"`
	IdBR               string `db:"idbr"`
	IdPBR              string `db:"idpbr"`
	IdGR               string
	LastLogin          string
	LastPasswordChange string
	Role               UserType
}

type LoginForm struct {
	Login           string `form:"login" db:"login"`
	Password        string `form:"password" db:"password"`
	ValidationError bool   `form:"-"`
}

type Statusy struct {
	IDGR                string `db:"idgr"`
	IDBR                string `db:"idbr"`
	IDPBR               string `db:"idpbr"`
	Etap                string `db:"etap"`
	O                   sql.NullInt64  `db:"o"`
	OW                  sql.NullInt64  `db:"ow"`
	OO                  sql.NullInt64  `db:"oo"`
	B                   sql.NullInt64  `db:"b"`
	BW                  sql.NullInt64  `db:"bw"`
	BNW                 sql.NullInt64  `db:"bnw"`
	BO                  sql.NullInt64  `db:"bo"`
	K                   sql.NullInt64  `db:"k"`
	Z                   sql.NullInt64  `db:"z"`
	KomentarzZBR        sql.NullString `db:"komentarz_zbr"`
	KomentarzInst       sql.NullString `db:"komentarz_inst"`
	DataPrzepisaniaNaSP string `db:"data_przepisania_na_sp"`
	RokAuweitr          sql.NullInt64  `db:"rok_auweitr"`
	DataTestowania      sql.NullString `db:"data_testowania"`
	DataPrzekazaniaZBR  sql.NullString `db:"data_przekazania_zbr"`
	DataZwrotuPBR       sql.NullString `db:"data_zwrotu_pbr"`
	DataPrzekazaniaInst sql.NullString `db:"data_przekazania_inst"`
	DataZwrotuZBR       sql.NullString `db:"data_zwrotu_zbr"`
	DataEksportu        sql.NullString `db:"data_eksportu"`
	DataImportu         sql.NullString `db:"data_importu"`
	DataAkceptacji      sql.NullString `db:"data_akceptacji"`
	DataZamkniecia      sql.NullString `db:"data_zamkniecia"`
	DataPrzepisaniaZSK  sql.NullString `db:"data_przepisania_z_sk"`
}

type BTabele struct {
	Tabela string         `db:"tabela"`
	Tytul  string         `db:"tytul"`
	LP     int64          `db:"lp"`
	Symbol string         `db:"symbol"`
	Opis   sql.NullString `db:"opis"`
	Uwagi  sql.NullString `db:"uwagi"`
}

type BPodtabele struct {
	Subtable    string         `db:"podtabela"`
	Table       string         `db:"tabela"`
	TableKind   string         `db:"rodzaj_tabeli"`
	TableType   string         `db:"typ_tabeli"`
	TableCodes  string         `db:"kody_w_tabeli"`
	TableSchema string         `db:"schemat_tabeli"`
	Title       string         `db:"tytul"`
	Lp          int64          `db:"lp"`
	Symbol      string         `db:"symbol"`
	CarryOver   int64          `db:"czy_przepisac"`
	Description sql.NullString `db:"opis"`
	Remarks     sql.NullString `db:"uwagi"`
}

type BKolumny struct {
	Name            string         `db:"kolumna"` // INPUT NAME
	Title           string         `db:"tytul"`
	Label           string         `db:"symbol"`
	DataTypeLabel   string         `db:"jm"`
	DataType        string         `db:"typ_jm"`
	Format          string         `db:"format"`
	Required        int64          `db:"wymagana"`
	Visible         int64          `db:"widoczna"`
	Width           int64          `db:"szerokosc"`
	Formula         sql.NullString `db:"formula"`
	Regex           sql.NullString `db:"walidacja"`
	Min             sql.NullInt64  `db:"min"`
	Max             sql.NullInt64  `db:"max"`
	Lp              int64          `db:"lp"`
	Dictionary      sql.NullString `db:"slownik"`
	DictionaryValue sql.NullString `db:"wartosc"`
	DictionaryType  sql.NullString `db:"typ_slownika"`
	PrzepisacNa     string         `db:"przepisac_na"`
	Opis            sql.NullString `db:"opis"`
	Uwagi           sql.NullString `db:"uwagi"`
}

type BBlokady struct {
	Podtabela string         `db:"podtabela"`
	Column    string         `db:"kolumna"`
	Code      string         `db:"kod"`
	Opis      sql.NullString `db:"opis"`
	Uwagi     sql.NullString `db:"uwagi"`
}

type BKodyPodtabele struct {
	Code        string         `db:"kod"`
	Subtable    string         `db:"podtabela"`
	Title       string         `db:"tytul"`
	FRTableCode string         `db:"fr_tabela_kod"`
	Lp          sql.NullInt64  `db:"lp"`
	Description sql.NullString `db:"opis"`
	Remarks     sql.NullString `db:"uwagi"`
}

type BDGROBMSP struct {
	IDGR            string `db:"idgr"`
	Podtabela       string `db:"podtabela"`
	Dane            string `db:"dane"`
	DataModyfikacji string `db:"data_modyfikacji"`
}

// ============================================================================
// Administracja Tables
// ============================================================================

type Lata struct {
	Year        int64  `db:"rok"`
	Locked      int64  `db:"zablokowany"`
	Detached    int64  `db:"odlaczony"`
	Description string `db:"opis"`
	Remarks     string `db:"uwagi"`
}

type Role struct {
	Rola  string         `db:"rola"`
	Opis  sql.NullString `db:"opis"`
	Uwagi sql.NullString `db:"opis"`
}

// Gospodarstwa represents a farm, the primary subject of data collection
type Gospodarstwa struct {
	IDGR            string         `db:"idgr"`
	ID              string         `db:"id"`      // na zewnatrz
	IDTMPGR         string         `db:"idtmpgr"` // auweirt
	IDBR            string         `db:"idbr"`
	DataWylosowania string         `db:"data_wylosowania"`
	DataNadania     string         `db:"data_nadania"`
	Opis            sql.NullString `db:"opis"`
	Uwagi           sql.NullString `db:"opis"`
	IDPBR           string         `db:"idpbr"`
}

type TmplYears struct {
	Year   string
	Locked bool
}

type TableName string

type TmplTabsRow struct {
	Items   []TmplTabItem
	BaseUrl string
}

type TmplTabItem struct {
	Label      string
	URLSegment string
	Tooltip    string
	Lp         uint8
	Selected   bool	
}

type TmplBaseData struct {
	PageTitle   string
	Module      string
	CurrentYear *TmplYears
	Years       []TmplYears
	IdGR        string
	User        User
	TabRows     []TmplTabsRow
	Table       TableSchema
	Statusy     []Statusy
	BaseUrl     string
}

const (
	TmplModuleBDGR = "BDGRoBMSP"
)

type ColumnSlownik struct {
	Code []string `json:"Kod"`
	Opis []string `json:"Opis"`
}

func (c ColumnSlownik) ToSliceTableEnum() []TableEnum {
	var tableEnum []TableEnum
	for i := range c.Code {
		tableEnum = append(tableEnum, TableEnum{
			Value: c.Code[i],
			Label: c.Opis[i],
		})
	}
	return tableEnum
}

type TableEnum struct {
	Value string
	Label string
}

type TableCell struct {
	Value    string
	Name     string
	Column   *TableColumn
	Required int64
	Editable int64
	Blocked  bool
}

type TableRow struct {
	Cells []TableCell
	Title string
	Code  string
	Index int64
}

type TableColumn struct {
	Enum          []TableEnum
	Title         string // Required for vertical tables
	Name          string
	Label         string
	Tooltip       string
	DataTypeLabel string
	DataType      string
	Format        string
	Required      int64
	Visiable      int64
	Width         int64
	Formula       string
	Regex         string
	Min           *int64
	Max           *int64
	Lp            int64
	IsPK          bool
}

const (
	HORIZONTAL_DYNAMIC_DUPLICABLE = "HORIZONTAL_DYNAMIC_DUPLICABLE"
	HORIZONTAL_DYNAMIC_UNIQUE     = "HORIZONTAL_DYNAMIC_UNIQUE"
	HORIZONTAL_STATIC_UNIQUE      = "HORIZONTAL_STATIC_UNIQUE"
	PKD_STATIC_UNIQUE             = "PKD_STATIC_UNIQUE"
	SIMC_STATIC_UNIQUE            = "SIMC_STATIC_UNIQUE"
	VERTICAL_STATIC_UNIQUE        = "VERTICAL_STATIC_UNIQUE"
	SYSTEM_DEFINITON              = "SYSTEM_DEFINITION"
)

type TableSchema struct {
	Columns   []TableColumn
	Rows      []TableRow
	Type      string
	Year      string
	TableName string
	Table     string
	Subtable  string
	IdGR      string
	Data      string
}

type Constructor func(http.Handler) http.Handler

type Chain struct {
	Constructors []Constructor
}

func (c Chain) ThenFuncChain(middlewares ...ConstructorFunc) ConstructorFunc {
	return func(final http.HandlerFunc) http.HandlerFunc {
		h := final
		for _, fn := range middlewares {
			h = fn(h)
		}
		return h
	}
}

func ChainNew(constructors ...Constructor) Chain {
	return Chain{slices.Clone(constructors)}
}

func (c Chain) Then(h http.Handler) http.Handler {
	if h == nil {
		h = http.DefaultServeMux
	}

	for i := range c.Constructors {
		h = c.Constructors[len(c.Constructors)-1-i](h)
	}

	return h
}

func (c Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
	if fn == nil {
		return c.Then(nil)
	}
	return c.Then(fn)
}

func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, 0, len(c.Constructors)+len(constructors))
	newCons = append(newCons, c.Constructors...)
	newCons = append(newCons, constructors...)

	return Chain{newCons}
}

func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.Constructors...)
}

type ConstructorFunc func(http.HandlerFunc) http.HandlerFunc

type ChainFunc struct {
	constructors []ConstructorFunc
}

func ChainFuncNew(constructors ...ConstructorFunc) ChainFunc {
	return ChainFunc{slices.Clone(constructors)}
}

func (c ChainFunc) Then(fn http.HandlerFunc) http.HandlerFunc {
	if fn == nil {
		fn = http.DefaultServeMux.ServeHTTP
	}
	for i := len(c.constructors) - 1; i >= 0; i-- {
		fn = c.constructors[i](fn)
	}
	return fn
}

func (c ChainFunc) Append(constructors ...ConstructorFunc) ChainFunc {
	newCons := make([]ConstructorFunc, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)
	return ChainFunc{newCons}
}

func (c ChainFunc) Extend(chain ChainFunc) ChainFunc {
	return c.Append(chain.constructors...)
}

type Application struct {
	DBManager   *DBManager
	Logger      *slog.Logger
	FormDecoder *form.Decoder
	Session     *scs.SessionManager
	Debug       bool
}

// PathValueYearParse extracts and validates year from request path.
func (app *Application) PathValueYearParse(r *http.Request) (YearDB, error) {
	yearString := r.PathValue("year")
	year, err := strconv.Atoi(yearString)
	if err != nil {
		return 0, fmt.Errorf("invalid year parameter: %w", err)
}
	return YearDB(year), nil
}

// TabRowsTableBuild builds tab row with all tables, marking selectedTable as selected.
func (app *Application) TabRowsTableBuild(yearDB YearDB, selectedTable string) ([]TmplTabItem, error) {
	rows, err := app.DBManager.YQueryx(yearDB, "b_tabele_select_tabela_tytul")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TmplTabItem
	for rows.Next() {
		var tabLabel, tooltip string
		if err := rows.Scan(&tabLabel, &tooltip); err != nil {
			return nil, err
		}
		items = append(items, TmplTabItem{
			Label:      tabLabel,
			URLSegment: tabLabel,
			Tooltip: tooltip,
			Selected:   tabLabel == selectedTable,
		})
	}

	return items, rows.Err()
}

// TabRowsSubtableBuild builds tab row with subtables for given table.
func (app *Application) TabRowsSubtableBuild(yearDB YearDB, table, selectedSubtable string) ([]TmplTabItem, error) {
	rows, err := app.DBManager.YQueryx(yearDB, "b_tabele_select_podtabela_tytul_where_tabela", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TmplTabItem
	for rows.Next() {
		var subtableLabel, tytul string
		if err := rows.Scan(&subtableLabel, &tytul); err != nil {
			return nil, err
		}
		items = append(items, TmplTabItem{
			Label:      subtableLabel,
			URLSegment: table + "/" + subtableLabel,
			Tooltip: tytul,
			Selected:   subtableLabel == selectedSubtable,
		})
	}

	return items, rows.Err()
}

// ColumnsBuildFromKolumny converts database column definitions to TableColumn slice.
func ColumnsBuildFromKolumny(kolumny []BKolumny) []TableColumn {
	columns := make([]TableColumn, 0, len(kolumny))

	for _, k := range kolumny {
		column := TableColumn{
			Name:          k.Name,
			Title:         k.Title,
			Label:         k.Label,
			DataTypeLabel: k.DataTypeLabel,
			DataType:      k.DataType,
			Format:        k.Format,
			Required:      k.Required,
			Visiable:      k.Visible,
			Width:         k.Width,
			Lp:            k.Lp,
		}

		if k.Formula.Valid {
			column.Formula = k.Formula.String
		}

		if k.Regex.Valid {
			column.Regex = k.Regex.String
		}

		if k.Min.Valid {
			column.Min = &k.Min.Int64
		}

		if k.Max.Valid {
			column.Max = &k.Max.Int64
		}

		if k.DictionaryType.Valid {
			column.DataType = k.DictionaryType.String
			var columnSlownik ColumnSlownik
			json.Unmarshal([]byte(k.DictionaryValue.String), &columnSlownik)
			column.Enum = columnSlownik.ToSliceTableEnum()
		} else if k.Dictionary.Valid && k.Dictionary.String != "Kody" {
			column.DataType = "P"
			var columnSlownik ColumnSlownik
			json.Unmarshal([]byte(k.DictionaryValue.String), &columnSlownik)
			column.Enum = columnSlownik.ToSliceTableEnum()
		}

		columns = append(columns, column)
	}

	return columns
}

// KolumnySelectBySubtable fetches column definitions for a subtable.
func (app *Application) KolumnySelectBySubtable(yearDB YearDB, subtable string) ([]BKolumny, error) {
	rows, err := app.DBManager.YQueryx(yearDB, "b_kolumny_select_where_podtabela", subtable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kolumny []BKolumny
	if err := sqlx.StructScan(rows, &kolumny); err != nil {
		return nil, err
	}

	return kolumny, nil
}

// Add this method to fetch existing data
func (app *Application) DaneSelectByIdGRAndSubtable(yearDB YearDB, idGR, subtable string) (string, error) {
	row := app.DBManager.YQueryRowx(yearDB, "b_bdgrobmsp_dane_select_where_idgr_podtabela", idGR, subtable)

	var dane BDGROBMSP
	if err := row.StructScan(&dane); err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No data yet, that's fine
		}
		return "", err
	}
	return dane.Dane, nil
}

// Populate cells for horizontal tables (static or dynamic)
func PopulateCellsFromArray(rows []TableRow, jsonData string) error {
	if jsonData == "" {
		return nil
	}

	var dataArray []map[string]any
	if err := json.Unmarshal([]byte(jsonData), &dataArray); err != nil {
		return err
	}

	// Build lookup: code -> row data
	lookup := make(map[string]map[string]any)
	for _, item := range dataArray {
		// Find the _Kod field to use as key
		for k, v := range item {
			if strings.HasSuffix(k, "_Kod") {
				if code, ok := v.(string); ok {
					lookup[code] = item
				}
				break
			}
		}
	}

	// Populate cells
	for i := range rows {
		row := &rows[i]
		data, exists := lookup[row.Code]
		if !exists {
			continue
		}

		for j := range row.Cells {
			cell := &row.Cells[j]
			if val, ok := data[cell.Name]; ok {
				cell.Value = formatValue(val)
			}
		}
	}

	return nil
}

// Populate cells for vertical tables
func PopulateCellsFromObject(rows []TableRow, jsonData string) error {
	if jsonData == "" {
		return nil
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return err
	}

	for i := range rows {
		row := &rows[i]
		for j := range row.Cells {
			cell := &row.Cells[j]
			if val, ok := data[cell.Name]; ok {
				cell.Value = formatValue(val)
			}
		}
	}

	return nil
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// Check if it's actually an integer
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}

// BlokadySelectBySubtable fetches blocks for a subtable.
func (app *Application) BlokadySelectBySubtable(yearDB YearDB, subtable string) ([]BBlokady, error) {
	rows, err := app.DBManager.YQueryx(yearDB, "b_blokady_where_podtabela", subtable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []BBlokady
	if err := sqlx.StructScan(rows, &blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

// BlokadySelectBySubtableAndCode fetches blocks for a subtable and specific code.
func (app *Application) BlokadySelectBySubtableAndCode(yearDB YearDB, subtable, code string) ([]BBlokady, error) {
	rows, err := app.DBManager.YQueryx(yearDB, "b_blokady_where_podtabela_and_kod", subtable, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []BBlokady
	if err := sqlx.StructScan(rows, &blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (app *Application) TmplBaseDataUserDate(r *http.Request) (*TmplBaseData, error) {
	user, ok := app.Session.Get(r.Context(), "user").(User)
	if !ok {
		return nil, fmt.Errorf("user type mismatch")
	}

	tmplBaseData := &TmplBaseData{
		PageTitle: "Dashboard",
		User:      user,
	}

	rows, err := app.DBManager.MQueryx("lata_select_year_status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tmplYears []TmplYears
	for rows.Next() {
		var year Lata
		if err := rows.StructScan(&year); err != nil {
			return nil, err
		}

		if year.Detached == 1 {
			continue
		}

		tmplYears = append(tmplYears, TmplYears{
			Year:   strconv.FormatInt(year.Year, 10),
			Locked: year.Locked == 1,
		})
	}

	tmplBaseData.Years = tmplYears

	if currentYear := r.PathValue("year"); currentYear != "" {
		tmplBaseData.CurrentYear = &TmplYears{Year: currentYear, Locked: false}
	}
	
	if currentIdGR := r.PathValue("idgr"); currentIdGR != "" {
		tmplBaseData.IdGR = currentIdGR
	}

	return tmplBaseData, nil
}

func (app *Application) Render(w http.ResponseWriter, r *http.Request, status int, tmpl *html.Template, data any) {
	buf := new(bytes.Buffer)

	err := tmpl.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}

	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (app *Application) ClientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *Application) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	trace := string(debug.Stack())

	app.Logger.Error("internal error",
		slog.String("method", r.Method),
		slog.String("uri", r.URL.RequestURI()),
		slog.String("error", err.Error()),
		slog.String("trace", trace),
	)

	if app.Debug {
		fmt.Println("\nSTACK TRACE:\n" + err.Error() + "\n" + trace)
	}

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *Application) Forbidden(w http.ResponseWriter, r *http.Request) {
	app.Logger.Warn("forbidden access",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)
	http.Error(w, "403 Forbidden", http.StatusForbidden)
}

func (app *Application) MiddleLogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.Logger.Info("received request",
			slog.String("ip", r.RemoteAddr),
			slog.String("proto", r.Proto),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.RequestURI()),
		)
		next.ServeHTTP(w, r)
	})
}

func (app *Application) MiddleRecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if pv := recover(); pv != nil {
				w.Header().Set("Connection", "close")
				app.ServerError(w, r, fmt.Errorf("%v", pv))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *Application) MiddleLoged(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := app.Session.Get(r.Context(), "user").(User)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *Application) MiddleAccessIdGR(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		yearDB, err := app.PathValueYearParse(r)
		if err != nil {
			http.Redirect(w, r, "/app/", http.StatusSeeOther)
			return
		}

		idGR := r.PathValue("idgr")
		if idGR == "" {
			http.Redirect(w, r, "/app/", http.StatusSeeOther)
			return
		}

		user := app.Session.Get(r.Context(), "user").(User)
		if user.Role & UserAdmin != 0 {	
			next.ServeHTTP(w, r)
			return 
		}
		
		if user.Role & UserManager != 0 {	
			app.Logger.Error("Noo Accesss")
			var access int64
			row := app.DBManager.MQueryRowx("rok_idbr_check", int(yearDB), idGR, user.IdBR)	
			row.Scan(&access)
			if access == 1 {	
				next.ServeHTTP(w, r)
				return 
			}
		}
		
		var access int64
		row := app.DBManager.MQueryRowx("rok_idgr_idpbr_check", int(yearDB), idGR, user.IdPBR)	
		row.Scan(&access)
		if access == 1 {	
			next.ServeHTTP(w, r)
			return 
		}

		http.Redirect(w, r, "/app/", http.StatusSeeOther)
	})
}

func MiddlewareStaticHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        next.ServeHTTP(w, r)
    })
}

func MiddlewareMainHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "deny")
        w.Header().Set("X-XSS-Protection", "0")
        // TODO: Causes issue with style set as an attribute
        // w.Header().Set("Content-Security-Policy", "default-src 'self'") 
        w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
        w.Header().Set("Pragma", "no-cache")
        w.Header().Set("Expires", "0")
        next.ServeHTTP(w, r)
    })
}

func (app *Application) Routes() http.Handler {
	staticContent := http.NewServeMux()
	staticContent.Handle("GET  /frontend/", http.FileServer(http.FS(FS_FRONTEND)))
	staticContent.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		data, err := FS_FRONTEND.ReadFile("frontend/favicon.ico")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(data)
	})
	
	staticWrapped := ChainNew(MiddlewareStaticHeaders).Then(staticContent)
	
	Logged := ChainFuncNew(app.MiddleLoged)
	AccessIdGR := Logged.Append(app.MiddleAccessIdGR)

	main := http.NewServeMux()
	main.HandleFunc("GET  /{$}", app.LoginGet)
	main.HandleFunc("POST /login", app.LoginPost)
	main.HandleFunc("GET  /logout", app.LogoutGet)
	main.HandleFunc("GET  /app/", Logged.Then(app.AppGet))
	main.HandleFunc("GET  /app/{year}/", Logged.Then(app.YearGet))
	main.HandleFunc("GET  /app/{year}/bdgr/lista-ankiet/", Logged.Then(app.ListGRGet))
	main.HandleFunc("GET  /app/{year}/bdgr/lista-ankiet/{idgr}", AccessIdGR.Then(app.AnkietIdGRGet))
	main.HandleFunc("GET  /app/{year}/bdgr/lista-ankiet/{idgr}/{table}/", AccessIdGR.Then(app.AnkietTableGet))
	main.HandleFunc("GET  /app/{year}/bdgr/lista-ankiet/{idgr}/{table}/{subtable}/", AccessIdGR.Then(app.AnkietSubtableGet))
	main.HandleFunc("POST /app/{year}/bdgr/lista-ankiet/{idgr}/{table}/{subtable}/", AccessIdGR.Then(app.AnkietSubtablePost))
	main.HandleFunc("GET  /app/{year}/bdgr/lista-ankiet/{idgr}/{table}/{subtable}/{code}/{index}", AccessIdGR.Then(app.AnkietRowGet))
	// main.HandleFunc("GET  /app/{year}/bdgr/metodyka/{path...}", app.MiddleLoged(app.MetodykaGet))

	mainWrapped := ChainNew(
		app.MiddleRecoverPanic,
		app.Session.LoadAndSave,
		app.MiddleLogRequest,
		MiddlewareMainHeaders,
	).Then(main)
	
	root := http.NewServeMux()
	root.Handle("/frontend/", staticWrapped)
    root.Handle("/favicon.ico", staticWrapped)
    root.Handle("/", mainWrapped)
    
    return root
}

func (app *Application) LoginGet(w http.ResponseWriter, r *http.Request) {	
	_, ok := app.Session.Get(r.Context(), "user").(User)
	if ok {
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		return
	}
	
	if r.URL.Query().Get("login_error") == "1" {	
		app.Render(w, r, http.StatusOK, TMPL_LOGIN, LoginForm{ValidationError: true})
		return
	}
	
	app.Render(w, r, http.StatusOK, TMPL_LOGIN, nil)
}

func (app *Application) LoginPost(w http.ResponseWriter, r *http.Request) {		
	var loginForm LoginForm
	r.ParseForm()
	app.FormDecoder.Decode(&loginForm, r.PostForm)

	var userCreds LoginForm
	row := app.DBManager.MQueryRowx("login_password_get", loginForm.Login)
	if err := row.StructScan(&userCreds); err != nil {
		app.Logger.Error(err.Error())
		http.Redirect(w, r, "/?login_error=1", http.StatusSeeOther)
	}

	loginFormLower := strings.ToLower(loginForm.Login)
	userCredsLower := strings.ToLower(userCreds.Login)
	
	if loginFormLower != userCredsLower || loginForm.Password != userCreds.Password {
		http.Redirect(w, r, "/?login_error=1", http.StatusSeeOther)
		return
	}

	var userData User
	row = app.DBManager.MQueryRowx("user_data_get", loginForm.Login)
	if err := row.StructScan(&userData); err != nil {
		app.ServerError(w, r, err)
		return
	}

	switch userData.Rola {
	case "Adm":
		userData.Role = UserAdmin
	case "Met":
		userData.Role = UserMethodolgist
	case "ZBR":
		userData.Role = UserManager
	case "PBR":
		userData.Role = UserNormal
	default:
		app.ServerError(w, r, fmt.Errorf("unknown role: %s", userData.Rola))
		return
	}

	app.Session.Put(r.Context(), "user", userData)

	http.Redirect(w, r, "/app/", http.StatusSeeOther)
}

func (app *Application) LogoutGet(w http.ResponseWriter, r *http.Request) {
	if err := app.Session.Destroy(r.Context()); err != nil {
		app.ServerError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Application) AppGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}

	app.Render(w, r, http.StatusOK, TMPL_APP, data)
}

func (app *Application) YearGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		app.ServerError(w, r, err)
		return
	}

	app.Render(w, r, http.StatusOK, TMPL_APP_YEAR, data)
}

func (app *Application) AnkietListGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}
	data.Module = TmplModuleBDGR

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	tabItems, err := app.TabRowsTableBuild(yearDB, "")
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	data.TabRows = []TmplTabsRow{{Items: tabItems, BaseUrl: r.URL.Path}}

	app.Render(w, r, http.StatusOK, TMPL_GRID, data)
}

func (app *Application) ListGRGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.Logger.Error(err.Error())
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		return
	}
	data.Module = TmplModuleBDGR
	data.BaseUrl = r.URL.Path

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		return
	}


	if data.User.Role&UserMethodolgist != 0 {	
		app.Render(w, r, http.StatusOK, TMPL_LIST_GR, data)
	}
	
	var statusy []Statusy
	var rows *sqlx.Rows 
	if data.User.Role&UserAdmin != 0 {	 
		rows, err = app.DBManager.YQueryx(yearDB, "b_statusy_list_all")
	} else if data.User.Role&UserManager != 0 {		
		rows, err = app.DBManager.YQueryx(yearDB, "b_statusy_list_where_idbr", data.User.IdBR)
	} else {		
		rows, err = app.DBManager.YQueryx(yearDB, "b_statusy_list_where_idpbr", data.User.IdPBR)
	}
		
	if err != nil {
		app.Logger.Error(err.Error())
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		return
	}
	defer rows.Close()

	if err = sqlx.StructScan(rows, &statusy); err != nil {
		app.Logger.Error(err.Error())
		http.Redirect(w, r, "/app/", http.StatusSeeOther)
		return
	}

	data.Statusy = statusy

	app.Render(w, r, http.StatusOK, TMPL_LIST_GR, data)
}

func (app *Application) AnkietIdGRGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}
	data.Module = TmplModuleBDGR

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	tabItems, err := app.TabRowsTableBuild(yearDB, "")
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	data.TabRows = []TmplTabsRow{{Items: tabItems, BaseUrl: r.URL.Path}}

	app.Render(w, r, http.StatusOK, TMPL_GRID, data)
}

func (app *Application) AnkietTableGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}
	data.Module = TmplModuleBDGR

	selectedTable := r.PathValue("table")

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	tabItems, err := app.TabRowsTableBuild(yearDB, selectedTable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	subtabItems, err := app.TabRowsSubtableBuild(yearDB, selectedTable, "")
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	baseUrl := path.Dir(path.Dir(r.URL.Path))
	data.TabRows = []TmplTabsRow{
		{Items: tabItems, BaseUrl: baseUrl},
		{Items: subtabItems, BaseUrl: baseUrl},
	}

	app.Render(w, r, http.StatusOK, TMPL_GRID, data)
}

func (app *Application) AnkietSubtablePost(w http.ResponseWriter, r *http.Request) {
	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.jsonError(w, "Invalid year", http.StatusBadRequest)
		return
	}

	idGR := r.PathValue("idgr")
	subtable := r.PathValue("subtable")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.jsonError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if app.Debug {
		app.Logger.Debug("received JSON", slog.String("body", string(body)))
	}

	_, err = app.DBManager.YExec(yearDB, "b_bdgrobmsp_dane_replace", idGR, subtable, string(body))
	if err != nil {
		app.Logger.Error("failed to save data", slog.String("error", err.Error()))
		app.jsonError(w, "Failed to save data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func (app *Application) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"message": message,
	})
}

func (app *Application) AnkietSubtableGet(w http.ResponseWriter, r *http.Request) {
	data, err := app.TmplBaseDataUserDate(r)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}
	data.Module = TmplModuleBDGR

	selectedTable := r.PathValue("table")
	selectedSubtable := r.PathValue("subtable")
	yearString := r.PathValue("year")
	idGR := r.PathValue("idgr")

	data.Table.Table = selectedTable
	data.Table.Subtable = selectedSubtable
	data.Table.Year = yearString
	data.Table.IdGR = idGR

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	tabItems, err := app.TabRowsTableBuild(yearDB, selectedTable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	subtabItems, err := app.TabRowsSubtableBuild(yearDB, selectedTable, selectedSubtable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	baseUrl := path.Dir(path.Dir(path.Dir(r.URL.Path)))
	data.TabRows = []TmplTabsRow{
		{Items: tabItems, BaseUrl: baseUrl},
		{Items: subtabItems, BaseUrl: baseUrl},
	}

	row := app.DBManager.YQueryRowx(yearDB, "b_podtabeal_select_where_podtabela", selectedSubtable)
	var podtabelaGrid BPodtabele
	if err := row.StructScan(&podtabelaGrid); err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	data.Table.TableName = podtabelaGrid.Symbol + podtabelaGrid.Title
	data.Table.Type = podtabelaGrid.TableSchema

	kolumny, err := app.KolumnySelectBySubtable(yearDB, selectedSubtable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	data.Table.Columns = ColumnsBuildFromKolumny(kolumny)

	rows, err := app.DBManager.YQueryx(yearDB, "b_kody__podtabele_select_kod_tytul_join_kod_where_podtabela", selectedSubtable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}
	defer rows.Close()

	var kodyPodtabele []BKodyPodtabele
	if err := sqlx.StructScan(rows, &kodyPodtabele); err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	// Fetch existing data
	jsonData, err := app.DaneSelectByIdGRAndSubtable(yearDB, idGR, selectedSubtable)
	if err != nil {
		app.Logger.Warn("no existing data", slog.String("error", err.Error()))
	}

	switch data.Table.Type {
	case HORIZONTAL_DYNAMIC_DUPLICABLE, HORIZONTAL_DYNAMIC_UNIQUE:
		tableRows := make([]TableRow, 0, len(kodyPodtabele))
		for _, row := range kodyPodtabele {
			tableRows = append(tableRows, TableRow{Title: row.Title, Code: row.Code})
		}
		data.Table.Rows = tableRows
		data.Table.Data = jsonData

	case HORIZONTAL_STATIC_UNIQUE:
		blocks, err := app.BlokadySelectBySubtable(yearDB, selectedSubtable)
		if err != nil {
			app.ServerError(w, r, err)
			return
		}
		tableRows := make([]TableRow, 0, len(kodyPodtabele))
		for _, row := range kodyPodtabele {
			tableRow := TableRow{Title: row.Title, Code: row.Code} // Add Code here
			for i := range data.Table.Columns {
				column := &data.Table.Columns[i]
				cell := TableCell{
					Name:     column.Name,
					Column:   column,
					Required: column.Required,
					Editable: 1,
				}
				for _, block := range blocks {
					if block.Column == column.Name && block.Code == row.Code {
						cell.Blocked = true
						break
					}
				}
				if strings.Contains(cell.Name, "_Kod") {
					cell.Editable = 0
					cell.Value = row.Code
				}
				tableRow.Cells = append(tableRow.Cells, cell)
			}
			tableRows = append(tableRows, tableRow)
		}
		data.Table.Rows = tableRows

		// Populate with existing data
		if err := PopulateCellsFromArray(data.Table.Rows, jsonData); err != nil {
			app.Logger.Warn("failed to populate horizontal static data", slog.String("error", err.Error()))
		}

	case VERTICAL_STATIC_UNIQUE:
		for i := range data.Table.Columns {
			column := &data.Table.Columns[i]
			title := column.Label + " " + column.Title
			tableRow := TableRow{
				Title: title,
				Cells: []TableCell{{Column: column, Editable: 1, Name: column.Name}}, // Add Name here
			}
			data.Table.Rows = append(data.Table.Rows, tableRow)
		}

		// Populate with existing data
		if err := PopulateCellsFromObject(data.Table.Rows, jsonData); err != nil {
			app.Logger.Warn("failed to populate vertical static data", slog.String("error", err.Error()))
		}

	default:
		app.Logger.Error("not implemented table schema type", slog.String("type", data.Table.Type))
		return
	}

	app.Render(w, r, http.StatusOK, TMPL_GRID, data)
}

func (app *Application) AnkietRowGet(w http.ResponseWriter, r *http.Request) {
	subtable := r.PathValue("subtable")
	code := r.PathValue("code")

	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	yearDB, err := app.PathValueYearParse(r)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	kolumny, err := app.KolumnySelectBySubtable(yearDB, subtable)
	if err != nil {
		app.Logger.Error(err.Error())
		app.Forbidden(w, r)
		return
	}

	tableColumns := ColumnsBuildFromKolumny(kolumny)

	blocks, err := app.BlokadySelectBySubtableAndCode(yearDB, subtable, code)
	if err != nil {
		app.ServerError(w, r, err)
		return
	}

	tableRow := TableRow{Code: code, Index: int64(index)}
	for i := range tableColumns {
		column := &tableColumns[i]
		cell := TableCell{
			Name:     column.Name,
			Column:   column,
			Required: column.Required,
			Editable: 1,
		}
		
		for _, block := range blocks {
			if block.Column == column.Name {
				cell.Blocked = true
				break
			}
		}

		if strings.Contains(cell.Name, "_Kod") {
			cell.Editable = 0
			cell.Value = code
		}
	
		if strings.Contains(cell.Name, "_Wyszczegolnienie") {	
			row := app.DBManager.YQueryRowx(yearDB, "b_kody_tytul_where_kod", code)
			if err != nil {
				app.ServerError(w, r, err)
				return
			}
			
			var wyczegolnienie string
			row.Scan(&wyczegolnienie)			
			cell.Value = wyczegolnienie
		}
		
		tableRow.Cells = append(tableRow.Cells, cell)
	}

	w.Header().Set("Content-Type", "text/html")
	TMPL_DYNAMIC_ROW.Execute(w, tableRow)
}

func setupApplication(dbPath string) *Application {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	dbManager := &DBManager{
		Logger:       logger,
		yearCacheMap: make(map[YearDB]*SqlCache),
	}

	dbManager.Connect(dbPath)

	session := scs.New()
	session.IdleTimeout = 30 * time.Minute
	
	app := &Application{
		DBManager:   dbManager,
		Logger:      logger,
		FormDecoder: form.NewDecoder(),
		Session:     session,
		Debug:       true,
	}

	return app
}

func main() {
	addr := flag.String("addr", ":8082", "HTTP network address")
	dbDir := flag.String("db", "db/", "database directory")
	flag.Parse()

	app := setupApplication(*dbDir)
	defer app.DBManager.Disconnect()

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}

	server := &http.Server{
		Addr:         *addr,
		Handler:      app.Routes(),
		ErrorLog:     slog.NewLogLogger(app.Logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app.Logger.Info("starting server", slog.String("addr", *addr))
	err := server.ListenAndServe()
	app.Logger.Error(err.Error())
	os.Exit(1)
}
