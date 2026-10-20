# Operación y conciliación

## Secuencia reproducible

```bash
npm ci
npm run ci
npm run build
./out/omendtl --list
./out/omendtl scenario settlement > report.json
npm run inspect -- report.json ousdc
./out/omendtl validate settlement
```

En Windows, sustituir el ejecutable por `./out/omendtl.exe`. El build escribe
en `out/`, que no se versiona. El inspector solo acepta un archivo JSON y un
ID de activo; informa con código distinto de cero si el contrato o las
cantidades son inválidos.

## Conciliación de una ejecución

1. Fijar revisión Git, escenario, versión de toolchains y `state_digest`.
2. Revisar `invariants` y relacionar cada evento con su entidad por ID.
3. Conciliar por activo reservas iniciales, movimientos entre vaults,
   withdrawals y comisiones finales; no mezclar decimales de activos distintos.
4. Comparar `reservations`, `settlements` y sus estados finales con el ciclo y
   el reloj lógico. Una solicitud en cola no cuenta como settlement finalizado.
5. Investigar diferencias de importes con reportes de la misma revisión antes
   de atribuirlas a cambios de política o de código.

No publicar reportes con datos operativos sensibles. Los escenarios incluidos
usan identificadores y saldos deterministas para pruebas; los archivos locales
generados no forman parte del control de versiones.
