# brag

Uma ferramenta de linha de comando para engenheiros de software registrarem conquistas, sincronizarem PRs/issues do GitHub e tickets do Jira, enriquecerem entradas com IA (formato STAR) e gerarem relatórios estruturados para avaliações de desempenho.

## Funcionalidades

- **Entradas manuais** via `brag add` com enriquecimento por IA (formato STAR, tags, pontuação de impacto)
- **Sincronização com GitHub** — PRs mergeados e issues fechadas por você
- **Sincronização com Jira** — tickets movidos para Done
- **Rastreamento de OKRs** — associe entradas a OKRs; a IA infere com alta confiança
- **StarTrail (trilha de carreira)** — contexto opcional de trilha de carreira para que o enriquecimento por IA reflita as competências do seu cargo
- **Relatórios de desempenho** — narrativa sintetizada por IA, agrupada por OKR, exportável em Markdown e PDF
- **Armazenamento em repositório GitHub privado** — entradas em JSON, sem banco de dados

## Instalação

### Homebrew (recomendado)

```sh
brew install eduardohitek/tap/brag
```

### Download manual (GitHub Releases)

1. Acesse a [página de releases](https://github.com/eduardohitek/brag-cli/releases/latest)
2. Baixe o arquivo correspondente ao seu sistema operacional e arquitetura
3. Extraia o binário e mova para um diretório no seu `$PATH`:

```sh
# Exemplo macOS (Apple Silicon)
tar -xzf brag_*_darwin_arm64.tar.gz
mv brag /usr/local/bin/
```

### Via Go

```sh
go install github.com/eduardohitek/brag@latest
```

### Compilar a partir do código-fonte

```sh
git clone https://github.com/eduardohitek/brag-cli
cd brag-cli
go build -o brag .
```

## Configuração inicial

```sh
brag init
```

O assistente interativo coleta:
- Token do GitHub + repositório para armazenamento (`owner/repo`)
- Token do GitHub + usuário para sincronização de PRs/issues
- URL base do Jira, e-mail e token de API (opcional)
- Chave de API da Anthropic
- Janela padrão de sincronização (dias)
- OKRs iniciais (opcional)

A configuração é salva em `~/.brag/config.yaml`.

### Arquivo de configuração (`~/.brag/config.yaml`)

```yaml
anthropic_api_key: "sk-ant-..."
openai_api_key: "sk-..."          # opcional, se usar OpenAI
ai_provider: "anthropic"          # anthropic (padrão) ou openai
ai_model: "claude-sonnet-4-20250514"  # opcional, sobrescreve o modelo padrão
storage:
  github_token: "ghp_..."
  repo: "usuario/brag-data"
github_sync:
  token: "ghp_..."
  username: "seuusuariogithub"
jira:
  base_url: "https://suaempresa.atlassian.net"
  email: "voce@empresa.com"
  api_token: "..."
sync:
  default_days: 30
okrs:
  - id: "OKR-2025-Q1-01"
    title: "Entregar o novo fluxo de onboarding até o fim do Q1"
    active: true
star_trail:
  file_path: "/home/voce/empresa/trilha-carreira.md"
  current_role: "Engenheiro Sênior"
  target_role: "Staff Engineer"   # opcional
```

## Uso

### Adicionar uma entrada manual

```sh
brag add "Corrigi um bug crítico de autenticação que afetava 20% dos usuários"
brag add "Liderança na migração para Kubernetes" --project minha-plataforma --okr OKR-2025-Q1-01
```

### Sincronizar GitHub + Jira

```sh
brag sync           # usa default_days do config
brag sync --days 14
```

### Sincronizar cache local com o repositório GitHub

```sh
brag sync-cache
```

Lê todos os arquivos de `~/.brag/cache/` e envia para o repositório GitHub as entradas que ainda não estão presentes. Útil quando o armazenamento no GitHub foi configurado após a criação de entradas locais.

### Listar entradas

```sh
brag list
brag list --period Q1-2025
brag list --tag reliability
brag list --source github
brag list --okr OKR-2025-Q1-01
brag list --no-okr
brag list --from 2025-01-01 --to 2025-03-31
```

### Enriquecer entradas com IA

```sh
brag enrich                              # enriquece entradas pendentes
brag enrich --period Q1-2025             # por período
brag enrich --from 2025-01-01 --to 2025-03-31
brag enrich --all                        # re-enriquece entradas já enriquecidas
brag enrich --provider openai --model gpt-4o
```

### Gerar relatório

```sh
brag report
brag report --period Q1-2025
brag report --from 2025-01-01 --to 2025-03-31
brag report --provider openai --model gpt-4o
```

O relatório é salvo em `reports/AAAA-MM-DD-report.md` no repositório GitHub de armazenamento.

### Exportar

```sh
brag export                  # PDF (padrão), relatório mais recente
brag export --format md      # cópia em Markdown
brag export --format pdf --report 2025-03-10-report.md
```

> A exportação para PDF requer o Google Chrome ou Chromium instalado.

### Gerenciar OKRs

```sh
brag okr list
brag okr add --id "OKR-2025-Q2-01" --title "Reduzir latência p95 para abaixo de 200ms"
brag okr deactivate OKR-2025-Q1-01
```

### StarTrail (trilha de carreira)

Aponte o `brag` para o documento de trilha de carreira da sua empresa para que o enriquecimento por IA e os relatórios reflitam as competências esperadas do seu cargo.

```sh
# Configurar (--target-role é opcional)
brag startrail set --file ~/empresa/trilha-carreira.md \
                   --current-role "Engenheiro Sênior" \
                   --target-role "Staff Engineer"

# Exibir a configuração ativa e uma prévia do documento
brag startrail show

# Remover a configuração (volta ao comportamento padrão de enriquecimento)
brag startrail clear
```

Quando configurado, cada enriquecimento do `brag add` e `brag sync` levará em conta como a atividade demonstra ou avança as competências esperadas do seu cargo atual (e do cargo alvo, se definido). A narrativa do `brag report` também mencionará o progresso nessas competências.

A configuração é armazenada em `~/.brag/config.yaml`:

```yaml
star_trail:
  file_path: "/home/voce/empresa/trilha-carreira.md"
  current_role: "Engenheiro Sênior"
  target_role: "Staff Engineer"   # opcional
```

Esta funcionalidade é completamente opcional — sem configuração, o comportamento é idêntico ao anterior.

## Armazenamento

As entradas são armazenadas como JSON em um repositório GitHub privado dentro de `entries/`:

```
entries/2025-03-10-143022-manual.json
entries/2025-03-10-153022-github.json
reports/2025-03-10-report.md
```

Um cache local também é mantido em `~/.brag/cache/`.

## Enriquecimento por IA

O `brag` suporta múltiplos provedores de IA para enriquecer entradas e gerar relatórios:

| Provedor | Padrão | Modelo padrão |
|---|---|---|
| **Anthropic** (padrão) | ✓ | `claude-sonnet-4-20250514` |
| **OpenAI** | — | `gpt-4o` |

**Ordem de resolução do provedor/modelo:**
1. Flag `--provider` / `--model` na linha de comando
2. Chaves `ai_provider` / `ai_model` no `~/.brag/config.yaml`
3. Padrão: Anthropic + `claude-sonnet-4-20250514`

O enriquecimento realiza:
- Converter notas brutas em declarações de conquista no formato STAR
- Atribuir tags: `confiabilidade`, `velocidade`, `liderança`, `mentoria`, `entrega`, `qualidade`, `impacto`
- Pontuar o impacto de 1 a 5
- Inferir associação com OKRs (apenas com alta confiança, nunca forçado)

## Obtendo tokens de API

### Anthropic API key
1. Acesse [console.anthropic.com](https://console.anthropic.com)
2. Vá em **API Keys** e crie uma nova chave
3. Adicione ao config como `anthropic_api_key`

### OpenAI API key
1. Acesse [platform.openai.com](https://platform.openai.com)
2. Vá em **API Keys** e crie uma nova chave
3. Adicione ao config como `openai_api_key`

### GitHub token (armazenamento)
1. Acesse **github.com → Settings → Developer settings → Personal access tokens → Fine-grained tokens**
2. Crie um token com acesso ao repositório de armazenamento
3. Permissão necessária: **Contents: Read & write**
4. Adicione ao config como `storage.github_token`

### GitHub token (sincronização)
1. Mesmo caminho: **Fine-grained tokens**
2. Permissões necessárias: **Contents: Read** e **Metadata: Read**
3. Adicione ao config como `github_sync.token`

### Jira API token
1. Acesse [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Crie um novo token de API
3. Adicione ao config como `jira.api_token`
