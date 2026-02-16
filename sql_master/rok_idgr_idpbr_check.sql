SELECT EXISTS(
  SELECT 1
  FROM gospodarstwa__lata gl
  JOIN gospodarstwa g ON g.idgr = gl.idgr
  WHERE gl.rok = ?
    AND gl.idgr = ?
    AND g.idpbr = ?
) AS result;