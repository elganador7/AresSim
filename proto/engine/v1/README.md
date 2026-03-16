# Engine v1 Protos

These files define the main simulator model.

- `unit_definition.proto`: library and editor-facing platform definitions.
- `unit.proto`: live scenario units and runtime behavior.
- `scenario.proto`: scenario container types.
- `events.proto`: event stream emitted during sim execution.
- `weapon.proto`, `common.proto`, `status.proto`: shared supporting types.

Keep these schemas additive when possible. Breaking changes require regeneration and usually a DB schema update too.
