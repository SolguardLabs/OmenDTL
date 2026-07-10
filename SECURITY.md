# Security Policy

## Modelo De Seguridad

OmenDTL modela un motor de preasignacion de liquidez para rutas de demanda. El
sistema asume que los assets, vaults, cuentas operativas y rutas se registran
en un ledger local determinista. Las operaciones criticas son forecast,
reserva, actualizacion de demanda, withdrawal operativo y settlement.

El motor usa cantidades enteras y evita dependencias externas en tiempo de
ejecucion. La CLI emite reportes JSON reproducibles para que auditores y
herramientas automaticas puedan comparar estados entre escenarios.

## Invariantes Esperadas

- Los balances de cuentas y vaults no deben ser negativos.
- Las rutas deben enlazar vaults existentes del mismo asset.
- Los forecasts deben enlazar una ruta y vaults consistentes.
- Las reservas deben enlazar forecast, ruta, owner y vaults existentes.
- Los settlements deben enlazar reservas y rutas existentes.
- Los withdrawals deben debitar el mismo asset del vault origen.
- Los ciclos deben referenciar forecasts del mismo ciclo.
- La contabilidad de forecast debe permanecer no negativa.

## Validaciones Automatizadas

El comando:

```bash
out/omendtl validate <scenario>
```

ejecuta el escenario indicado y comprueba las invariantes publicas del reporte.
La suite TypeScript valida escenarios normales de forecast, reservas,
actualizaciones de demanda, settlement y ventanas de liquidez.

## Gestion De Dependencias

El repositorio usa:

- Go modules para la toolchain Go;
- `package.json` para scripts Node y dependencias de desarrollo;
- Dependabot para `gomod`, `npm` y GitHub Actions.

## Alcance De Revision

Estan dentro de alcance:

- `src/`;
- `tests/`;
- `scripts/`;
- workflows de CI;
- documentacion publica.

Quedan fuera de alcance artefactos generados localmente como `out/`,
`node_modules/`, `.tools/` y cualquier archivo `.env`.

## Formato De Reporte Interno

Los reportes de revision deben incluir:

- escenario y comando usado;
- digest de estado;
- invariantes afectadas;
- impacto economico esperado;
- pasos de reproduccion;
- recomendacion tecnica y tests de regresion.

