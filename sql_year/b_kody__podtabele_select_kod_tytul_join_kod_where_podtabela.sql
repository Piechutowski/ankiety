SELECT kody__podtabele.kod, kody.tytul
FROM b_kody__podtabele kody__podtabele
LEFT JOIN b_kody kody
ON kody__podtabele.kod = kody.kod 
WHERE kody__podtabele.podtabela = ?
ORDER BY kody__podtabele.lp;