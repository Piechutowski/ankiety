SELECT idgr, idbr, idpbr, etap, o, ow, oo, b, bw, bnw, bo, k, z,
       komentarz_zbr, komentarz_inst, data_przepisania_na_sp, rok_auweitr,
       data_testowania, data_przekazania_zbr, data_zwrotu_pbr,
       data_przekazania_inst, data_zwrotu_zbr, data_eksportu,
       data_importu, data_akceptacji, data_zamkniecia, data_przepisania_z_sk
FROM b_statusy
WHERE idbr = ?;