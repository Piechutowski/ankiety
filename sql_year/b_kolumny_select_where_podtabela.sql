SELECT 
    b_kolumny.kolumna,
    b_kolumny.tytul,
    b_kolumny.symbol,
    b_kolumny.lp,
    b_kolumny.jm,
    b_kolumny.wymagana,
    b_kolumny.widoczna,
    b_kolumny.szerokosc,
    b_kolumny.min,
    b_kolumny.max,
    b_kolumny.slownik,
    b_jm.typ_jm,
    b_jm.format,
    b_slowniki.wartosc,
    b_slowniki.typ_slownika
FROM b_kolumny
LEFT JOIN b_jm 
    ON b_kolumny.jm = b_jm.jm
LEFT JOIN b_slowniki
    ON b_kolumny.slownik = b_slowniki.slownik
WHERE b_kolumny.podtabela = ?
ORDER BY b_kolumny.lp ASC;