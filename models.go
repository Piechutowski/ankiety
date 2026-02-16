package main

import "database/sql"




type TerytSIMC struct {
	SIMC        string `db:"simc"`
	Miejscowosc string `db:"miejscowosc"`
	NRWPGR      string `db:"nrwpgr"`
}

type TerytTeryt struct {
	NRWPGR      string `db:"nrwpgr"`
	Wojewodztwo string `db:"wojewodztwo"`
	Powiat      string `db:"powiat"`
	Gmina       string `db:"gmina"`
	RodzajGminy string `db:"rodzaj_gminy"`
}

type PKDPKD struct {
	Kod  string         `db:"kod"`
	Opis sql.NullString `db:"opis"`
}

type Kody struct {
	Kod         string         `db:"kod"`
	KodSOC      string         `db:"kod_soc"`
	Tytul       string         `db:"tytul"`
	Opis        sql.NullString `db:"opis"`
	Uwagi       sql.NullString `db:"opis"`
	StawkaVATZO string         `db:"stawka_vat_zo"`
	StawkaVATRR string         `db:"stawka_vat_rr"`
}

type KodyWTabeli struct {
	KodyWTabli         string         `db:"kody_w_tabli"`
	KodyWTabli4Schemat string         `db:"kody_w_tabli4schemat"`
	Opis               sql.NullString `db:"opis"`
	Uwagi              sql.NullString `db:"opis"`
}

type TypyTabel struct {
	TypTabeli         string         `db:"typ_tabeli"`
	TypTabeli4Schemat string         `db:"typ_tabeli4schemat"`
	Opis              sql.NullString `db:"opis"`
	Uwagi             sql.NullString `db:"opis"`
}


type RodzajeTabel struct {
	RodzajTabeli         string         `db:"rodzaj_tabeli"`
	RodzajTabeli4Schemat string         `db:"rodzaj_tabeli4schemat"`
	Opis                 sql.NullString `db:"opis"`
	Uwagi                sql.NullString `db:"opis"`
}

type TypyJM struct {
	JM     string         `db:"jm"`
	Opis   sql.NullString `db:"opis"`
	TypJM  string         `db:"typ_jm"` // int, float, string
	Format string         `db:"format"`
	Uwagi  sql.NullString `db:"opis"`
}

type Slowniki struct {
	Slownik     string         `db:"slownik"`
	Opis        sql.NullString `db:"opis"`
	Uwagi       sql.NullString `db:"opis"`
	Wartosc     string         `db:"wartosc"`
	TypSlownika string         `db:"typ_slownika"`
}

type TypySlownikow struct {
	TypSlownika string         `db:"typ_slownika"`
	Opis        sql.NullString `db:"opis"`
	Uwagi       sql.NullString `db:"opis"`
}

// BiuraRachunkowe represents an accounting office (user group)
type BiuraRachunkowe struct {
	IDBR            string `db:"idbr"`
	Nazwa           string `db:"nazwa"`
	DataWylosowania string `db:"data_wylosowania"`
	DataNadania     string `db:"data_nadania"`
	Aktywne         int64  `db:"aktywne"`
}

// Uzytkownicy represents person working within the system
type Uzytkownicy struct {
	IDPBR           string         `db:"idpbr"`
	Login           string         `db:"login"`
	Password        string         `db:"password"`
	Salt            string         `db:"salt"`
	Imie            string         `db:"imie"`
	Nazwisko        string         `db:"nazwisko"`
	Email           string         `db:"email"`
	Rola            int64          `db:"rola"`
	Aktywny         int64          `db:"aktywny"`
	Zablokowany     int64          `db:"zablokowany"`
	DataWylosowania string         `db:"data_wylosowania"`
	DataNadania     string         `db:"data_nadania"`
	Opis            sql.NullString `db:"opis"`
	Uwagi           sql.NullString `db:"opis"`
	IDBR            string         `db:"idbr"`
}


type GospodarstwaLata struct {
	Rok  int64  `db:"rok"`
	IDGR string `db:"idgr"`
}