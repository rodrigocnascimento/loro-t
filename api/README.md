# API contracts

Esta pasta separa os contratos por responsabilidade:

- `api/events/v1`: mensagens recebidas no daemon (input stream de integrações como Neovim/WezTerm).
- `api/state/v1`: snapshot agregado retornado pelo daemon e consumido por TUI/CLI.

## Compatibilidade e versionamento

Política aplicada aos dois domínios (`events` e `state`):

- **Minor additions sem break**: adicionar campos opcionais, novos tipos de evento (sem alterar existentes) e novos objetos opcionais em snapshot não quebra clientes existentes.
- **Major para breaking changes**: remoção/renomeação de campos, alteração de tipo de campo existente, mudança de semântica obrigatória ou de comportamento que invalida parsing atual exige nova major.

Exemplo de semver de payload:

- `1.0`, `1.1`, `1.2` = compatível na mesma major.
- `2.0` = quebra de compatibilidade com consumidores `1.x`.

## Eventos (`api/events/v1`)

Schema: `api/events/v1/events.schema.json`.

### Campos obrigatórios para **todos** os eventos

- `version`: versão do contrato do evento (ex.: `"1.0"`).
- `source`: origem do evento (ex.: `"nvim"`, `"wezterm"`).
- `timestamp`: instante RFC3339 (ex.: `"2026-04-28T12:00:00Z"`).
- `type`: discriminador do tipo de evento.

### Exemplo: `BufEnterEvent`

```json
{
  "version": "1.0",
  "source": "nvim",
  "timestamp": "2026-04-28T12:00:00Z",
  "type": "BufEnterEvent",
  "path": "/workspace/loro-t/src/main.rs",
  "buffer_id": 12,
  "project_root": "/workspace/loro-t"
}
```

### Exemplo: `ProjectSwitchEvent`

```json
{
  "version": "1.0",
  "source": "wezterm",
  "timestamp": "2026-04-28T12:00:05Z",
  "type": "ProjectSwitchEvent",
  "project_root": "/workspace/another-project",
  "workspace_name": "another-project"
}
```

## Snapshot (`api/state/v1`)

Schema: `api/state/v1/context_snapshot.schema.json`.

### Exemplo: `ContextSnapshot`

```json
{
  "version": "1.0",
  "timestamp": "2026-04-28T12:00:10Z",
  "active_project": {
    "root": "/workspace/loro-t",
    "name": "loro-t"
  },
  "active_buffer": {
    "path": "/workspace/loro-t/src/main.rs",
    "buffer_id": 12
  },
  "recent_events": [
    {
      "type": "BufEnterEvent",
      "source": "nvim",
      "timestamp": "2026-04-28T12:00:00Z",
      "summary": "Entered buffer src/main.rs"
    },
    {
      "type": "ProjectSwitchEvent",
      "source": "wezterm",
      "timestamp": "2026-04-28T12:00:05Z",
      "summary": "Switched project to /workspace/loro-t"
    }
  ]
}
```
